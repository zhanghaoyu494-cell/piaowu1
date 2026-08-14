from __future__ import annotations

import re
from dataclasses import dataclass

from .security import redact_sensitive


@dataclass(frozen=True, slots=True)
class MemoryCandidate:
    content: str
    kind: str
    confidence: float
    status: str
    sensitivity: str
    source_authority: str


RULES = (
    (
        "todo",
        re.compile(r"\bTODO\b|待办|下一步|后续(?:需要|计划)|需要继续", re.IGNORECASE),
    ),
    (
        "decision",
        re.compile(
            r"决定|确定(?:采用|使用)|最终选择|统一使用|\bwe (?:decided|will use)\b",
            re.IGNORECASE,
        ),
    ),
    (
        "preference",
        re.compile(
            r"我(?:喜欢|希望|倾向)|请不要|以后不要|优先(?:使用|选择)|\bI prefer\b",
            re.IGNORECASE,
        ),
    ),
    (
        "constraint",
        re.compile(
            r"必须|不得|不能|只允许|要求(?:是|使用)|限制|\bmust\b|\bshould not\b",
            re.IGNORECASE,
        ),
    ),
    (
        "solution",
        re.compile(
            r"解决(?:方法|方案)|修复(?:方法|方案)|根因|原因是|通过.+解决|\b(?:fixed|resolved) by\b",
            re.IGNORECASE,
        ),
    ),
    (
        "fact",
        re.compile(
            r"架构是|接口是|数据库是|项目使用|部署在|路径是|\bproject uses\b",
            re.IGNORECASE,
        ),
    ),
)

EXPLICIT_MEMORY = re.compile(
    r"^(?:请)?记住(?:这件事|这一点|：|:)?|^remember(?: that)?\b", re.IGNORECASE
)
NOISE = re.compile(
    r"^(?:好的|收到|明白|谢谢|ok|okay|yes|no)[！!。.\s]*$", re.IGNORECASE
)
ASSISTANT_PROCESS = re.compile(
    r"^(?:接下来|然后)?我(?:也)?(?:会|将|准备|正在)", re.IGNORECASE
)
INCOMPLETE_INTRODUCTION = re.compile(r"[:：]\s*$")


class HeuristicExtractor:
    def extract(self, content: str, role: str) -> list[MemoryCandidate]:
        candidates = []
        in_code_block = False
        segments = re.split(r"[\r\n]+|(?<=[。！？!?])", content)
        for segment in segments:
            stripped = segment.strip(" \t-*#>").replace("**", "").strip()
            if "```" in stripped:
                in_code_block = not in_code_block
                continue
            if (
                in_code_block
                or len(stripped) < 6
                or len(stripped) > 600
                or NOISE.match(stripped)
                or INCOMPLETE_INTRODUCTION.search(stripped)
                or (role == "assistant" and ASSISTANT_PROCESS.search(stripped))
            ):
                continue

            explicit = role == "user" and bool(EXPLICIT_MEMORY.search(stripped))
            matched_kind = "fact" if explicit else None
            if not matched_kind:
                for kind, pattern in RULES:
                    if pattern.search(stripped):
                        matched_kind = kind
                        break
            if not matched_kind:
                continue

            redaction = redact_sensitive(stripped)
            if redaction.secret_detected:
                continue
            authority = (
                role if role in {"user", "assistant", "system", "tool"} else "unknown"
            )
            confidence = (
                0.78 if role == "user" else 0.58 if role == "assistant" else 0.65
            )
            candidates.append(
                MemoryCandidate(
                    content=redaction.text,
                    kind=matched_kind,
                    confidence=confidence,
                    status="confirmed" if explicit else "candidate",
                    sensitivity=redaction.sensitivity,
                    source_authority=authority,
                )
            )
        return candidates
