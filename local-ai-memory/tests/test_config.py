from __future__ import annotations

import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from local_ai_memory.config import Settings


class SettingsTests(unittest.TestCase):
    def test_explicit_and_environment_homes_are_resolved(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            explicit = Settings.load(Path(directory) / "explicit")
            with patch.dict(
                os.environ, {"LOCAL_AI_MEMORY_HOME": str(Path(directory) / "env")}
            ):
                configured = Settings.load()

        self.assertEqual(explicit.database_path.name, "memory.sqlite3")
        self.assertEqual(configured.home.name, "env")

    def test_ensure_directories_creates_home(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            settings = Settings.load(Path(directory) / "nested" / "data")
            settings.ensure_directories()
            self.assertTrue(settings.home.is_dir())


if __name__ == "__main__":
    unittest.main()
