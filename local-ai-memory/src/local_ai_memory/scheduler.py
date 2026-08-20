from __future__ import annotations

import threading
from datetime import datetime

from .service import MemoryService

LAST_RUN_KEY = "nightly_consolidation_date"


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
