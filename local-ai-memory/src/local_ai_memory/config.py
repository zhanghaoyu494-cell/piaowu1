from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True, slots=True)
class Settings:
    home: Path
    database_path: Path
    key_path: Path
    schedule_hour: int = 3
    schedule_minute: int = 0

    @classmethod
    def load(cls, home: str | Path | None = None) -> Settings:
        if home is None:
            configured = os.getenv("LOCAL_AI_MEMORY_HOME")
            if configured:
                resolved_home = Path(configured)
            elif os.name == "nt" and os.getenv("LOCALAPPDATA"):
                resolved_home = Path(os.environ["LOCALAPPDATA"]) / "LocalAIMemory"
            else:
                resolved_home = Path.home() / ".local" / "share" / "local-ai-memory"
        else:
            resolved_home = Path(home)

        resolved_home = resolved_home.expanduser().resolve()
        return cls(
            home=resolved_home,
            database_path=resolved_home / "memory.sqlite3",
            key_path=resolved_home / "master.key",
        )

    def ensure_directories(self) -> None:
        self.home.mkdir(parents=True, exist_ok=True)
