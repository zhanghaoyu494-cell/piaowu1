from __future__ import annotations

import os
import subprocess
import sys
import threading
from datetime import datetime
from pathlib import Path

from .service import MemoryService

LAST_RUN_KEY = "nightly_consolidation_date"
WINDOWS_TASK_NAME = "LocalAIMemoryNightly"


class NightlyScheduler:
    def __init__(
        self,
        service: MemoryService,
        hour: int = 3,
        minute: int = 0,
        poll_seconds: int = 60,
    ):
        self.service = service
        self.hour = hour
        self.minute = minute
        self.poll_seconds = poll_seconds

    def is_due(self, now: datetime | None = None) -> bool:
        current = now or datetime.now().astimezone()
        last_run = self.service.get_setting(LAST_RUN_KEY)
        scheduled_time_passed = (current.hour, current.minute) >= (
            self.hour,
            self.minute,
        )
        return scheduled_time_passed and last_run != current.date().isoformat()

    def run_pending(self, now: datetime | None = None) -> dict[str, int] | None:
        current = now or datetime.now().astimezone()
        if not self.is_due(current):
            return None
        result = self.service.consolidate()
        self.service.set_setting(LAST_RUN_KEY, current.date().isoformat())
        return result

    def run_forever(self, stop_event: threading.Event | None = None) -> None:
        stopper = stop_event or threading.Event()
        while not stopper.is_set():
            self.run_pending()
            stopper.wait(self.poll_seconds)


def install_windows_task(hour: int = 3, minute: int = 0) -> None:
    if os.name != "nt":
        raise RuntimeError(
            "Windows Task Scheduler installation is only available on Windows"
        )
    task_time = f"{hour:02d}:{minute:02d}"
    task_command = f'"{Path(sys.executable).resolve()}" -m local_ai_memory consolidate'
    subprocess.run(
        [
            "schtasks",
            "/Create",
            "/TN",
            WINDOWS_TASK_NAME,
            "/TR",
            task_command,
            "/SC",
            "DAILY",
            "/ST",
            task_time,
            "/RL",
            "LIMITED",
            "/F",
        ],
        check=True,
    )


def uninstall_windows_task() -> None:
    if os.name != "nt":
        raise RuntimeError(
            "Windows Task Scheduler removal is only available on Windows"
        )
    subprocess.run(["schtasks", "/Delete", "/TN", WINDOWS_TASK_NAME, "/F"], check=True)
