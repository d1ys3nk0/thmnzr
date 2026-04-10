from __future__ import annotations

import json
from typing import Any


from trace_humanizer.span_tree import get_input, get_output


TRUNCATE_LEN = 200


def _truncate(s: str, limit: int = TRUNCATE_LEN) -> str:
    if len(s) <= limit:
        return s
    return s[:limit] + "..."


def _get_duration_ms(span: dict) -> float | None:
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


def _get_tokens(span: dict) -> int:
    attrs = span.get("attributes", {}) or {}
    token_count = attrs.get("llm.token_count.total") or attrs.get("token_count") or 0
    return int(token_count) if token_count else 0


def _get_status(span: dict) -> str:
    status = span.get("status_code", "") or "UNSET"
    return status if status not in ("UNSET", "OK", "") else ""


def _compute_subtree_stats(span: dict, children: dict[str, list[dict]]) -> tuple[float, int]:
    total_time = _get_duration_ms(span) or 0
    total_tokens = _get_tokens(span)

    span_id = span.get("context", {}).get("span_id", "") or span.get("span_id", "")
    for child in children.get(span_id, []):
        child_time, child_tokens = _compute_subtree_stats(child, children)
        total_time += child_time
        total_tokens += child_tokens

    return total_time, total_tokens


def _format_time_ms(ms: float) -> str:
    if ms >= 1000:
        return f"{ms/1000:.2f}s"
    return f"{ms:.0f}ms"


def _format_attrs(span: dict, show_outputs: bool = False, show_inputs: bool = True, truncate: bool = False) -> list[str]:
    attrs = span.get("attributes", {}) or {}
    lines = []

    if show_inputs:
        inp = get_input(span)
        if inp is not None:
            inp_str = json.dumps(inp, ensure_ascii=False) if isinstance(inp, (dict, list)) else str(inp)
            lines.append(f"input: {_truncate(inp_str) if truncate else inp_str}")

    if show_outputs:
        out = get_output(span)
        if out is not None:
            out_str = json.dumps(out, ensure_ascii=False) if isinstance(out, (dict, list)) else str(out)
            lines.append(f"output: {_truncate(out_str) if truncate else out_str}")

    model = attrs.get("llm.model_name") or attrs.get("model_name")
    if model:
        lines.append(f"model: {model}")

    return lines


def _render_node(
    span: dict,
    children: dict[str, list[dict]],
    depth: int,
    is_last: bool,
    prefix: str,
    new_messages_map: dict[int, list] | None,
    span_idx: int | None,
    show_attrs: bool,
    show_outputs: bool,
    show_inputs: bool,
    truncate: bool,
) -> list[str]:
    name = span.get("name", "unnamed")
    kind = span.get("span_kind", span.get("openinference", {}).get("span", {}).get("kind", ""))
    span_id = span.get("context", {}).get("span_id", "") or span.get("span_id", "")

    total_time, total_tokens = _compute_subtree_stats(span, children)
    status = _get_status(span)

    time_str = _format_time_ms(total_time)
    token_str = str(total_tokens) if total_tokens > 0 else ""
    status_str = f" {status}" if status else ""

    metrics = f"[{time_str} | {token_str}]{status_str}"

    if depth == 0:
        tree_char = "┌── "
    elif is_last:
        tree_char = "└── "
    else:
        tree_char = "├── "

    node_prefix = prefix if depth >= 2 else ""
    lines = [f"{node_prefix}{tree_char}{name} {metrics}"]
    if kind:
        lines[0] += f" [{kind}]"
    if span_id:
        lines[0] += f" {span_id[:8]}..."

    child_cont = ("   " if is_last else "│  ") if depth == 0 else prefix + ("   " if is_last else "│  ")

    if show_attrs:
        for line in _format_attrs(span, show_outputs=show_outputs, show_inputs=show_inputs, truncate=truncate):
            lines.append(f"{child_cont}│  {line}")

    if new_messages_map and span_idx is not None:
        new_msgs = new_messages_map.get(span_idx, [])
        for msg in new_msgs:
            role = msg.get("role", "unknown")
            content = msg.get("content", "")
            if isinstance(content, list):
                content_str = " ".join(c.get("text", str(c)) for c in content if isinstance(c, dict))
            else:
                content_str = str(content)
            content_display = _truncate(content_str, 150) if truncate else content_str
            lines.append(f"{child_cont}│  → {role}: {content_display}")

    node_id = span_id
    child_spans = children.get(node_id, [])
    for i, child in enumerate(child_spans):
        child_is_last = (i == len(child_spans) - 1)
        for line in _render_node(child, children, depth + 1, child_is_last, child_cont, new_messages_map, None, show_attrs, show_outputs, show_inputs, truncate):
            lines.append(line)

    return lines


def _get_trace_times(spans: list[dict]) -> tuple[str, str, float]:
    start_times = []
    end_times = []
    for span in spans:
        start = span.get("start_time", "")
        end = span.get("end_time", "")
        if start:
            start_times.append(start)
        if end:
            end_times.append(end)

    if start_times and end_times:
        from datetime import datetime
        try:
            starts = [datetime.fromisoformat(t.replace("Z", "+00:00").replace("+00:00", "")) for t in start_times]
            ends = [datetime.fromisoformat(t.replace("Z", "+00:00").replace("+00:00", "")) for t in end_times]
            earliest = min(starts)
            latest = max(ends)
            total_ms = (latest - earliest).total_seconds() * 1000
            return earliest.strftime("%Y-%m-%d %H:%M:%S"), latest.strftime("%Y-%m-%d %H:%M:%S"), total_ms
        except Exception:
            pass
    return "", "", 0


def _total_tokens_in_tree(children: dict[str, list[dict]], span: dict) -> int:
    total = _get_tokens(span)
    span_id = span.get("context", {}).get("span_id", "") or span.get("span_id", "")
    for child in children.get(span_id, []):
        total += _total_tokens_in_tree(children, child)
    return total


def format_markdown(
    tree_result: dict,
    flat_spans: list[tuple[dict, int]],
    new_messages_map: dict[int, list],
    title: str = "Agent Trace",
    show_attrs: bool = False,
    show_outputs: bool = False,
    show_inputs: bool = True,
    span_id_filter: str | None = None,
    truncate: bool = False,
) -> str:
    children = tree_result.get("children", {})

    span_idx_map: dict[str, int] = {}
    for i, (span, _) in enumerate(flat_spans):
        sid = span.get("context", {}).get("span_id", "") or span.get("span_id", "")
        span_idx_map[sid] = i

    spans = [span for span, _ in flat_spans]
    start_time, end_time, total_ms = _get_trace_times(spans)

    root_spans = children.get("__root__", [])
    total_tokens = sum(_total_tokens_in_tree(children, s) for s in root_spans)

    lines = [f"# {title}\n"]

    if span_id_filter:
        lines.append(f"*Focused on span: `{span_id_filter}`*\n")

    lines.append("Summary:")
    lines.append(f"  Total time: {_format_time_ms(total_ms)}")
    if total_tokens > 0:
        lines.append(f"  Total tokens: {total_tokens}")
    if start_time:
        lines.append(f"  Started: {start_time}")
    if end_time:
        lines.append(f"  Finished: {end_time}")
    lines.append("")

    lines.append("```")
    for i, span in enumerate(root_spans):
        is_last = (i == len(root_spans) - 1)
        span_id = span.get("context", {}).get("span_id", "") or span.get("span_id", "")
        span_idx = span_idx_map.get(span_id)
        for line in _render_node(span, children, 0, is_last, "", new_messages_map, span_idx, show_attrs, show_outputs, show_inputs, truncate):
            lines.append(line)
    lines.append("```")

    return "\n".join(lines)