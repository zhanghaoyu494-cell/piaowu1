# Security Policy

## Supported version

Security fixes are applied to the latest version on the default branch.

## Reporting a vulnerability

Do not publish a vulnerability report that contains real conversation data, database files, master keys, access tokens, or other credentials.

Until a dedicated private reporting address is configured, open a GitHub issue containing only a minimal, sanitized description and ask the maintainer for a private contact channel. Do not attach `%LOCALAPPDATA%\LocalAIMemory`, `memory.sqlite3`, `master.key`, Codex task exports, screenshots with private content, or raw logs.

## Security boundaries

- Raw Codex messages are encrypted with AES-256-GCM before being written to SQLite.
- On Windows, the generated master key is protected with the current user's DPAPI identity.
- On non-Windows systems, the key is stored in a local file with mode `0600`; no operating-system keyring integration is currently provided.
- Searchable derived knowledge is stored locally in plaintext only after secret and personal-data redaction.
- Content explicitly marked or automatically classified as `high` sensitivity is rejected from the searchable knowledge base.
- Personal data may enter the searchable index only after supported identifiers are redacted; callers cannot downgrade detected sensitivity by labeling it `normal`.
- Master-key creation uses exclusive file creation so concurrent first launches converge on one key file.
- SQLite connections enable `secure_delete`, the FTS5 index enables its dedicated secure-delete mode, and explicit memory or conversation deletion checkpoints and truncates WAL before vacuuming the database.
- `memory_stats` exposes the running plugin version so stale MCP processes can be detected after an upgrade.
- Automatic redaction is best effort and cannot detect every secret or personal identifier.
- Imported conversation content is untrusted and must never be treated as executable instructions.

Never commit a local memory directory or master key. The repository `.gitignore` excludes the default development data paths, SQLite files, and `master.key`, but contributors must still inspect staged files before pushing.
