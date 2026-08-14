from __future__ import annotations

import json
import sqlite3
import uuid
from dataclasses import asdict, dataclass
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from .codex_adapter import parse_codex_thread_page
from .config import Settings
from .database import Database
from .extractor import HeuristicExtractor, MemoryCandidate
from .ingestion import ImportedConversation, load_conversations
from .security import RawMessageCipher, content_hash, redact_sensitive
from .text import fts_query, normalize_text, search_document


def utc_now() -> str:
    return datetime.now(UTC).isoformat()


@dataclass(frozen=True, slots=True)
class ImportReport:
    conversations_seen: int
    conversations_added: int
    messages_seen: int
    messages_added: int
    memories_added: int


class MemoryService:
    def __init__(
        self,
        settings: Settings | None = None,
        extractor: HeuristicExtractor | None = None,
    ):
        self.settings = settings or Settings.load()
        self.settings.ensure_directories()
        self.database = Database(self.settings.database_path)
        self.cipher = RawMessageCipher.load_or_create(self.settings.key_path)
        self.extractor = extractor or HeuristicExtractor()

    def import_file(
        self, path: str | Path, source: str, project: str = ""
    ) -> ImportReport:
        conversations = load_conversations(Path(path), source=source, project=project)
        return self.ingest_conversations(conversations)

    def ingest_conversations(
        self, conversations: list[ImportedConversation]
    ) -> ImportReport:
        conversations_added = 0
        messages_seen = 0
        messages_added = 0
        memory_count_before = self.stats()["memories"]
        new_message_ids: list[str] = []

        with self.database.transaction() as connection:
            for conversation in conversations:
                conversation_id, added = self._upsert_conversation(
                    connection, conversation
                )
                conversations_added += int(added)
                for message in conversation.messages:
                    messages_seen += 1
                    message_id = str(uuid.uuid4())
                    cursor = connection.execute(
                        """
                        INSERT OR IGNORE INTO messages(
                            id, conversation_id, external_id, role, content_cipher,
                            content_hash, created_at, metadata_json
                        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                        """,
                        (
                            message_id,
                            conversation_id,
                            message.external_id,
                            message.role,
                            self.cipher.encrypt(message.content),
                            content_hash(message.content),
                            message.created_at,
                            json.dumps(message.metadata, ensure_ascii=False),
                        ),
                    )
                    if cursor.rowcount:
                        messages_added += 1
                        new_message_ids.append(message_id)

        for message_id in new_message_ids:
            self._process_message(message_id)

        return ImportReport(
            conversations_seen=len(conversations),
            conversations_added=conversations_added,
            messages_seen=messages_seen,
            messages_added=messages_added,
            memories_added=self.stats()["memories"] - memory_count_before,
        )

    def codex_sync_plan(self, threads: list[dict[str, Any]]) -> dict[str, Any]:
        with self.database.connect() as connection:
            rows = connection.execute(
                "SELECT key, value FROM settings WHERE key LIKE 'codex_thread_version:%'"
            ).fetchall()
        versions = {
            row["key"].removeprefix("codex_thread_version:"): row["value"]
            for row in rows
        }
        pending = []
        active = []
        ignored = 0
        for thread in threads:
            if not isinstance(thread, dict) or thread.get("kind") != "codex":
                ignored += 1
                continue
            thread_id = str(thread.get("id") or "")
            if not thread_id:
                ignored += 1
                continue
            if thread.get("status") == "active":
                active.append(thread_id)
                continue
            source_version = str(thread.get("updatedAt") or "")
            if versions.get(thread_id) == source_version:
                continue
            pending.append(
                {
                    "thread_id": thread_id,
                    "host_id": thread.get("hostId"),
                    "project_id": thread.get("projectId"),
                    "cwd": thread.get("cwd"),
                    "title": thread.get("title"),
                    "summary": thread.get("summary"),
                    "updated_at": thread.get("updatedAt"),
                    "reason": "new" if thread_id not in versions else "updated",
                }
            )
        return {
            "threads_seen": len(threads),
            "pending": pending,
            "pending_count": len(pending),
            "active_skipped": active,
            "ignored_count": ignored,
        }

    def ingest_codex_page(
        self, payload: dict[str, Any], cursor_used: str | None = None
    ) -> dict[str, Any]:
        page = parse_codex_thread_page(payload)
        progress_key = f"codex_sync_progress:{page.conversation.external_id}"
        version_key = f"codex_thread_version:{page.conversation.external_id}"
        progress = self._codex_sync_progress(progress_key)

        if cursor_used is None:
            progress = None
        elif (
            progress is None
            or progress.get("source_version") != page.source_version
            or progress.get("expected_cursor") != cursor_used
        ):
            raise ValueError(
                "Codex page cursor is out of sequence; restart from the newest page"
            )

        if page.has_more and not page.next_cursor:
            raise ValueError("Codex page reports more history without a next cursor")

        report = self.ingest_conversations([page.conversation])
        fully_synchronized = False
        if page.has_more:
            self.set_setting(
                progress_key,
                json.dumps(
                    {
                        "source_version": page.source_version,
                        "expected_cursor": page.next_cursor,
                    }
                ),
            )
        else:
            self.set_setting(version_key, page.source_version)
            self.delete_setting(progress_key)
            fully_synchronized = True
        return {
            **asdict(report),
            "thread_id": page.conversation.external_id,
            "page_messages": len(page.conversation.messages),
            "fully_synchronized": fully_synchronized,
            "next_cursor": page.next_cursor,
        }

    def _codex_sync_progress(self, key: str) -> dict[str, str] | None:
        value = self.get_setting(key)
        if value is None:
            return None
        try:
            progress = json.loads(value)
        except json.JSONDecodeError:
            return None
        if not isinstance(progress, dict):
            return None
        source_version = progress.get("source_version")
        expected_cursor = progress.get("expected_cursor")
        if not isinstance(source_version, str) or not isinstance(expected_cursor, str):
            return None
        return {
            "source_version": source_version,
            "expected_cursor": expected_cursor,
        }

    def _upsert_conversation(
        self, connection: sqlite3.Connection, conversation: ImportedConversation
    ) -> tuple[str, bool]:
        existing = connection.execute(
            "SELECT id FROM conversations WHERE source = ? AND external_id = ?",
            (conversation.source, conversation.external_id),
        ).fetchone()
        if existing:
            connection.execute(
                """
                UPDATE conversations
                SET project = ?, title = ?, source_uri = ?, updated_at = ?, imported_at = ?
                WHERE id = ?
                """,
                (
                    conversation.project,
                    conversation.title,
                    conversation.source_uri,
                    conversation.updated_at,
                    utc_now(),
                    existing["id"],
                ),
            )
            return existing["id"], False

        conversation_id = str(uuid.uuid4())
        connection.execute(
            """
            INSERT INTO conversations(
                id, source, external_id, project, title, source_uri,
                created_at, updated_at, imported_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (
                conversation_id,
                conversation.source,
                conversation.external_id,
                conversation.project,
                conversation.title,
                conversation.source_uri,
                conversation.created_at,
                conversation.updated_at,
                utc_now(),
            ),
        )
        return conversation_id, True

    def _process_message(self, message_id: str) -> int:
        with self.database.connect() as connection:
            row = connection.execute(
                """
                SELECT m.*, c.project
                FROM messages m JOIN conversations c ON c.id = m.conversation_id
                WHERE m.id = ?
                """,
                (message_id,),
            ).fetchone()
        if not row or row["knowledge_processed_at"]:
            return 0

        content = self.cipher.decrypt(row["content_cipher"])
        candidates = self.extractor.extract(content, row["role"])
        added = 0
        with self.database.transaction() as connection:
            for candidate in candidates:
                added += int(
                    self._store_candidate(
                        connection, candidate, row["project"], message_id
                    )
                )
            connection.execute(
                "UPDATE messages SET knowledge_processed_at = ? WHERE id = ?",
                (utc_now(), message_id),
            )
        return added

    def _store_candidate(
        self,
        connection: sqlite3.Connection,
        candidate: MemoryCandidate,
        project: str,
        message_id: str | None,
    ) -> bool:
        normalized = normalize_text(candidate.content)
        fingerprint = content_hash(f"{project}\0{candidate.kind}\0{normalized}")
        existing = connection.execute(
            "SELECT id, confidence, status FROM memories WHERE fingerprint = ?",
            (fingerprint,),
        ).fetchone()
        if existing:
            memory_id = existing["id"]
            status = (
                "confirmed"
                if "confirmed" in {existing["status"], candidate.status}
                else existing["status"]
            )
            connection.execute(
                "UPDATE memories SET confidence = ?, status = ?, updated_at = ? WHERE id = ?",
                (
                    max(existing["confidence"], candidate.confidence),
                    status,
                    utc_now(),
                    memory_id,
                ),
            )
            added = False
        else:
            memory_id = str(uuid.uuid4())
            now = utc_now()
            connection.execute(
                """
                INSERT INTO memories(
                    id, content, normalized_content, search_terms, fingerprint,
                    kind, project, confidence, status, sensitivity,
                    source_authority, created_at, updated_at, last_verified_at
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    memory_id,
                    candidate.content,
                    normalized,
                    search_document(candidate.content),
                    fingerprint,
                    candidate.kind,
                    project,
                    candidate.confidence,
                    candidate.status,
                    candidate.sensitivity,
                    candidate.source_authority,
                    now,
                    now,
                    now if candidate.status == "confirmed" else None,
                ),
            )
            added = True

        if message_id:
            connection.execute(
                "INSERT OR IGNORE INTO memory_sources(memory_id, message_id, quote) VALUES (?, ?, ?)",
                (memory_id, message_id, candidate.content),
            )
        return added

    def remember(
        self,
        content: str,
        project: str = "",
        kind: str = "fact",
        sensitivity: str | None = None,
    ) -> dict[str, Any]:
        redaction = redact_sensitive(content.strip())
        if not content.strip():
            raise ValueError("Memory content cannot be empty")
        if redaction.secret_detected:
            raise ValueError(
                "Refusing to place a detected secret in the searchable knowledge base"
            )
        candidate = MemoryCandidate(
            content=redaction.text,
            kind=kind,
            confidence=1.0,
            status="confirmed",
            sensitivity=sensitivity or redaction.sensitivity,
            source_authority="user",
        )
        with self.database.transaction() as connection:
            self._store_candidate(connection, candidate, project, None)
            fingerprint = content_hash(
                f"{project}\0{kind}\0{normalize_text(candidate.content)}"
            )
            memory_id = connection.execute(
                "SELECT id FROM memories WHERE fingerprint = ?", (fingerprint,)
            ).fetchone()["id"]
        return self.get_memory(memory_id)

    def search(
        self,
        query: str,
        project: str | None = None,
        limit: int = 5,
        include_candidates: bool = False,
    ) -> list[dict[str, Any]]:
        match = fts_query(query)
        if not match:
            return []
        statuses = ("confirmed", "candidate") if include_candidates else ("confirmed",)
        placeholders = ",".join("?" for _ in statuses)
        parameters: list[Any] = [match, *statuses]
        project_clause = ""
        if project is not None:
            project_clause = " AND m.project = ?"
            parameters.append(project)
        parameters.append(max(1, min(limit, 50)))
        sql = f"""
            SELECT m.*, bm25(memories_fts, 4.0, 1.5, 0.5, 0.5) AS fts_rank
            FROM memories_fts
            JOIN memories m ON m.rowid = memories_fts.rowid
            WHERE memories_fts MATCH ?
              AND m.status IN ({placeholders})
              {project_clause}
            ORDER BY fts_rank ASC, m.confidence DESC, m.updated_at DESC
            LIMIT ?
        """
        with self.database.connect() as connection:
            rows = connection.execute(sql, parameters).fetchall()
        return [self._memory_dict(row, include_sources=True) for row in rows]

    def list_candidates(
        self, project: str | None = None, limit: int = 50
    ) -> list[dict[str, Any]]:
        sql = "SELECT * FROM memories WHERE status = 'candidate'"
        parameters: list[Any] = []
        if project is not None:
            sql += " AND project = ?"
            parameters.append(project)
        sql += " ORDER BY confidence DESC, created_at DESC LIMIT ?"
        parameters.append(max(1, min(limit, 200)))
        with self.database.connect() as connection:
            rows = connection.execute(sql, parameters).fetchall()
        return [self._memory_dict(row, include_sources=True) for row in rows]

    def get_memory(self, memory_id: str) -> dict[str, Any]:
        with self.database.connect() as connection:
            row = connection.execute(
                "SELECT * FROM memories WHERE id = ?", (memory_id,)
            ).fetchone()
        if not row:
            raise KeyError(f"Memory not found: {memory_id}")
        return self._memory_dict(row, include_sources=True)

    def _memory_dict(self, row: sqlite3.Row, include_sources: bool) -> dict[str, Any]:
        result = {
            "id": row["id"],
            "content": row["content"],
            "kind": row["kind"],
            "project": row["project"],
            "confidence": row["confidence"],
            "status": row["status"],
            "sensitivity": row["sensitivity"],
            "source_authority": row["source_authority"],
            "created_at": row["created_at"],
            "updated_at": row["updated_at"],
            "last_verified_at": row["last_verified_at"],
        }
        try:
            result["rank"] = row["fts_rank"]
        except IndexError:
            pass
        if include_sources:
            result["sources"] = self._sources(row["id"])
        return result

    def _sources(self, memory_id: str) -> list[dict[str, Any]]:
        with self.database.connect() as connection:
            rows = connection.execute(
                """
                SELECT ms.quote, m.id AS message_id, m.created_at,
                       c.id AS conversation_id, c.source, c.external_id,
                       c.title, c.source_uri
                FROM memory_sources ms
                JOIN messages m ON m.id = ms.message_id
                JOIN conversations c ON c.id = m.conversation_id
                WHERE ms.memory_id = ?
                ORDER BY m.created_at DESC
                """,
                (memory_id,),
            ).fetchall()
        return [dict(row) for row in rows]

    def get_source_message(self, message_id: str) -> dict[str, Any]:
        with self.database.connect() as connection:
            row = connection.execute(
                """
                SELECT m.*, c.source, c.external_id AS conversation_external_id,
                       c.title, c.source_uri, c.project
                FROM messages m JOIN conversations c ON c.id = m.conversation_id
                WHERE m.id = ?
                """,
                (message_id,),
            ).fetchone()
        if not row:
            raise KeyError(f"Message not found: {message_id}")
        return {
            "id": row["id"],
            "role": row["role"],
            "content": self.cipher.decrypt(row["content_cipher"]),
            "created_at": row["created_at"],
            "source": row["source"],
            "conversation_external_id": row["conversation_external_id"],
            "title": row["title"],
            "source_uri": row["source_uri"],
            "project": row["project"],
        }

    def set_status(self, memory_id: str, status: str) -> dict[str, Any]:
        if status not in {"confirmed", "rejected", "superseded"}:
            raise ValueError("Invalid memory status")
        now = utc_now()
        with self.database.transaction() as connection:
            cursor = connection.execute(
                """
                UPDATE memories
                SET status = ?, updated_at = ?, last_verified_at = CASE WHEN ? = 'confirmed' THEN ? ELSE last_verified_at END
                WHERE id = ?
                """,
                (status, now, status, now, memory_id),
            )
            if not cursor.rowcount:
                raise KeyError(f"Memory not found: {memory_id}")
        return self.get_memory(memory_id)

    def delete_memory(self, memory_id: str) -> bool:
        with self.database.transaction() as connection:
            cursor = connection.execute(
                "DELETE FROM memories WHERE id = ?", (memory_id,)
            )
        return bool(cursor.rowcount)

    def list_conversations(
        self, source: str | None = None, project: str | None = None, limit: int = 100
    ) -> list[dict[str, Any]]:
        clauses = []
        parameters: list[Any] = []
        if source is not None:
            clauses.append("c.source = ?")
            parameters.append(source)
        if project is not None:
            clauses.append("c.project = ?")
            parameters.append(project)
        where = f"WHERE {' AND '.join(clauses)}" if clauses else ""
        parameters.append(max(1, min(limit, 500)))
        with self.database.connect() as connection:
            rows = connection.execute(
                f"""
                SELECT c.id, c.source, c.external_id, c.project, c.title,
                       c.source_uri, c.created_at, c.updated_at, c.imported_at,
                       COUNT(m.id) AS message_count
                FROM conversations c
                LEFT JOIN messages m ON m.conversation_id = c.id
                {where}
                GROUP BY c.id
                ORDER BY c.updated_at DESC
                LIMIT ?
                """,
                parameters,
            ).fetchall()
        return [dict(row) for row in rows]

    def delete_conversation(self, conversation_id: str) -> dict[str, int | bool]:
        with self.database.transaction() as connection:
            linked_memory_ids = [
                row["memory_id"]
                for row in connection.execute(
                    """
                    SELECT DISTINCT ms.memory_id
                    FROM memory_sources ms
                    JOIN messages m ON m.id = ms.message_id
                    WHERE m.conversation_id = ?
                    """,
                    (conversation_id,),
                ).fetchall()
            ]
            message_count = connection.execute(
                "SELECT COUNT(*) FROM messages WHERE conversation_id = ?",
                (conversation_id,),
            ).fetchone()[0]
            deleted = connection.execute(
                "DELETE FROM conversations WHERE id = ?", (conversation_id,)
            ).rowcount
            orphaned_deleted = 0
            for memory_id in linked_memory_ids:
                orphaned_deleted += connection.execute(
                    """
                    DELETE FROM memories
                    WHERE id = ?
                      AND NOT EXISTS (
                          SELECT 1 FROM memory_sources WHERE memory_id = memories.id
                      )
                    """,
                    (memory_id,),
                ).rowcount
        return {
            "deleted": bool(deleted),
            "messages_deleted": message_count if deleted else 0,
            "orphaned_memories_deleted": orphaned_deleted,
        }

    def consolidate(self) -> dict[str, int]:
        with self.database.connect() as connection:
            pending = [
                row["id"]
                for row in connection.execute(
                    "SELECT id FROM messages WHERE knowledge_processed_at IS NULL ORDER BY created_at"
                ).fetchall()
            ]
        memories_added = sum(
            self._process_message(message_id) for message_id in pending
        )
        self.database.optimize()
        return {"messages_processed": len(pending), "memories_added": memories_added}

    def stats(self) -> dict[str, int]:
        with self.database.connect() as connection:
            return {
                "conversations": connection.execute(
                    "SELECT COUNT(*) FROM conversations"
                ).fetchone()[0],
                "messages": connection.execute(
                    "SELECT COUNT(*) FROM messages"
                ).fetchone()[0],
                "memories": connection.execute(
                    "SELECT COUNT(*) FROM memories"
                ).fetchone()[0],
                "confirmed": connection.execute(
                    "SELECT COUNT(*) FROM memories WHERE status = 'confirmed'"
                ).fetchone()[0],
                "candidates": connection.execute(
                    "SELECT COUNT(*) FROM memories WHERE status = 'candidate'"
                ).fetchone()[0],
            }

    def get_setting(self, key: str) -> str | None:
        with self.database.connect() as connection:
            row = connection.execute(
                "SELECT value FROM settings WHERE key = ?", (key,)
            ).fetchone()
        return row["value"] if row else None

    def set_setting(self, key: str, value: str) -> None:
        with self.database.transaction() as connection:
            connection.execute(
                "INSERT INTO settings(key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
                (key, value),
            )

    def delete_setting(self, key: str) -> None:
        with self.database.transaction() as connection:
            connection.execute("DELETE FROM settings WHERE key = ?", (key,))

    @staticmethod
    def report_dict(report: ImportReport) -> dict[str, int]:
        return asdict(report)
