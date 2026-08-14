from __future__ import annotations

import ctypes
import hashlib
import os
import re
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
    def load_or_create(cls, key_path: Path) -> RawMessageCipher:
        key_path.parent.mkdir(parents=True, exist_ok=True)
        if key_path.exists():
            stored = key_path.read_bytes()
            if stored.startswith(cls._DPAPI_PREFIX):
                key = _dpapi_unprotect(stored[len(cls._DPAPI_PREFIX) :])
            elif stored.startswith(cls._PLAIN_PREFIX):
                key = stored[len(cls._PLAIN_PREFIX) :]
            else:
                raise ValueError("Unsupported key file format")
            return cls(key)

        key = AESGCM.generate_key(bit_length=256)
        if os.name == "nt":
            stored = cls._DPAPI_PREFIX + _dpapi_protect(key)
        else:
            stored = cls._PLAIN_PREFIX + key
        key_path.write_bytes(stored)
        try:
            key_path.chmod(0o600)
        except OSError:
            pass
        return cls(key)

    def encrypt(self, text: str) -> bytes:
        nonce = os.urandom(12)
        payload = self._cipher.encrypt(nonce, text.encode("utf-8"), self._AAD)
        return nonce + payload

    def decrypt(self, payload: bytes) -> str:
        nonce, ciphertext = payload[:12], payload[12:]
        return self._cipher.decrypt(nonce, ciphertext, self._AAD).decode("utf-8")
