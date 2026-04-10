from __future__ import annotations

import argparse
import sys
from pathlib import Path

from .client import PhoenixClient
from .formatter import format_markdown
from .span_tree import (
    build_tree,
    find_llm_spans_chronological,
    deduplicate_messages,
    flatten_tree,
    get_id,
    get_trace_id,
)
from .url_parser import parse_input


def main():
    parser = argparse.ArgumentParser(description="Convert Phoenix traces to human-readable markdown")
    parser.add_argument("trace_url", help="Phoenix trace URL or trace ID")
    parser.add_argument("--server", default="http://localhost:6007", help="Phoenix server URL")
    parser.add_argument("--api-key", default=None, help="Phoenix API key")
    parser.add_argument("--show-outputs", "-o", action="store_true", help="Show tool/LLM outputs (short: -o)")
    parser.add_argument("--show-inputs", action="store_true", default=True, help="Show inputs (default: true)")
    parser.add_argument("--show-attrs", action="store_true", help="Show all attributes")
    parser.add_argument("--truncate", action="store_true", help="Truncate long messages")
    parser.add_argument("--no-dedup", action="store_true", help="Disable LLM message deduplication")
    parser.add_argument("--project-id", help="Project ID (if not in URL)")
    parser.add_argument("--save", "-s", nargs="?", const="", help="Save output to file (short: -s). If omitted, prints to stdout. If -s is given without argument, generates filename from trace ID.")
    args = parser.parse_args()

    parsed = parse_input(args.trace_url)
    trace_id = parsed.get("trace_id")
    span_id = parsed.get("span_id")
    project_id = parsed.get("project_id") or args.project_id

    if not project_id:
        print("Error: Could not extract project ID. Use --project-id or provide full URL.", file=sys.stderr)
        sys.exit(1)

    client = PhoenixClient(args.server, args.api_key)
    try:
        if span_id:
            span = client.get_span(project_id, span_id)
            if not span:
                print(f"Error: Span {span_id} not found", file=sys.stderr)
                sys.exit(1)
            real_trace_id = get_trace_id(span)
            if real_trace_id:
                trace_id = real_trace_id

        if not trace_id:
            print("Error: Could not extract trace ID from input", file=sys.stderr)
            sys.exit(1)

        spans = client.get_trace_spans(project_id, trace_id)
    except Exception as e:
        print(f"Error fetching spans: {e}", file=sys.stderr)
        sys.exit(1)
    finally:
        client.close()

    if not spans:
        print("Error: No spans found", file=sys.stderr)
        sys.exit(1)

    tree = build_tree(spans)

    llm_spans = find_llm_spans_chronological(spans)
    if args.no_dedup or not llm_spans:
        new_messages_map = {}
    else:
        new_messages_map = deduplicate_messages(llm_spans)

    flat = flatten_tree(tree["children"], "__root__")

    trace_ids_in_tree = set()
    for span, _ in flat:
        tid = get_trace_id(span)
        if tid:
            trace_ids_in_tree.add(tid)

    if trace_ids_in_tree:
        title = f"Trace {trace_ids_in_tree.pop()}"
    else:
        title = "Agent Trace"

    md = format_markdown(
        tree,
        flat,
        new_messages_map,
        title=title,
        show_attrs=args.show_attrs or args.show_outputs,
        show_outputs=args.show_outputs,
        show_inputs=args.show_inputs,
        span_id_filter=span_id,
        truncate=args.truncate,
    )

    output_dest = args.save
    if output_dest is None:
        print(md)
    else:
        output_file = output_dest if output_dest else f"{trace_id}.md"
        Path(output_file).write_text(md)
        print(f"Wrote {len(md)} chars to {output_file}")


if __name__ == "__main__":
    main()
