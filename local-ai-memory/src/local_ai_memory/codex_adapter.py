from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from .ingestion import (
    ImportedConversation,
    ImportedMessage,
    normalize_timestamp,
)


@dataclass(frozen=True, slots=True)
class CodexPage:
    conversation: ImportedConversation
    source_version: str
    has_more: bool
    next_cursor: str | None


def _content_text(content: Any) -> str:
    if isinstance(content, str):
        return content
    if not isinstance(content, list):
        return ""
    parts = []
    for item in content:
        if isinstance(item, str):
            parts.append(item)
        elif isinstance(item, dict) and item.get("type") == "text":
            parts.append(str(item.get("text") or ""))
    return "\n".join(part for part in parts if part)


def _item_message(
    item: dict[str, Any], turn: dict[str, Any], index: int
) -> ImportedMessage | None:
    item_type = item.get("type")
    if item_type == "userMessage":
        role = "user"
        content = _content_text(item.get("content"))
        timestamp = turn.get("startedAt")
    elif item_type == "agentMessage":
        role = "assistant"
        content = str(item.get("text") or "")
        timestamp = turn.get("completedAt") or turn.get("startedAt")
    else:
        return None
    if not content.strip():
        return None
    external_id = str(item.get("id") or f"{turn.get('id', 'turn')}:{index}")
    return ImportedMessage(
        external_id=external_id,
        role=role,
        content=content.strip(),
        created_at=normalize_timestamp(timestamp),
        metadata={
            "codex_turn_id": turn.get("id"),
            "codex_item_type": item_type,
            "phase": item.get("phase"),
        },
    )


def parse_codex_thread_page(payload: dict[str, Any]) -> CodexPage:
    thread = payload.get("thread")
    if not isinstance(thread, dict) or not thread.get("id"):
        raise ValueError("Codex thread payload must include thread.id")
    if thread.get("kind") != "codex":
        raise ValueError("Only Codex task history can be synchronized")

    messages = []
    turns = payload.get("turns") or []
    for turn in reversed(turns):
        if not isinstance(turn, dict):
            continue
        for index, item in enumerate(turn.get("items") or []):
            if not isinstance(item, dict):
                continue
            message = _item_message(item, turn, index)
            if message:
                messages.append(message)

    created_at = normalize_timestamp(thread.get("createdAt"))
    updated_at = normalize_timestamp(thread.get("updatedAt"))
    project = str(thread.get("projectId") or thread.get("cwd") or "")
    thread_id = str(thread["id"])
    page = payload.get("page") if isinstance(payload.get("page"), dict) else {}
    return CodexPage(
        conversation=ImportedConversation(
            external_id=thread_id,
            source="codex",
            project=project,
            title=str(thread.get("title") or "Untitled Codex task"),
            source_uri=f"codex://thread/{thread_id}",
            created_at=created_at,
            updated_at=updated_at,
            messages=messages,
        ),
        source_version=str(thread.get("updatedAt") or ""),
        has_more=bool(page.get("hasMore")),
        next_cursor=str(page["nextCursor"]) if page.get("nextCursor") else None,
    )
