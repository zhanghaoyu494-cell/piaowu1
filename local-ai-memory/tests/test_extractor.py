from __future__ import annotations

import unittest

from local_ai_memory.extractor import HeuristicExtractor


class HeuristicExtractorTests(unittest.TestCase):
    def setUp(self) -> None:
        self.extractor = HeuristicExtractor()

    def test_explicit_user_memory_is_confirmed(self) -> None:
        memories = self.extractor.extract(
            "请记住：所有项目时间字段统一使用 UTC。", "user"
        )

        self.assertEqual(len(memories), 1)
        self.assertEqual(memories[0].status, "confirmed")
        self.assertEqual(memories[0].source_authority, "user")

    def test_markdown_emphasis_is_removed(self) -> None:
        memories = self.extractor.extract(
            "**防止 AI 胡编**：未知事实必须标记。", "assistant"
        )

        self.assertEqual(len(memories), 1)
        self.assertEqual(memories[0].content, "防止 AI 胡编：未知事实必须标记。")

    def test_incomplete_introduction_is_rejected(self) -> None:
        memories = self.extractor.extract(
            "根因是校验器只检查关键词，不检查：", "assistant"
        )

        self.assertEqual(memories, [])

    def test_assistant_process_statement_is_rejected(self) -> None:
        memories = self.extractor.extract(
            "我也会明确列出边界：这个工具不能替代人工评审。", "assistant"
        )

        self.assertEqual(memories, [])

    def test_request_sentence_is_not_extracted_as_a_decision(self) -> None:
        memories = self.extractor.extract(
            "测试项目最终决定采用 PostgreSQL 17 作为主数据库；请只简短复述这项项目决定。",
            "user",
        )

        self.assertEqual(len(memories), 1)
        self.assertIn("PostgreSQL 17", memories[0].content)
        self.assertNotIn("复述", memories[0].content)

    def test_explicit_memory_request_is_not_filtered(self) -> None:
        memories = self.extractor.extract("请记住：项目必须使用 UTC。", "user")

        self.assertEqual(len(memories), 1)
        self.assertEqual(memories[0].status, "confirmed")


if __name__ == "__main__":
    unittest.main()
