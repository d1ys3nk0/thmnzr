from __future__ import annotations

import base64
import re
from urllib.parse import urlparse

TRACE_ID_RE = re.compile(r"\b([a-f0-9]{32})\b")
SPAN_ID_RE = re.compile(r"\b([a-f0-9]{16})\b")


def _decode_base64(raw: str) -> str:
    missing = 4 - (len(raw) % 4)
    if missing < 4:
        raw += "=" * missing
    return base64.b64decode(raw).decode("utf-8", errors="replace")


def parse_phoenix_url(url: str) -> dict:
    parsed = urlparse(url)
    path = parsed.path.rstrip("/")
    segments = [s for s in path.split("/") if s]

    result = {
        "url": url,
        "project_id": None,
        "trace_id": None,
        "span_id": None,
    }

    try:
        idx = segments.index("projects")
        if idx + 1 < len(segments):
            result["project_id"] = segments[idx + 1]
    except ValueError:
        pass

    try:
        idx = segments.index("spans")
        if idx + 1 < len(segments):
            result["span_id"] = segments[idx + 1]
    except ValueError:
        pass

    try:
        idx = segments.index("traces")
        if idx + 1 < len(segments):
            result["trace_id"] = segments[idx + 1]
    except ValueError:
        trace_matches = TRACE_ID_RE.findall(path)
        if trace_matches:
            result["trace_id"] = trace_matches[0]
        else:
            for seg in segments:
                if len(seg) == 32 and all(c in "0123456789abcdef" for c in seg):
                    result["trace_id"] = seg
                    break

    if result["project_id"]:
        try:
            result["project_id_decoded"] = _decode_base64(result["project_id"])
        except Exception:
            result["project_id_decoded"] = result["project_id"]
    else:
        result["project_id_decoded"] = None

    return result


def extract_trace_id(text: str) -> str | None:
    match = TRACE_ID_RE.search(text)
    return match.group(1) if match else None


def extract_span_id(text: str) -> str | None:
    match = SPAN_ID_RE.search(text)
    return match.group(1) if match else None


def parse_input(raw: str) -> dict:
    raw = raw.strip()
    url_match = re.search(r"https?://[^\s]+", raw)
    if url_match:
        return parse_phoenix_url(url_match.group(0))

    trace_id = extract_trace_id(raw)
    if trace_id:
        return {"url": None, "project_id": None, "trace_id": trace_id, "span_id": None, "project_id_decoded": None}

    return {"url": None, "project_id": None, "trace_id": None, "span_id": None, "project_id_decoded": None}
