from __future__ import annotations

import json
from collections import defaultdict
from typing import Any


def get_id(span: dict) -> str:
    return span.get("context", {}).get("span_id", "") or span.get("span_id", "") or span.get("id", "")


def get_parent_id(span: dict) -> str | None:
    return span.get("parent_id") or span.get("context", {}).get("parent_span_id")


def get_trace_id(span: dict) -> str:
    return span.get("context", {}).get("trace_id", "") or span.get("trace_id", "") or span.get("traceId", "")


def get_span_kind(span: dict) -> str:
    return (
        span.get("openinference", {}).get("span", {}).get("kind", "")
        or span.get("span_kind", "")
        or span.get("kind", "")
    )


def get_name(span: dict) -> str:
    return span.get("name", "") or ""


def get_status_code(span: dict) -> str:
    return span.get("status_code", "") or "UNSET"


def get_start_time(span: dict) -> str:
    return span.get("start_time", "") or ""


def get_end_time(span: dict) -> str:
    return span.get("end_time", "") or ""


def get_duration_ms(span: dict) -> float | None:
    start = span.get("start_time", "") or ""
    end = span.get("end_time", "") or ""
    if start and end:
        try:
            from datetime import datetime

            fmt = "%Y-%m-%dT%H:%M:%S.%fZ"
            t0 = datetime.fromisoformat(start.replace("Z", "+00:00").replace("+00:00", ""))
            t1 = datetime.fromisoformat(end.replace("Z", "+00:00").replace("+00:00", ""))
            return (t1 - t0).total_seconds() * 1000
        except Exception:
            pass
    return None


def get_attributes(span: dict) -> dict:
    return span.get("attributes", {}) or {}


def get_input(span: dict) -> Any:
    attrs = get_attributes(span)
    return attrs.get("input", attrs.get("llm.input", attrs.get("input.value")))


def get_output(span: dict) -> Any:
    attrs = get_attributes(span)
    return attrs.get("output", attrs.get("llm.output", attrs.get("output.value")))


def build_tree(spans: list[dict]) -> dict[str, list[dict]]:
    children: dict[str, list[dict]] = defaultdict(list)
    nodes: dict[str, dict] = {}

    for span in spans:
        span_id = get_id(span)
        parent_id = get_parent_id(span)
        if span_id:
            nodes[span_id] = span

    for span in spans:
        span_id = get_id(span)
        parent_id = get_parent_id(span)
        if parent_id and parent_id in nodes:
            children[parent_id].append(span)
        else:
            children["__root__"].append(span)

    return {"children": children, "nodes": nodes}


def get_llm_messages(span: dict) -> list | None:
    inp = get_input(span)
    if isinstance(inp, str):
        try:
            inp = json.loads(inp)
        except (json.JSONDecodeError, ValueError):
            pass
    if isinstance(inp, dict):
        if "messages" in inp:
            return inp["messages"]
        if "prompt" in inp:
            prompt = inp["prompt"]
            if isinstance(prompt, list):
                return prompt
    if isinstance(inp, list):
        return inp
    return None


def find_llm_spans_chronological(spans: list[dict]) -> list[tuple[int, dict]]:
    llm_spans = []
    for i, span in enumerate(spans):
        kind = get_span_kind(span).upper()
        name = get_name(span).upper()
        if "LLM" in kind or "CHAT" in name or "COMPLETION" in name or "MESSAGE" in name:
            llm_spans.append((i, span))
    return llm_spans


def deduplicate_messages(llm_spans: list[tuple[int, dict]]) -> dict[int, list]:
    result: dict[int, list] = {}
    all_prev_messages: list = []

    for idx, span in llm_spans:
        messages = get_llm_messages(span)
        if messages is None:
            result[idx] = []
            continue

        new_messages = []
        for msg in messages:
            if msg not in all_prev_messages:
                new_messages.append(msg)
                all_prev_messages.append(msg)

        result[idx] = new_messages

    return result


def flatten_tree(
    children: dict[str, list[dict]],
    node_id: str,
    depth: int = 0,
    visited: set | None = None,
) -> list[tuple[dict, int]]:
    if visited is None:
        visited = set()
    result: list[tuple[dict, int]] = []
    for child in children.get(node_id, []):
        cid = get_id(child)
        if cid in visited:
            continue
        visited.add(cid)
        result.append((child, depth))
        result.extend(flatten_tree(children, cid, depth + 1, visited))
    return result
