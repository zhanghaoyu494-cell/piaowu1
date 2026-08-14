from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from .security import content_hash


@dataclass(slots=True)
class ImportedMessage:
    external_id: str
    role: str
    content: str
    created_at: str
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass(slots=True)
class ImportedConversation:
    external_id: str
    source: str
    project: str
    title: str
    source_uri: str | None
    created_at: str
    updated_at: str
    messages: list[ImportedMessage]


ROLE_MAP = {
    "human": "user",
    "user": "user",
    "用户": "user",
    "assistant": "assistant",
    "ai": "assistant",
    "助手": "assistant",
    "system": "system",
    "系统": "system",
    "tool": "tool",
}


def utc_now() -> str:
    return datetime.now(UTC).isoformat()


def normalize_timestamp(value: Any) -> str:
    if value is None or value == "":
        return utc_now()
    if isinstance(value, (int, float)):
        return datetime.fromtimestamp(value, UTC).isoformat()
    text = str(value).strip()
    if text.endswith("Z"):
        text = text[:-1] + "+00:00"
    try:
        parsed = datetime.fromisoformat(text)
    except ValueError:
        return utc_now()
    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=UTC)
    return parsed.astimezone(UTC).isoformat()


def normalize_role(value: Any) -> str:
    return ROLE_MAP.get(str(value or "unknown").lower(), "unknown")


def _message_text(content: Any) -> str:
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        parts = []
        for item in content:
            if isinstance(item, str):
                parts.append(item)
            elif isinstance(item, dict):
                parts.append(str(item.get("text") or item.get("content") or ""))
        return "\n".join(part for part in parts if part)
    if isinstance(content, dict):
        if isinstance(content.get("parts"), list):
            return _message_text(content["parts"])
        return str(content.get("text") or content.get("content") or "")
    return str(content or "")


def _chatgpt_messages(item: dict[str, Any]) -> list[ImportedMessage]:
    messages = []
    for node_id, node in item.get("mapping", {}).items():
        message = node.get("message") if isinstance(node, dict) else None
        if not message:
            continue
        text = _message_text(message.get("content"))
        if not text.strip():
            continue
        author = message.get("author") or {}
        messages.append(
            ImportedMessage(
                external_id=str(message.get("id") or node_id),
                role=normalize_role(author.get("role")),
                content=text,
                created_at=normalize_timestamp(message.get("create_time")),
                metadata={"recipient": message.get("recipient")},
            )
        )
    messages.sort(key=lambda message: message.created_at)
    return messages


def _generic_messages(item: dict[str, Any]) -> list[ImportedMessage]:
    raw_messages = (
        item.get("messages")
        or item.get("chat_messages")
        or item.get("conversation")
        or []
    )
    messages = []
    for index, message in enumerate(raw_messages):
        if not isinstance(message, dict):
            continue
        text = _message_text(
            message.get("content") or message.get("text") or message.get("message")
        )
        if not text.strip():
            continue
        external_id = (
            message.get("id")
            or message.get("uuid")
            or content_hash(f"{index}:{text}")[:24]
        )
        role = message.get("role") or message.get("sender") or message.get("author")
        if isinstance(role, dict):
            role = role.get("role") or role.get("name")
        messages.append(
            ImportedMessage(
                external_id=str(external_id),
                role=normalize_role(role),
                content=text,
                created_at=normalize_timestamp(
                    message.get("created_at")
                    or message.get("create_time")
                    or message.get("timestamp")
                ),
                metadata={},
            )
        )
    return messages


def _json_conversations(
    data: Any, source: str, project: str
) -> list[ImportedConversation]:
    if (
        isinstance(data, list)
        and data
        and all(isinstance(item, dict) for item in data)
        and all(
            "role" in item and ("content" in item or "text" in item) for item in data
        )
    ):
        items = [
            {
                "id": content_hash(
                    json.dumps(data, ensure_ascii=False, sort_keys=True)
                )[:24],
                "messages": data,
            }
        ]
    else:
        items = data if isinstance(data, list) else [data]
    conversations = []
    for index, item in enumerate(items):
        if not isinstance(item, dict):
            continue
        messages = (
            _chatgpt_messages(item) if "mapping" in item else _generic_messages(item)
        )
        if not messages:
            continue
        external_id = item.get("id") or item.get("uuid") or item.get("conversation_id")
        if not external_id:
            external_id = content_hash(f"{source}:{index}:{messages[0].content}")[:24]
        created_at = normalize_timestamp(
            item.get("create_time") or item.get("created_at") or messages[0].created_at
        )
        updated_at = normalize_timestamp(
            item.get("update_time") or item.get("updated_at") or messages[-1].created_at
        )
        conversations.append(
            ImportedConversation(
                external_id=str(external_id),
                source=source,
                project=project,
                title=str(
                    item.get("title") or item.get("name") or "Untitled conversation"
                ),
                source_uri=item.get("url") or item.get("source_uri"),
                created_at=created_at,
                updated_at=updated_at,
                messages=messages,
            )
        )
    return conversations


MARKDOWN_ROLE = re.compile(
    r"^(?:#{1,6}\s*)?(user|assistant|system|human|ai|用户|助手|系统)\s*[:：]\s*(.*)$",
    re.IGNORECASE,
)


def _markdown_conversation(
    path: Path, source: str, project: str
) -> ImportedConversation:
    text = path.read_text(encoding="utf-8-sig")
    messages: list[ImportedMessage] = []
    current_role = "unknown"
    current_lines: list[str] = []

    def flush() -> None:
        if not current_lines:
            return
        content = "\n".join(current_lines).strip()
        if content:
            messages.append(
                ImportedMessage(
                    external_id=content_hash(f"{len(messages)}:{content}")[:24],
                    role=current_role,
                    content=content,
                    created_at=utc_now(),
                )
            )

    for line in text.splitlines():
        match = MARKDOWN_ROLE.match(line.strip())
        if match:
            flush()
            current_lines = [match.group(2)] if match.group(2) else []
            current_role = normalize_role(match.group(1))
        else:
            current_lines.append(line)
    flush()
    if not messages and text.strip():
        messages.append(
            ImportedMessage(content_hash(text)[:24], "unknown", text.strip(), utc_now())
        )
    timestamp = utc_now()
    return ImportedConversation(
        external_id=content_hash(str(path.resolve()))[:24],
        source=source,
        project=project,
        title=path.stem,
        source_uri=str(path.resolve()),
        created_at=timestamp,
        updated_at=timestamp,
        messages=messages,
    )


def load_conversations(
    path: Path, source: str, project: str = ""
) -> list[ImportedConversation]:
    suffix = path.suffix.lower()
    if suffix == ".json":
        data = json.loads(path.read_text(encoding="utf-8-sig"))
        return _json_conversations(data, source, project)
    if suffix in {".md", ".txt"}:
        return [_markdown_conversation(path, source, project)]
    raise ValueError(f"Unsupported import format: {suffix or '<none>'}")
