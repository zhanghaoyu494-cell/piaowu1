from __future__ import annotations

import ctypes
import hashlib
import os
import re
import time
from dataclasses import dataclass
from pathlib import Path

from cryptography.hazmat.primitives.ciphers.aead import AESGCM


@dataclass(frozen=True, slots=True)
class RedactionResult:
    text: str
    sensitivity: str
    secret_detected: bool


SECRET_PATTERNS = (
    re.compile(
        r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----.*?-----END (?:RSA |EC |OPENSSH )?PRIVATE KEY-----",
        re.IGNORECASE | re.DOTALL,
    ),
    re.compile(r"\b(?:sk|pk)-[A-Za-z0-9_-]{20,}\b"),
    re.compile(r"\b(?:ghp|github_pat)_[A-Za-z0-9_]{20,}\b", re.IGNORECASE),
    re.compile(r"(?i)\bBearer\s+[A-Za-z0-9._~+/-]{16,}=*"),
    re.compile(
        r"(?i)\b(?:api[_-]?key|access[_-]?token|secret|password|passwd)\b\s*[:=]\s*[^\s,;]{6,}"
    ),
)

PII_PATTERNS = (
    (
        re.compile(r"\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b", re.IGNORECASE),
        "[REDACTED_EMAIL]",
    ),
    (re.compile(r"(?<!\d)1[3-9]\d{9}(?!\d)"), "[REDACTED_PHONE]"),
    (re.compile(r"(?<!\d)\d{17}[0-9Xx](?!\d)"), "[REDACTED_ID]"),
)


def redact_sensitive(text: str) -> RedactionResult:
    redacted = text
    secret_detected = False
    for pattern in SECRET_PATTERNS:
        redacted, count = pattern.subn("[REDACTED_SECRET]", redacted)
        secret_detected = secret_detected or count > 0

    pii_detected = False
    for pattern, replacement in PII_PATTERNS:
        redacted, count = pattern.subn(replacement, redacted)
        pii_detected = pii_detected or count > 0

    sensitivity = (
        "high" if secret_detected else "personal" if pii_detected else "normal"
    )
    return RedactionResult(redacted, sensitivity, secret_detected)


def content_hash(text: str) -> str:
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


class _DataBlob(ctypes.Structure):
    _fields_ = [("cbData", ctypes.c_ulong), ("pbData", ctypes.POINTER(ctypes.c_ubyte))]


def _to_blob(data: bytes) -> tuple[_DataBlob, ctypes.Array[ctypes.c_char]]:
    buffer = ctypes.create_string_buffer(data)
    blob = _DataBlob(len(data), ctypes.cast(buffer, ctypes.POINTER(ctypes.c_ubyte)))
    return blob, buffer


def _dpapi_protect(data: bytes) -> bytes:
    in_blob, buffer = _to_blob(data)
    out_blob = _DataBlob()
    if not ctypes.windll.crypt32.CryptProtectData(
        ctypes.byref(in_blob), None, None, None, None, 0, ctypes.byref(out_blob)
    ):
        raise ctypes.WinError()
    del buffer
    try:
        return ctypes.string_at(out_blob.pbData, out_blob.cbData)
    finally:
        ctypes.windll.kernel32.LocalFree(out_blob.pbData)


def _dpapi_unprotect(data: bytes) -> bytes:
    in_blob, buffer = _to_blob(data)
    out_blob = _DataBlob()
    if not ctypes.windll.crypt32.CryptUnprotectData(
        ctypes.byref(in_blob), None, None, None, None, 0, ctypes.byref(out_blob)
    ):
        raise ctypes.WinError()
    del buffer
    try:
        return ctypes.string_at(out_blob.pbData, out_blob.cbData)
    finally:
        ctypes.windll.kernel32.LocalFree(out_blob.pbData)


class RawMessageCipher:
    _AAD = b"local-ai-memory/raw-message/v1"
    _DPAPI_PREFIX = b"DPAPI1\0"
    _PLAIN_PREFIX = b"FILE1\0"

    def __init__(self, key: bytes):
        if len(key) != 32:
            raise ValueError("Encryption key must be 32 bytes")
        self._cipher = AESGCM(key)

    @classmethod
    def _decode_stored_key(cls, stored: bytes) -> bytes:
        if stored.startswith(cls._DPAPI_PREFIX):
            return _dpapi_unprotect(stored[len(cls._DPAPI_PREFIX) :])
        if stored.startswith(cls._PLAIN_PREFIX):
            return stored[len(cls._PLAIN_PREFIX) :]
        raise ValueError("Unsupported key file format")

    @classmethod
    def load_or_create(cls, key_path: Path) -> RawMessageCipher:
        key_path.parent.mkdir(parents=True, exist_ok=True)
        last_read_error: OSError | ValueError | None = None
        for _ in range(100):
            try:
                return cls(cls._decode_stored_key(key_path.read_bytes()))
            except FileNotFoundError:
                pass
            except (OSError, ValueError) as error:
                last_read_error = error
                time.sleep(0.01)
                continue

            key = AESGCM.generate_key(bit_length=256)
            if os.name == "nt":
                stored = cls._DPAPI_PREFIX + _dpapi_protect(key)
            else:
                stored = cls._PLAIN_PREFIX + key
            try:
                descriptor = os.open(
                    key_path,
                    os.O_CREAT | os.O_EXCL | os.O_WRONLY,
                    0o600,
                )
            except FileExistsError:
                time.sleep(0.01)
                continue

            try:
                with os.fdopen(descriptor, "wb") as key_file:
                    key_file.write(stored)
                    key_file.flush()
                    os.fsync(key_file.fileno())
            except BaseException:
                key_path.unlink(missing_ok=True)
                raise
            try:
                key_path.chmod(0o600)
            except OSError:
                pass
            return cls(key)

        if last_read_error is not None:
            raise ValueError("Master key file did not become readable") from last_read_error
        raise TimeoutError("Timed out waiting for concurrent master key creation")

    def encrypt(self, text: str) -> bytes:
        nonce = os.urandom(12)
        payload = self._cipher.encrypt(nonce, text.encode("utf-8"), self._AAD)
        return nonce + payload

    def decrypt(self, payload: bytes) -> str:
        nonce, ciphertext = payload[:12], payload[12:]
        return self._cipher.decrypt(nonce, ciphertext, self._AAD).decode("utf-8")
