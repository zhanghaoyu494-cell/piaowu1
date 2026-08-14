# Contributing

Contributions are welcome through GitHub issues and pull requests.

## Development setup

```powershell
python -m venv .venv
.\.venv\Scripts\python -m pip install -e .
.\.venv\Scripts\python -m pip install ruff build hatchling
```

## Required checks

```powershell
.\.venv\Scripts\python -m unittest discover -s tests -v
.\.venv\Scripts\python -m ruff check .
.\.venv\Scripts\python -m build --wheel
```

Changes to the Skill or plugin package should also pass the Codex `skill-creator` and `plugin-creator` validators.

## Privacy rules

- Use synthetic conversations and fake credentials in tests.
- Never commit local databases, master keys, task exports, real logs, or screenshots containing private messages.
- Keep automatically extracted knowledge as a candidate unless the user explicitly confirms it.
- Preserve source attribution and Codex-only filtering when changing synchronization behavior.
