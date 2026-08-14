from __future__ import annotations

import tempfile
import unittest
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


if __name__ == "__main__":
    unittest.main()
