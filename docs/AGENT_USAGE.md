# Agent Usage Guide

This guide defines deterministic usage patterns for AI agents invoking
`thmnzr`.

## Required Inputs

An agent needs:

- a Phoenix trace URL, span URL, or raw 32-character trace ID
- a Phoenix project ID when the input is not a full Phoenix URL
- a Phoenix server URL when the default `http://localhost:6007` is not correct
- a Phoenix API key when the server requires authentication

Never print API keys in logs, Markdown output, or task summaries.

## Preferred Local Commands

Use a full trace URL when available:

```bash
thmnzr 'http://localhost:6007/projects/default/traces/6eee3b57c1bf0ea5db5eae9d56362bdc'
```

Use a raw trace ID only with an explicit project:

```bash
thmnzr --server http://localhost:6007 --project-id default 6eee3b57c1bf0ea5db5eae9d56362bdc
```

Save a Markdown artifact with a deterministic name:

```bash
thmnzr --server http://localhost:6007 --project-id default 6eee3b57c1bf0ea5db5eae9d56362bdc --save trace.md
```

Prefer dense plain output when the result will be consumed by an AI agent:

```bash
thmnzr --server http://localhost:6007 --project-id default --format plain 6eee3b57c1bf0ea5db5eae9d56362bdc
```

Save with the trace ID as the filename:

```bash
thmnzr --server http://localhost:6007 --project-id default 6eee3b57c1bf0ea5db5eae9d56362bdc --save
```

Show outputs only when the user asks for them or they are needed for debugging:

```bash
thmnzr --project-id default --show-outputs 6eee3b57c1bf0ea5db5eae9d56362bdc
```

## Preferred Docker Commands

Remote Phoenix:

```bash
docker run --rm -i \
  -e PHOENIX_API_KEY \
  ghcr.io/d1ys3nk0/thmnzr:latest \
  thmnzr --server https://phoenix.example.com --project-id default 6eee3b57c1bf0ea5db5eae9d56362bdc
```

Phoenix on the host:

```bash
docker run --rm -i \
  ghcr.io/d1ys3nk0/thmnzr:latest \
  thmnzr --server http://host.docker.internal:6007 --project-id default 6eee3b57c1bf0ea5db5eae9d56362bdc
```

Write output to the current workspace:

```bash
docker run --rm -i \
  -v "$PWD:$PWD" \
  -w "$PWD" \
  ghcr.io/d1ys3nk0/thmnzr:latest \
  thmnzr --server http://host.docker.internal:6007 --project-id default 6eee3b57c1bf0ea5db5eae9d56362bdc --save
```

## Output Contract

On success:

- stdout contains ASCII Markdown unless `--save` is used
- with `--format plain`, stdout contains dense key/value text optimized for AI agents
- with `--save`, stdout contains `Wrote N chars to FILE`
- stderr is empty
- exit code is `0`

On failure:

- stderr starts with `Error:`
- stdout should not be treated as trace output
- exit code is `1` for runtime failures or `2` for CLI usage errors

## Safety Rules

- Treat trace content as potentially sensitive.
- Prefer `--truncate` when sending output to chat systems with limited context.
- Use `--show-outputs` only when the outputs are explicitly needed.
- Do not retry indefinitely; Phoenix/API errors should be surfaced to the user.
- Do not infer a project ID if one is not present in the URL; pass
  `--project-id`.
