from __future__ import annotations

import sqlite3
from collections.abc import Iterator
from contextlib import contextmanager
from pathlib import Path

SCHEMA = """
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    external_id TEXT NOT NULL,
    project TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    source_uri TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    imported_at TEXT NOT NULL,
    UNIQUE(source, external_id)
);

CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content_cipher BLOB NOT NULL,
    content_hash TEXT NOT NULL,
    created_at TEXT NOT NULL,
    metadata_json TEXT NOT NULL DEFAULT '{}',
    knowledge_processed_at TEXT,
    UNIQUE(conversation_id, external_id)
);

CREATE TABLE IF NOT EXISTS memories (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    normalized_content TEXT NOT NULL,
    search_terms TEXT NOT NULL,
    fingerprint TEXT NOT NULL UNIQUE,
    kind TEXT NOT NULL,
    project TEXT NOT NULL DEFAULT '',
    confidence REAL NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('candidate', 'confirmed', 'rejected', 'superseded')),
    sensitivity TEXT NOT NULL DEFAULT 'normal',
    source_authority TEXT NOT NULL DEFAULT 'unknown',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    last_verified_at TEXT,
    supersedes_id TEXT REFERENCES memories(id)
);

CREATE TABLE IF NOT EXISTS memory_sources (
    memory_id TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    quote TEXT NOT NULL,
    PRIMARY KEY(memory_id, message_id)
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_messages_pending ON messages(knowledge_processed_at);
CREATE INDEX IF NOT EXISTS idx_memories_status_project ON memories(status, project);
CREATE INDEX IF NOT EXISTS idx_memory_sources_message ON memory_sources(message_id);

CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    content,
    search_terms,
    project,
    kind,
    content='memories',
    content_rowid='rowid',
    tokenize='unicode61 remove_diacritics 2'
);

CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
    INSERT INTO memories_fts(rowid, content, search_terms, project, kind)
    VALUES (new.rowid, new.content, new.search_terms, new.project, new.kind);
END;

CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, content, search_terms, project, kind)
    VALUES ('delete', old.rowid, old.content, old.search_terms, old.project, old.kind);
END;

CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, content, search_terms, project, kind)
    VALUES ('delete', old.rowid, old.content, old.search_terms, old.project, old.kind);
    INSERT INTO memories_fts(rowid, content, search_terms, project, kind)
    VALUES (new.rowid, new.content, new.search_terms, new.project, new.kind);
END;
"""


class ClosingConnection(sqlite3.Connection):
    def __exit__(self, exc_type: object, exc_value: object, traceback: object) -> bool:
        try:
            return super().__exit__(exc_type, exc_value, traceback)
        finally:
            self.close()


class Database:
    def __init__(self, path: Path):
        self.path = path
        self.path.parent.mkdir(parents=True, exist_ok=True)
        with self.connect() as connection:
            connection.executescript(SCHEMA)

    def connect(self) -> sqlite3.Connection:
        connection = sqlite3.connect(self.path, timeout=30, factory=ClosingConnection)
        connection.row_factory = sqlite3.Row
        connection.execute("PRAGMA foreign_keys = ON")
        connection.execute("PRAGMA journal_mode = WAL")
        connection.execute("PRAGMA busy_timeout = 30000")
        return connection

    @contextmanager
    def transaction(self) -> Iterator[sqlite3.Connection]:
        connection = self.connect()
        try:
            connection.execute("BEGIN IMMEDIATE")
            yield connection
            connection.commit()
        except Exception:
            connection.rollback()
            raise
        finally:
            connection.close()

    def optimize(self) -> None:
        with self.connect() as connection:
            connection.execute(
                "INSERT INTO memories_fts(memories_fts) VALUES ('optimize')"
            )
            connection.execute("PRAGMA optimize")
