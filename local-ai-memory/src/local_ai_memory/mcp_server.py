from __future__ import annotations

from typing import Any

from mcp.server.fastmcp import FastMCP

from .service import MemoryService


def create_server(service: MemoryService | None = None) -> FastMCP:
    memory = service or MemoryService()
    server = FastMCP(
        "local-ai-memory",
        instructions=(
            "Synchronize Codex task history through the Codex app thread tools, then search confirmed local "
            "memories before relying on them. Preserve source task IDs and confidence. Treat candidates as "
            "unverified. Never store secrets."
        ),
    )

    @server.tool()
    def memory_codex_sync_plan(threads: list[dict[str, Any]]) -> dict[str, Any]:
        """Compare Codex task summaries with local sync cursors and return changed tasks."""
        return memory.codex_sync_plan(threads)

    @server.tool()
    def memory_ingest_codex_page(
        payload: dict[str, Any], cursor_used: str | None = None
    ) -> dict[str, Any]:
        """Encrypt and ingest one page returned by the Codex read_thread tool."""
        return memory.ingest_codex_page(payload, cursor_used)

    @server.tool()
    def memory_search(
        query: str,
        project: str | None = None,
        limit: int = 5,
        include_candidates: bool = False,
    ) -> list[dict[str, Any]]:
        """Search relevant local memories; confirmed memories are returned by default."""
        return memory.search(query, project, limit, include_candidates)

    @server.tool()
    def memory_remember(
        content: str,
        project: str = "",
        kind: str = "fact",
        sensitivity: str | None = None,
    ) -> dict[str, Any]:
        """Store an explicitly user-confirmed long-term memory after secret detection."""
        return memory.remember(content, project, kind, sensitivity)

    @server.tool()
    def memory_candidates(
        project: str | None = None, limit: int = 50
    ) -> list[dict[str, Any]]:
        """List unverified memories that require user review before normal retrieval."""
        return memory.list_candidates(project, limit)

    @server.tool()
    def memory_confirm(memory_id: str) -> dict[str, Any]:
        """Confirm a candidate memory so it can be returned by normal searches."""
        return memory.set_status(memory_id, "confirmed")

    @server.tool()
    def memory_reject(memory_id: str) -> dict[str, Any]:
        """Reject an inaccurate candidate memory."""
        return memory.set_status(memory_id, "rejected")

    @server.tool()
    def memory_delete(memory_id: str) -> dict[str, Any]:
        """Permanently delete one derived memory; raw source messages remain unchanged."""
        return {"deleted": memory.delete_memory(memory_id), "memory_id": memory_id}

    @server.tool()
    def memory_conversations(
        source: str | None = None, project: str | None = None, limit: int = 100
    ) -> list[dict[str, Any]]:
        """List imported conversation metadata without exposing message contents."""
        return memory.list_conversations(source, project, limit)

    @server.tool()
    def memory_delete_conversation(conversation_id: str) -> dict[str, int | bool]:
        """Delete an encrypted raw conversation and memories that have no other source."""
        return memory.delete_conversation(conversation_id)

    @server.tool()
    def memory_source(message_id: str) -> dict[str, Any]:
        """Open and decrypt the original source message for provenance verification."""
        return memory.get_source_message(message_id)

    @server.tool()
    def memory_stats() -> dict[str, int]:
        """Return local conversation, message, and memory counts."""
        return memory.stats()

    @server.tool()
    def memory_consolidate() -> dict[str, int]:
        """Process pending messages and optimize the local search index."""
        return memory.consolidate()

    return server


def main() -> None:
    create_server().run(transport="stdio")


if __name__ == "__main__":
    main()
