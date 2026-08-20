from __future__ import annotations

import tempfile
import threading
import unittest
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

from local_ai_memory.security import RawMessageCipher, redact_sensitive


class SecurityTests(unittest.TestCase):
    def test_cipher_round_trip_and_ciphertext_hides_plaintext(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            cipher = RawMessageCipher.load_or_create(Path(directory) / "key")
            payload = cipher.encrypt("private conversation 私密对话")
            self.assertNotIn(b"private conversation", payload)
            self.assertEqual(cipher.decrypt(payload), "private conversation 私密对话")

    def test_redaction_detects_secret_and_personal_data(self) -> None:
        result = redact_sensitive(
            "api_key=sk-abcdefghijklmnopqrstuvwxyz123456 email user@example.com phone 13812345678"
        )
        self.assertTrue(result.secret_detected)
        self.assertEqual(result.sensitivity, "high")
        self.assertNotIn("sk-abcdefghijklmnopqrstuvwxyz", result.text)
        self.assertIn("[REDACTED_EMAIL]", result.text)
        self.assertIn("[REDACTED_PHONE]", result.text)

    def test_redaction_detects_aws_and_chinese_labeled_secrets(self) -> None:
        for content in (
            "AKIAIOSFODNN7EXAMPLE",
            "ASIAIOSFODNN7EXAMPLE",
            "数据库密码：example-value-123456",
            "访问令牌 = example-token-value-123456",
        ):
            with self.subTest(content=content):
                result = redact_sensitive(content)
                self.assertTrue(result.secret_detected)
                self.assertEqual(result.sensitivity, "high")
                self.assertNotIn(content, result.text)

    def test_concurrent_first_use_creates_one_master_key(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            key_path = Path(directory) / "key"
            barrier = threading.Barrier(16)

            def load_cipher() -> RawMessageCipher:
                barrier.wait()
                return RawMessageCipher.load_or_create(key_path)

            with ThreadPoolExecutor(max_workers=16) as executor:
                ciphers = list(executor.map(lambda _: load_cipher(), range(16)))

            payload = ciphers[0].encrypt("shared concurrent key")
            for cipher in ciphers:
                self.assertEqual(cipher.decrypt(payload), "shared concurrent key")


if __name__ == "__main__":
    unittest.main()
