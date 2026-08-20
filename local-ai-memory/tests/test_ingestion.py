from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

from local_ai_memory.ingestion import (
    load_conversations,
    normalize_role,
    normalize_timestamp,
)


class IngestionTests(unittest.TestCase):
    def test_normalization_handles_timestamp_and_role_variants(self) -> None:
        self.assertEqual(normalize_timestamp(0), "1970-01-01T00:00:00+00:00")
        self.assertEqual(
            normalize_timestamp("2026-08-20T08:00:00Z"),
            "2026-08-20T08:00:00+00:00",
        )
        self.assertEqual(
            normalize_timestamp("2026-08-20T08:00:00"),
            "2026-08-20T08:00:00+00:00",
        )
        self.assertIn("+00:00", normalize_timestamp("not-a-timestamp"))
        self.assertEqual(normalize_role("Human"), "user")
        self.assertEqual(normalize_role("missing-role"), "unknown")

    def test_generic_json_supports_content_shapes_and_generated_ids(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "conversation.json"
            path.write_text(
                json.dumps(
                    {
                        "name": "Mixed content",
                        "chat_messages": [
                            {
                                "uuid": "message-1",
                                "sender": {"name": "human"},
                                "content": {"parts": ["first", {"text": "second"}]},
                                "timestamp": "2026-08-20T08:00:00Z",
                            },
                            {
                                "author": {"role": "assistant"},
                                "message": [{"content": "reply"}],
                            },
                            "ignored",
                            {"role": "assistant", "content": ""},
                        ],
                    },
                    ensure_ascii=False,
                ),
                encoding="utf-8",
            )

            conversations = load_conversations(path, "test", "project")

        self.assertEqual(len(conversations), 1)
        self.assertEqual(conversations[0].title, "Mixed content")
        self.assertEqual(
            [message.role for message in conversations[0].messages],
            ["user", "assistant"],
        )
        self.assertEqual(conversations[0].messages[0].content, "first\nsecond")
        self.assertEqual(len(conversations[0].messages[1].external_id), 24)

    def test_bare_message_list_becomes_one_conversation(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "messages.json"
            path.write_text(
                json.dumps(
                    [
                        {"role": "user", "text": "question"},
                        {"role": "assistant", "content": "answer"},
                    ]
                ),
                encoding="utf-8",
            )
            conversations = load_conversations(path, "test")

        self.assertEqual(len(conversations), 1)
        self.assertEqual(len(conversations[0].messages), 2)
        self.assertEqual(len(conversations[0].external_id), 24)

    def test_chatgpt_mapping_skips_empty_nodes_and_sorts_messages(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "mapping.json"
            path.write_text(
                json.dumps(
                    {
                        "id": "mapped",
                        "mapping": {
                            "empty": {"message": None},
                            "invalid": "not-a-node",
                            "later": {
                                "message": {
                                    "id": "later-message",
                                    "author": {"role": "assistant"},
                                    "content": {"text": "later"},
                                    "create_time": 2,
                                    "recipient": "all",
                                }
                            },
                            "earlier": {
                                "message": {
                                    "author": {"role": "user"},
                                    "content": {"parts": ["earlier"]},
                                    "create_time": 1,
                                }
                            },
                            "blank": {
                                "message": {
                                    "author": {"role": "user"},
                                    "content": {"text": ""},
                                }
                            },
                        },
                    }
                ),
                encoding="utf-8",
            )
            conversations = load_conversations(path, "test")

        messages = conversations[0].messages
        self.assertEqual(
            [message.content for message in messages], ["earlier", "later"]
        )
        self.assertEqual(messages[0].external_id, "earlier")
        self.assertEqual(messages[1].metadata["recipient"], "all")

    def test_markdown_and_plain_text_imports(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            markdown_path = Path(directory) / "roles.md"
            markdown_path.write_text(
                "# User: first line\ncontinued\nAssistant:\nreply",
                encoding="utf-8",
            )
            plain_path = Path(directory) / "plain.txt"
            plain_path.write_text("plain content", encoding="utf-8")

            markdown = load_conversations(markdown_path, "test", "project")[0]
            plain = load_conversations(plain_path, "test")[0]

        self.assertEqual(
            [message.role for message in markdown.messages], ["user", "assistant"]
        )
        self.assertEqual(markdown.messages[0].content, "first line\ncontinued")
        self.assertEqual(plain.messages[0].role, "unknown")
        self.assertEqual(plain.messages[0].content, "plain content")
        self.assertTrue(markdown.source_uri.endswith("roles.md"))

    def test_unsupported_import_format_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "conversation.csv"
            path.write_text("content", encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "Unsupported import format"):
                load_conversations(path, "test")


if __name__ == "__main__":
    unittest.main()
