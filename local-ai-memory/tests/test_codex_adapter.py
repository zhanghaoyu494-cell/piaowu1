from __future__ import annotations

import unittest

from local_ai_memory.codex_adapter import parse_codex_thread_page


class CodexAdapterTests(unittest.TestCase):
    def test_rejects_missing_or_non_codex_threads(self) -> None:
        with self.assertRaisesRegex(ValueError, "thread.id"):
            parse_codex_thread_page({})
        with self.assertRaisesRegex(ValueError, "Only Codex"):
            parse_codex_thread_page({"thread": {"id": "chat", "kind": "chatgpt"}})

    def test_parses_supported_items_and_ignores_outputs(self) -> None:
        page = parse_codex_thread_page(
            {
                "thread": {
                    "id": "thread-1",
                    "kind": "codex",
                    "cwd": "F:\\project",
                    "createdAt": 1,
                    "updatedAt": 2,
                },
                "page": {"hasMore": True, "nextCursor": "older"},
                "turns": [
                    "ignored-turn",
                    {
                        "id": "newer",
                        "startedAt": 2,
                        "completedAt": 3,
                        "items": [
                            "ignored-item",
                            {"type": "commandExecution", "text": "secret output"},
                            {"type": "agentMessage", "text": ""},
                            {"type": "agentMessage", "text": "answer"},
                        ],
                    },
                    {
                        "id": "older",
                        "startedAt": 1,
                        "items": [
                            {
                                "type": "userMessage",
                                "content": [
                                    "first",
                                    {"type": "text", "text": "second"},
                                    {"type": "image", "url": "ignored"},
                                ],
                            }
                        ],
                    },
                ],
            }
        )

        self.assertEqual(page.conversation.project, "F:\\project")
        self.assertEqual(page.conversation.title, "Untitled Codex task")
        self.assertEqual(
            [message.role for message in page.conversation.messages],
            ["user", "assistant"],
        )
        self.assertEqual(page.conversation.messages[0].content, "first\nsecond")
        self.assertEqual(page.conversation.messages[0].external_id, "older:0")
        self.assertEqual(
            page.conversation.messages[1].created_at, "1970-01-01T00:00:03+00:00"
        )
        self.assertTrue(page.has_more)
        self.assertEqual(page.next_cursor, "older")
        self.assertNotIn(
            "secret output",
            " ".join(message.content for message in page.conversation.messages),
        )


if __name__ == "__main__":
    unittest.main()
