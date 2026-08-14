from __future__ import annotations

import argparse
import json
from typing import Any

from .config import Settings
from .service import MemoryService


def print_json(value: Any) -> None:
    print(json.dumps(value, ensure_ascii=False, indent=2))


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="lam", description="Local AI Memory command line"
    )
    parser.add_argument("--home", help="Override local data directory")
    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("init", help="Initialize and print storage details")
    subparsers.add_parser("stats", help="Show local database statistics")

    remember_parser = subparsers.add_parser("remember", help="Store a confirmed memory")
    remember_parser.add_argument("content")
    remember_parser.add_argument("--project", default="")
    remember_parser.add_argument("--kind", default="fact")

    search_parser = subparsers.add_parser("search", help="Search confirmed memories")
    search_parser.add_argument("query")
    search_parser.add_argument("--project")
    search_parser.add_argument("--limit", type=int, default=5)
    search_parser.add_argument("--include-candidates", action="store_true")

    candidates_parser = subparsers.add_parser(
        "candidates", help="List memories awaiting review"
    )
    candidates_parser.add_argument("--project")
    candidates_parser.add_argument("--limit", type=int, default=50)

    for command in ("confirm", "reject", "delete"):
        status_parser = subparsers.add_parser(
            command, help=f"{command.title()} a memory"
        )
        status_parser.add_argument("memory_id")

    source_parser = subparsers.add_parser(
        "source", help="Decrypt one original source message"
    )
    source_parser.add_argument("message_id")

    conversations_parser = subparsers.add_parser(
        "conversations", help="List imported conversation metadata"
    )
    conversations_parser.add_argument("--source")
    conversations_parser.add_argument("--project")
    conversations_parser.add_argument("--limit", type=int, default=100)

    delete_conversation_parser = subparsers.add_parser(
        "delete-conversation", help="Delete an encrypted raw conversation"
    )
    delete_conversation_parser.add_argument("conversation_id")

    subparsers.add_parser(
        "consolidate", help="Process pending messages and optimize indexes"
    )

    subparsers.add_parser("mcp", help="Start the MCP server over stdio")
    return parser


def main(argv: list[str] | None = None) -> None:
    arguments = build_parser().parse_args(argv)
    settings = Settings.load(arguments.home)

    if arguments.command == "mcp":
        from .mcp_server import create_server

        create_server(MemoryService(settings)).run(transport="stdio")
        return

    service = MemoryService(settings)
    if arguments.command == "init":
        print_json(
            {
                "home": str(settings.home),
                "database": str(settings.database_path),
                "raw_messages_encrypted": True,
                "stats": service.stats(),
            }
        )
    elif arguments.command == "stats":
        print_json(service.stats())
    elif arguments.command == "remember":
        print_json(
            service.remember(arguments.content, arguments.project, arguments.kind)
        )
    elif arguments.command == "search":
        print_json(
            service.search(
                arguments.query,
                arguments.project,
                arguments.limit,
                arguments.include_candidates,
            )
        )
    elif arguments.command == "candidates":
        print_json(service.list_candidates(arguments.project, arguments.limit))
    elif arguments.command == "confirm":
        print_json(service.set_status(arguments.memory_id, "confirmed"))
    elif arguments.command == "reject":
        print_json(service.set_status(arguments.memory_id, "rejected"))
    elif arguments.command == "delete":
        print_json({"deleted": service.delete_memory(arguments.memory_id)})
    elif arguments.command == "source":
        print_json(service.get_source_message(arguments.message_id))
    elif arguments.command == "conversations":
        print_json(
            service.list_conversations(
                arguments.source, arguments.project, arguments.limit
            )
        )
    elif arguments.command == "delete-conversation":
        print_json(service.delete_conversation(arguments.conversation_id))
    elif arguments.command == "consolidate":
        print_json(service.consolidate())


if __name__ == "__main__":
    main()
