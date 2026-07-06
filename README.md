# Trace Humanizer

`thmnzr` converts Phoenix trace data into compact plain text, Markdown, or JSON.

It is intended for developers and AI agents that need to inspect LLM, tool, and
agent traces without clicking through the Phoenix UI.

Agents should follow documentation `docs/AGENT_USAGE.md`.

## Features

- Accepts a Phoenix trace URL, span URL, or raw trace ID.
- Fetches spans from the Phoenix HTTP API.
- Renders dense plain output for AI agent consumption by default.
- Supports human-readable Markdown with an ASCII tree and valid JSON for `jq`.
- Optionally shows LLM/tool inputs, outputs, and model attributes.
- Deduplicates repeated LLM messages across chronological LLM spans.
- Runs as a local Go binary or through `docker run`.

## Requirements

- Go `1.26` for local builds.
- Phoenix server API access.
- A Phoenix project ID, either in the URL or via `--project-id`.

Optional environment variables:

- `PHOENIX_API_KEY`: used when `--api-key` is omitted.
- `PHOENIX_COLLECTOR_ENDPOINT`: used as the server URL when `--server` is omitted.

If no server is configured, `thmnzr` uses `http://localhost:6006`.

## Local Usage

Build the binary:

```bash
task build
```

Run from the workspace:

```bash
go run ./cmd/thmnzr --project-id default 6eee3b57c1bf0ea5db5eae9d56362bdc
```

Run an installed or built binary:

```bash
./bin/thmnzr --server http://localhost:6006 --project-id default 6eee3b57c1bf0ea5db5eae9d56362bdc
```

Use a full Phoenix URL:

```bash
thmnzr 'http://localhost:6006/projects/default/traces/6eee3b57c1bf0ea5db5eae9d56362bdc'
```

Focus through a span URL:

```bash
thmnzr 'http://localhost:6006/projects/default/spans/0123456789abcdef'
```

Save output:

```bash
thmnzr --project-id default 6eee3b57c1bf0ea5db5eae9d56362bdc --save trace.md
```

Use dense plain output for AI agents:

```bash
thmnzr --project-id default --format plain 6eee3b57c1bf0ea5db5eae9d56362bdc
```

Use human-readable Markdown with an ASCII tree:

```bash
thmnzr --project-id default --format markdown 6eee3b57c1bf0ea5db5eae9d56362bdc
```

Use JSON for filtering with `jq`:

```bash
thmnzr --project-id default --format json 6eee3b57c1bf0ea5db5eae9d56362bdc | jq '.spans[].name'
```

Generate the output filename from the trace ID:

```bash
thmnzr --project-id default 6eee3b57c1bf0ea5db5eae9d56362bdc --save
```

## CLI Reference

```text
Usage:
  thmnzr [options] TRACE_URL_OR_ID

Options:
  -h, --help                 Show help.
      --server URL           Phoenix server URL.
      --api-key KEY          Phoenix API key.
      --project-id ID        Project ID if it is not present in the input URL.
  -i, --inputs               Show tool/LLM inputs.
  -o, --outputs              Show tool/LLM outputs.
  -f, --format FORMAT        Output format: plain, markdown, or json. Defaults to plain.
      --show-attrs           Show input/model attributes for spans.
      --truncate             Truncate long messages.
      --no-dedup             Disable LLM message deduplication.
  -s, --save [FILE]          Save output to FILE. Without FILE, writes TRACE_ID with a format extension.
```

## Docker Usage

The GitHub workflow publishes:

- `ghcr.io/d1ys3nk0/thmnzr:latest`
- `ghcr.io/d1ys3nk0/thmnzr:<short-sha>`

Use Docker against a remote Phoenix server:

```bash
docker run --rm -i \
  -e PHOENIX_API_KEY \
  ghcr.io/d1ys3nk0/thmnzr:latest \
  thmnzr --server https://phoenix.example.com --project-id default 6eee3b57c1bf0ea5db5eae9d56362bdc
```

Use Docker against Phoenix running on the host machine:

```bash
docker run --rm -i \
  ghcr.io/d1ys3nk0/thmnzr:latest \
  thmnzr --server http://host.docker.internal:6006 --project-id default 6eee3b57c1bf0ea5db5eae9d56362bdc
```

Save output to the current directory:

```bash
docker run --rm -i \
  -v "$PWD:$PWD" \
  -w "$PWD" \
  ghcr.io/d1ys3nk0/thmnzr:latest \
  thmnzr --server http://host.docker.internal:6006 --project-id default 6eee3b57c1bf0ea5db5eae9d56362bdc --save
```

Because the image does not override the entrypoint, CI job scripts can call
`thmnzr` directly after selecting the image.

## Output

`thmnzr` prints dense plain text to stdout by default:

- trace title
- total time
- total tokens when available
- start and finish timestamps when available
- span hierarchy with timing, token, status, kind, and short span IDs
- selected span inputs, outputs, model names, and deduplicated LLM messages when enabled

Use `-f markdown` or `--format markdown` for human-readable Markdown with an
ASCII tree. Use `-f json` or `--format json` for valid JSON that can be filtered
with tools such as `jq`.

Errors are printed to stderr and return a non-zero exit code.

Exit codes:

- `0`: success
- `1`: runtime failure, such as Phoenix fetch errors or no spans found
- `2`: invalid CLI usage

## Troubleshooting

`could not extract project ID`

: Pass a full Phoenix URL containing `/projects/{project}` or add
  `--project-id`.

`could not extract trace ID`

: Pass a trace URL or raw 32-character trace ID. Span URLs are supported when
  Phoenix can resolve the span to its trace.

`phoenix returned 401`

: Set `PHOENIX_API_KEY` or pass `--api-key`.

`no spans found`

: Confirm the project ID and trace ID belong to the same Phoenix project.

Docker cannot reach local Phoenix

: Use `--server http://host.docker.internal:6006` on Docker Desktop. On Linux,
  use the host gateway address supported by your Docker setup.

## Development

Useful commands:

```bash
task fmt
task check
task build
task docker-build
```

Repository: <http://github.com/d1ys3nk0/thmnzr>
