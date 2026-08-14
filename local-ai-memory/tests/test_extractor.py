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


if __name__ == "__main__":
    unittest.main()
