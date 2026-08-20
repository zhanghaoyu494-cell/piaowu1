from __future__ import annotations

import re
import unicodedata

CJK_RUN = re.compile(r"[\u3400-\u4dbf\u4e00-\u9fff]+")
WORD = re.compile(r"[a-z0-9][a-z0-9_.-]{1,}")
SEARCH_ALIASES = {
    "时区": ("utc", "时间"),
    "规范": ("统一",),
}


def normalize_text(text: str) -> str:
    normalized = unicodedata.normalize("NFKC", text).lower()
    normalized = re.sub(r"\s+", " ", normalized)
    return normalized.strip()


def search_tokens(text: str) -> list[str]:
    normalized = normalize_text(text)
    tokens = set(WORD.findall(normalized))
    for run in CJK_RUN.findall(normalized):
        if len(run) == 1:
            tokens.add(run)
        else:
            tokens.update(run[index : index + 2] for index in range(len(run) - 1))
    return sorted(token for token in tokens if token and len(token) <= 64)


def search_document(text: str) -> str:
    return " ".join(search_tokens(text))


def fts_query(text: str, operator: str = "OR", expand_aliases: bool = False) -> str:
    if operator not in {"AND", "OR"}:
        raise ValueError("FTS operator must be AND or OR")
    if expand_aliases:
        normalized = normalize_text(text)
        alias_groups = []
        for phrase, aliases in SEARCH_ALIASES.items():
            if phrase in normalized:
                escaped = [token.replace('"', '""') for token in aliases]
                group = " OR ".join(f'"{token}"' for token in escaped)
                alias_groups.append(f"({group})" if len(escaped) > 1 else group)
        return f" {operator} ".join(alias_groups)
    tokens = search_tokens(text)
    return f" {operator} ".join(
        f'"{token.replace(chr(34), chr(34) * 2)}"' for token in tokens
    )
