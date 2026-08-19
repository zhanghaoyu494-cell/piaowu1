from __future__ import annotations

import json
import tempfile
import unittest
from datetime import datetime
from pathlib import Path

from local_ai_memory.config import Settings
from local_ai_memory.scheduler import LAST_RUN_KEY, NightlyScheduler
from local_ai_memory.service import MemoryService


class MemoryServiceTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.home = Path(self.temporary_directory.name) / "data"
        self.service = MemoryService(Settings.load(self.home))

    def tearDown(self) -> None:
        self.temporary_directory.cleanup()

    def test_remember_search_and_secret_refusal(self) -> None:
        memory = self.service.remember(
            "该项目必须使用 Python 3.12，并采用 SQLite 保存本地知识。",
            project="memory-project",
            kind="constraint",
        )
        results = self.service.search("Python SQLite", project="memory-project")
        self.assertEqual(results[0]["id"], memory["id"])
        self.assertEqual(results[0]["status"], "confirmed")
        with self.assertRaises(ValueError):
            self.service.remember("密码 password=super-secret-value")

    def test_remember_validates_kind_and_sensitivity(self) -> None:
        with self.assertRaisesRegex(ValueError, "Unsupported memory kind"):
            self.service.remember("Ordinary fact", kind="not-a-real-kind")  # type: ignore[arg-type]
        with self.assertRaisesRegex(ValueError, "Unsupported memory sensitivity"):
            self.service.remember("Ordinary fact", sensitivity="anything")  # type: ignore[arg-type]
        with self.assertRaisesRegex(ValueError, "high-sensitivity"):
            self.service.remember(
                "Internal compensation plan will start next month",
                sensitivity="high",
            )

        database_bytes = b"".join(
            path.read_bytes()
            for path in self.home.glob("memory.sqlite3*")
            if path.is_file()
        )
        self.assertNotIn(b"Internal compensation plan", database_bytes)

    def test_personal_data_cannot_be_downgraded(self) -> None:
        memory = self.service.remember(
            "Contact user@example.com for the approved process",
            sensitivity="normal",
        )
        self.assertEqual(memory["sensitivity"], "personal")
        self.assertIn("[REDACTED_EMAIL]", memory["content"])

    def test_delete_memory_removes_plaintext_from_database_files(self) -> None:
        marker = "LAM-SECURE-DELETE-MARKER-20260819"
        with self.service.database.connect() as connection:
            self.assertEqual(connection.execute("PRAGMA secure_delete").fetchone()[0], 1)
            fts_config = dict(
                connection.execute(
                    "SELECT k, v FROM memories_fts_config"
                ).fetchall()
            )
        self.assertEqual(fts_config.get("secure-delete"), 1)
        memory = self.service.remember(marker, kind="fact")
        self.assertTrue(self.service.delete_memory(memory["id"]))
        database_bytes = b"".join(
            path.read_bytes()
            for path in self.home.glob("memory.sqlite3*")
            if path.is_file()
        )
        self.assertNotIn(marker.encode(), database_bytes)
        self.assertEqual(self.service.search(marker), [])

    def test_delete_conversation_removes_raw_plaintext_from_database_files(self) -> None:
        raw_marker = "LAM-RAW-DELETE-MARKER-20260819"
        import_path = Path(self.temporary_directory.name) / "delete-test.json"
        import_path.write_text(
            json.dumps(
                {
                    "id": "delete-conversation",
                    "messages": [
                        {
                            "id": "delete-message",
                            "role": "assistant",
                            "content": raw_marker,
                        }
                    ],
                }
            ),
            encoding="utf-8",
        )
        self.service.import_file(import_path, "test", "delete-project")
        conversation_id = self.service.list_conversations(project="delete-project")[0][
            "id"
        ]
        self.assertTrue(self.service.delete_conversation(conversation_id)["deleted"])
        database_bytes = b"".join(
            path.read_bytes()
            for path in self.home.glob("memory.sqlite3*")
            if path.is_file()
        )
        self.assertNotIn(raw_marker.encode(), database_bytes)

    def test_import_extract_review_and_source(self) -> None:
        import_path = Path(self.temporary_directory.name) / "conversation.json"
        import_path.write_text(
            json.dumps(
                {
                    "id": "conversation-1",
                    "title": "架构讨论",
                    "messages": [
                        {
                            "id": "message-1",
                            "role": "user",
                            "content": "我们决定采用 SQLite 作为第一版本地数据库。",
                            "created_at": "2026-08-14T10:00:00+08:00",
                        },
                        {
                            "id": "message-2",
                            "role": "user",
                            "content": "请记住：所有检索结果必须保留原始来源。",
                            "created_at": "2026-08-14T10:01:00+08:00",
                        },
                    ],
                },
                ensure_ascii=False,
            ),
            encoding="utf-8",
        )
        report = self.service.import_file(import_path, "chatgpt", "memory-project")
        self.assertEqual(report.conversations_added, 1)
        self.assertEqual(report.messages_added, 2)
        self.assertEqual(
            len(self.service.search("SQLite", project="memory-project")), 0
        )
        candidates = self.service.list_candidates("memory-project")
        self.assertTrue(any("SQLite" in item["content"] for item in candidates))
        explicit = self.service.search("原始来源", project="memory-project")
        self.assertEqual(len(explicit), 1)
        message_id = explicit[0]["sources"][0]["message_id"]
        source = self.service.get_source_message(message_id)
        self.assertIn("请记住", source["content"])

        candidate = next(item for item in candidates if "SQLite" in item["content"])
        self.service.set_status(candidate["id"], "confirmed")
        self.assertEqual(
            len(self.service.search("SQLite", project="memory-project")), 1
        )

        second_report = self.service.import_file(
            import_path, "chatgpt", "memory-project"
        )
        self.assertEqual(second_report.conversations_added, 0)
        self.assertEqual(second_report.messages_added, 0)

        conversation_id = self.service.list_conversations(project="memory-project")[0][
            "id"
        ]
        deletion = self.service.delete_conversation(conversation_id)
        self.assertTrue(deletion["deleted"])
        self.assertEqual(deletion["messages_deleted"], 2)
        self.assertEqual(self.service.stats()["conversations"], 0)
        self.assertEqual(self.service.stats()["memories"], 0)

    def test_scheduler_runs_once_after_due_time(self) -> None:
        scheduler = NightlyScheduler(self.service, hour=3)
        due_time = datetime(2026, 8, 14, 3, 1).astimezone()
        self.assertTrue(scheduler.is_due(due_time))
        self.assertIsNotNone(scheduler.run_pending(due_time))
        self.assertEqual(
            self.service.get_setting(LAST_RUN_KEY), due_time.date().isoformat()
        )
        self.assertFalse(scheduler.is_due(due_time))

    def test_codex_zero_import_incremental_sync(self) -> None:
        thread_summary = {
            "id": "codex-thread-1",
            "kind": "codex",
            "hostId": "local",
            "projectId": "project-1",
            "status": "notLoaded",
            "cwd": "F:\\project-1",
            "updatedAt": 1786689000,
            "title": "本地记忆架构",
            "summary": "决定使用 SQLite",
        }
        plan = self.service.codex_sync_plan([thread_summary])
        self.assertEqual(plan["pending_count"], 1)
        self.assertEqual(plan["pending"][0]["reason"], "new")

        payload = {
            "thread": {
                "id": "codex-thread-1",
                "kind": "codex",
                "title": "本地记忆架构",
                "cwd": "F:\\project-1",
                "projectId": "project-1",
                "createdAt": 1786688000,
                "updatedAt": 1786689000,
            },
            "page": {
                "order": "newest_first",
                "limit": 20,
                "nextCursor": None,
                "hasMore": False,
            },
            "turns": [
                {
                    "id": "turn-1",
                    "startedAt": 1786688100,
                    "completedAt": 1786688200,
                    "items": [
                        {
                            "type": "userMessage",
                            "id": "item-user-1",
                            "content": [
                                {
                                    "type": "text",
                                    "text": "请记住：该项目必须保留 Codex 原始任务来源。",
                                }
                            ],
                        },
                        {
                            "type": "agentMessage",
                            "id": "item-agent-1",
                            "text": "已经按要求记录。",
                            "phase": "final_answer",
                        },
                    ],
                }
            ],
        }
        result = self.service.ingest_codex_page(payload)
        self.assertTrue(result["fully_synchronized"])
        self.assertEqual(result["messages_added"], 2)
        self.assertEqual(
            self.service.codex_sync_plan([thread_summary])["pending_count"], 0
        )
        memories = self.service.search("Codex 原始任务", project="project-1")
        self.assertEqual(len(memories), 1)
        self.assertEqual(memories[0]["sources"][0]["source"], "codex")

    def test_codex_multipage_sync_requires_newest_page(self) -> None:
        base_thread = {
            "id": "codex-thread-multi",
            "kind": "codex",
            "title": "多页任务",
            "cwd": "F:\\project-multi",
            "projectId": "project-multi",
            "createdAt": 1786687000,
            "updatedAt": 1786689001,
        }
        final_page = {
            "thread": base_thread,
            "page": {"hasMore": False, "nextCursor": None},
            "turns": [],
        }
        with self.assertRaisesRegex(ValueError, "restart from the newest page"):
            self.service.ingest_codex_page(final_page, cursor_used="older")

        newest_page = {
            "thread": base_thread,
            "page": {"hasMore": True, "nextCursor": "older"},
            "turns": [],
        }
        self.service.ingest_codex_page(newest_page)
        with self.assertRaisesRegex(ValueError, "restart from the newest page"):
            self.service.ingest_codex_page(final_page, cursor_used="wrong-cursor")
        complete = self.service.ingest_codex_page(final_page, cursor_used="older")
        self.assertTrue(complete["fully_synchronized"])

    def test_codex_multipage_sync_requires_every_cursor(self) -> None:
        thread = {
            "id": "codex-thread-three-pages",
            "kind": "codex",
            "title": "Three page task",
            "createdAt": 1786687000,
            "updatedAt": 1786689002,
        }
        newest_page = {
            "thread": thread,
            "page": {"hasMore": True, "nextCursor": "page-2"},
            "turns": [],
        }
        middle_page = {
            "thread": thread,
            "page": {"hasMore": True, "nextCursor": "page-3"},
            "turns": [],
        }
        final_page = {
            "thread": thread,
            "page": {"hasMore": False, "nextCursor": None},
            "turns": [],
        }

        self.service.ingest_codex_page(newest_page)
        with self.assertRaisesRegex(ValueError, "restart from the newest page"):
            self.service.ingest_codex_page(final_page, cursor_used="page-3")
        self.service.ingest_codex_page(middle_page, cursor_used="page-2")
        complete = self.service.ingest_codex_page(final_page, cursor_used="page-3")

        self.assertTrue(complete["fully_synchronized"])


if __name__ == "__main__":
    unittest.main()
