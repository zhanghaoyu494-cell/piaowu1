from __future__ import annotations

import json
import os
import sys
import tempfile
import unittest
from pathlib import Path
from typing import Any

from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client


def decode_result(result: Any) -> Any:
    if result.isError:
        message = "\n".join(getattr(item, "text", "") for item in result.content)
        raise AssertionError(f"MCP tool failed: {message}")
    structured = result.structuredContent
    if structured is not None:
        if set(structured) == {"result"}:
            return structured["result"]
        return structured
    texts = [
        item.text for item in result.content if getattr(item, "type", None) == "text"
    ]
    return json.loads("\n".join(texts)) if texts else None


class McpStdioIntegrationTests(unittest.IsolatedAsyncioTestCase):
    async def test_stdio_end_to_end_workflow(self) -> None:
        temporary_directory = tempfile.TemporaryDirectory(prefix="local-ai-memory-e2e-")
        data_directory = Path(temporary_directory.name)
        environment = dict(os.environ)
        environment["LOCAL_AI_MEMORY_HOME"] = str(data_directory)
        server = StdioServerParameters(
            command=sys.executable,
            args=["-m", "local_ai_memory.mcp_server"],
            cwd=str(Path(__file__).parents[1]),
            env=environment,
        )

        try:
            async with stdio_client(server) as streams:  # noqa: SIM117
                async with ClientSession(*streams) as session:
                    await session.initialize()
                    tools = await session.list_tools()
                    self.assertEqual(len(tools.tools), 13)
                    annotations = {tool.name: tool.annotations for tool in tools.tools}
                    read_only_tools = {
                        "memory_search",
                        "memory_candidates",
                        "memory_conversations",
                        "memory_source",
                        "memory_stats",
                        "memory_codex_sync_plan",
                    }
                    destructive_tools = {
                        "memory_delete",
                        "memory_delete_conversation",
                    }
                    for name in read_only_tools:
                        self.assertTrue(annotations[name].readOnlyHint)
                        self.assertFalse(annotations[name].destructiveHint)
                        self.assertFalse(annotations[name].openWorldHint)
                    for name in destructive_tools:
                        self.assertFalse(annotations[name].readOnlyHint)
                        self.assertTrue(annotations[name].destructiveHint)
                        self.assertFalse(annotations[name].openWorldHint)
                    for name in set(annotations) - read_only_tools - destructive_tools:
                        self.assertFalse(annotations[name].readOnlyHint)
                        self.assertFalse(annotations[name].destructiveHint)
                        self.assertFalse(annotations[name].openWorldHint)
                    initial_stats = await self.call(session, "memory_stats")
                    self.assertEqual(initial_stats["plugin_version"], "0.3.2")

                    threads = [
                        {
                            "id": "e2e-thread",
                            "kind": "codex",
                            "status": "notLoaded",
                            "hostId": "local",
                            "projectId": "e2e-project",
                            "updatedAt": 1786692000,
                            "title": "E2E memory test",
                        },
                        {
                            "id": "active-thread",
                            "kind": "codex",
                            "status": "active",
                            "updatedAt": 1786692001,
                        },
                        {
                            "id": "chatgpt-thread",
                            "kind": "chatgpt",
                            "status": "notLoaded",
                            "updatedAt": 1786692002,
                        },
                    ]
                    plan = await self.call(
                        session, "memory_codex_sync_plan", {"threads": threads}
                    )
                    self.assertEqual(plan["pending_count"], 1)
                    self.assertEqual(plan["active_skipped"], ["active-thread"])
                    self.assertEqual(plan["ignored_count"], 1)

                    nonce = "RAW-CIPHERTEXT-NONCE-7F3A9C"
                    payload = {
                        "schemaVersion": 4,
                        "thread": {
                            "id": "e2e-thread",
                            "kind": "codex",
                            "hostId": "local",
                            "title": "E2E memory test",
                            "projectId": "e2e-project",
                            "cwd": "F:\\e2e-project",
                            "createdAt": 1786691000,
                            "updatedAt": 1786692000,
                        },
                        "page": {
                            "order": "newest_first",
                            "limit": 20,
                            "nextCursor": None,
                            "hasMore": False,
                        },
                        "turns": [
                            {
                                "id": "turn-new",
                                "startedAt": 1786691200,
                                "completedAt": 1786691210,
                                "items": [
                                    {
                                        "type": "userMessage",
                                        "id": "user-explicit",
                                        "content": [
                                            {
                                                "type": "text",
                                                "text": "请记住：测试项目的时间字段统一使用 UTC。",
                                            }
                                        ],
                                    },
                                    {
                                        "type": "agentMessage",
                                        "id": "assistant-final",
                                        "text": f"已记录。{nonce}",
                                        "phase": "final_answer",
                                    },
                                ],
                            },
                            {
                                "id": "turn-old",
                                "startedAt": 1786691100,
                                "completedAt": 1786691110,
                                "items": [
                                    {
                                        "type": "userMessage",
                                        "id": "user-candidate",
                                        "content": [
                                            {
                                                "type": "text",
                                                "text": "我们决定使用 PostgreSQL 作为端到端测试数据库。",
                                            }
                                        ],
                                    }
                                ],
                            },
                        ],
                    }
                    ingest = await self.call(
                        session,
                        "memory_ingest_codex_page",
                        {"payload": payload},
                    )
                    self.assertTrue(ingest["fully_synchronized"])
                    self.assertEqual(ingest["messages_added"], 3)

                    repeated = await self.call(
                        session,
                        "memory_ingest_codex_page",
                        {"payload": payload},
                    )
                    self.assertEqual(repeated["messages_added"], 0)

                    plan_after = await self.call(
                        session,
                        "memory_codex_sync_plan",
                        {"threads": [threads[0]]},
                    )
                    self.assertEqual(plan_after["pending_count"], 0)

                    utc_results = await self.call(
                        session,
                        "memory_search",
                        {"query": "UTC", "project": "e2e-project"},
                    )
                    self.assertEqual(len(utc_results), 1)
                    self.assertEqual(utc_results[0]["status"], "confirmed")
                    source = await self.call(
                        session,
                        "memory_source",
                        {"message_id": utc_results[0]["sources"][0]["message_id"]},
                    )
                    self.assertIn("统一使用 UTC", source["content"])
                    self.assertEqual(source["conversation_external_id"], "e2e-thread")

                    candidates = await self.call(
                        session,
                        "memory_candidates",
                        {"project": "e2e-project"},
                    )
                    postgres = next(
                        item for item in candidates if "PostgreSQL" in item["content"]
                    )
                    default_search = await self.call(
                        session,
                        "memory_search",
                        {"query": "PostgreSQL", "project": "e2e-project"},
                    )
                    self.assertEqual(default_search, [])
                    confirmed = await self.call(
                        session,
                        "memory_confirm",
                        {"memory_id": postgres["id"]},
                    )
                    self.assertEqual(confirmed["status"], "confirmed")
                    confirmed_search = await self.call(
                        session,
                        "memory_search",
                        {"query": "PostgreSQL", "project": "e2e-project"},
                    )
                    self.assertEqual(len(confirmed_search), 1)

                    secret_result = await session.call_tool(
                        "memory_remember",
                        {
                            "content": "password=super-secret-value",
                            "project": "e2e-project",
                            "kind": "fact",
                        },
                    )
                    self.assertTrue(secret_result.isError)

                    high_sensitivity_result = await session.call_tool(
                        "memory_remember",
                        {
                            "content": "Internal compensation plan",
                            "sensitivity": "high",
                        },
                    )
                    self.assertTrue(high_sensitivity_result.isError)

                    invalid_kind_result = await session.call_tool(
                        "memory_remember",
                        {"content": "Ordinary fact", "kind": "not-a-real-kind"},
                    )
                    self.assertTrue(invalid_kind_result.isError)

                    multi_thread = {
                        "id": "multi-page-thread",
                        "kind": "codex",
                        "title": "Multi page",
                        "createdAt": 1786691000,
                        "updatedAt": 1786693000,
                    }
                    final_page = {
                        "thread": multi_thread,
                        "page": {"hasMore": False, "nextCursor": None},
                        "turns": [],
                    }
                    out_of_order = await session.call_tool(
                        "memory_ingest_codex_page",
                        {"payload": final_page, "cursor_used": "page-2"},
                    )
                    self.assertTrue(out_of_order.isError)
                    newest_page = {
                        "thread": multi_thread,
                        "page": {"hasMore": True, "nextCursor": "page-2"},
                        "turns": [],
                    }
                    await self.call(
                        session,
                        "memory_ingest_codex_page",
                        {"payload": newest_page},
                    )
                    completed = await self.call(
                        session,
                        "memory_ingest_codex_page",
                        {"payload": final_page, "cursor_used": "page-2"},
                    )
                    self.assertTrue(completed["fully_synchronized"])

                    conversations = await self.call(
                        session, "memory_conversations", {"source": "codex"}
                    )
                    for conversation in conversations:
                        await self.call(
                            session,
                            "memory_delete_conversation",
                            {"conversation_id": conversation["id"]},
                        )
                    stats = await self.call(session, "memory_stats")
                    self.assertEqual(
                        stats,
                        {
                            "conversations": 0,
                            "messages": 0,
                            "memories": 0,
                            "confirmed": 0,
                            "candidates": 0,
                            "plugin_version": "0.3.2",
                        },
                    )

            database_files = list(data_directory.glob("memory.sqlite3*"))
            raw_bytes = b"".join(
                path.read_bytes() for path in database_files if path.is_file()
            )
            self.assertNotIn(nonce.encode(), raw_bytes)
        finally:
            temporary_directory.cleanup()

    async def call(
        self,
        session: ClientSession,
        name: str,
        arguments: dict[str, Any] | None = None,
    ) -> Any:
        return decode_result(await session.call_tool(name, arguments or {}))


if __name__ == "__main__":
    unittest.main()
