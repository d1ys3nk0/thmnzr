package render

import (
	"strings"
	"testing"

	"github.com/d1ys3nk0/thmnzr/internal/trace"
)

func TestMarkdownRendersSummaryAndTree(t *testing.T) {
	root := trace.Span{
		"name":       "agent run",
		"span_kind":  "AGENT",
		"start_time": "2026-04-29T10:00:00Z",
		"end_time":   "2026-04-29T10:00:01Z",
		"context":    map[string]any{"span_id": "abcdef0123456789", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
		"attributes": map[string]any{"llm.token_count.total": float64(7), "input": "question"},
	}
	tree := trace.BuildTree([]trace.Span{root})
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	got := Markdown(tree, flat, Options{Title: "Trace test", ShowAttrs: true, ShowInputs: true})
	for _, want := range []string{
		"# Trace test",
		"Total time: 1.00s",
		"Total tokens: 7",
		"+-- agent run [1.00s | 7] [AGENT] abcdef01...",
		"input: question",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Focused on span") {
		t.Fatalf("markdown should not include span focus banner:\n%s", got)
	}
}

func TestMarkdownOmitsMissingTokenPlaceholder(t *testing.T) {
	root := trace.Span{
		"name":       "unknown work",
		"span_kind":  "UNKNOWN",
		"start_time": "2026-04-29T10:00:00Z",
		"end_time":   "2026-04-29T10:00:00.001Z",
		"context":    map[string]any{"span_id": "abcdef0123456789", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
	}
	tree := trace.BuildTree([]trace.Span{root})
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	got := Markdown(tree, flat, Options{Title: "Trace test"})
	if strings.Contains(got, "| ]") {
		t.Fatalf("markdown has empty token placeholder:\n%s", got)
	}
	if !strings.Contains(got, "+-- unknown work [1ms] [UNKNOWN] abcdef01...") {
		t.Fatalf("markdown missing compact metrics:\n%s", got)
	}
}

func TestMarkdownWrapsLLMMessagesInsideTree(t *testing.T) {
	root := trace.Span{
		"name":    "openrouter.chat",
		"context": map[string]any{"span_id": "abcdef0123456789"},
	}
	tree := trace.BuildTree([]trace.Span{root})
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	got := Markdown(tree, flat, Options{
		Title:     "Trace test",
		WrapWidth: 58,
		NewMessagesMap: map[string][]map[string]any{
			"abcdef0123456789": {
				{
					"role":    "user",
					"content": "First line is deliberately long enough to wrap cleanly inside the tree.\nSecond line stays prefixed too.",
				},
			},
		},
	})

	for _, want := range []string{
		"   |  -> user: First line is deliberately long enough to",
		"   |           wrap cleanly inside the tree.",
		"   |           Second line stays prefixed too.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing wrapped message line %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\nSecond line") {
		t.Fatalf("raw newline escaped tree prefix:\n%s", got)
	}
}

func TestMarkdownWrapsAttributesInsideTree(t *testing.T) {
	root := trace.Span{
		"name":       "tool.call",
		"context":    map[string]any{"span_id": "abcdef0123456789"},
		"attributes": map[string]any{"input": "this attribute value should wrap with an ascii-tree prefix on continuation"},
	}
	tree := trace.BuildTree([]trace.Span{root})
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	got := Markdown(tree, flat, Options{Title: "Trace test", ShowAttrs: true, ShowInputs: true, WrapWidth: 58})
	for _, want := range []string{
		"   |  input: this attribute value should wrap with an",
		"   |    ascii-tree prefix on continuation",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing wrapped attribute line %q:\n%s", want, got)
		}
	}
}

func TestMarkdownPlainFormatRendersDenseAgentOutput(t *testing.T) {
	root := trace.Span{
		"name":       "agent run",
		"span_kind":  "AGENT",
		"start_time": "2026-04-29T10:00:00Z",
		"end_time":   "2026-04-29T10:00:01Z",
		"context":    map[string]any{"span_id": "abcdef0123456789", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
		"attributes": map[string]any{"llm.token_count.total": float64(7), "input": "question"},
	}
	tree := trace.BuildTree([]trace.Span{root})
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	got := Markdown(tree, flat, Options{
		Title:      "Trace test",
		Format:     FormatPlain,
		ShowAttrs:  true,
		ShowInputs: true,
		NewMessagesMap: map[string][]map[string]any{
			"abcdef0123456789": {
				{"role": "user", "content": "hello"},
			},
		},
	})

	for _, want := range []string{
		"trace: Trace test",
		"total_time: 1.00s",
		"total_tokens: 7",
		`- name="agent run" duration=1.00s tokens=7 kind=AGENT span_id=abcdef01`,
		"input: question",
		"message role=user content=hello",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("plain output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "```") || strings.ContainsAny(got, "┌└├│→") {
		t.Fatalf("plain output should not include markdown fences or unicode tree characters:\n%s", got)
	}
}
