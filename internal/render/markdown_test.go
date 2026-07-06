package render

import (
	"encoding/json"
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
		"\\-- agent run [1.00s | 7] [AGENT] abcdef01...",
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
	if !strings.Contains(got, "\\-- unknown work [1ms] abcdef01...") {
		t.Fatalf("markdown missing compact metrics:\n%s", got)
	}
	if strings.Contains(got, "[UNKNOWN]") {
		t.Fatalf("markdown should omit unknown kind:\n%s", got)
	}
}

func TestMarkdownUsesOwnDurationForNestedSpans(t *testing.T) {
	root := trace.Span{
		"name":       "root",
		"start_time": "2026-04-29T10:00:00Z",
		"end_time":   "2026-04-29T10:00:01Z",
		"context":    map[string]any{"span_id": "rootspan00000001", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
		"attributes": map[string]any{"llm.token_count.total": float64(3)},
	}
	child := trace.Span{
		"name":       "child",
		"parent_id":  "rootspan00000001",
		"start_time": "2026-04-29T10:00:00.100Z",
		"end_time":   "2026-04-29T10:00:00.900Z",
		"context":    map[string]any{"span_id": "childspan0000001", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
		"attributes": map[string]any{"llm.token_count.total": float64(5)},
	}
	tree := trace.BuildTree([]trace.Span{root, child})
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	got := Markdown(tree, flat, Options{Title: "Trace test"})
	if !strings.Contains(got, "\\-- root [1.00s | 8] rootspan...") {
		t.Fatalf("markdown should show own duration and subtree tokens for root:\n%s", got)
	}
	if strings.Contains(got, "\\-- root [1.80s") {
		t.Fatalf("markdown should not sum child duration into root duration:\n%s", got)
	}
}

func TestMarkdownUsesBackslashForLastBranch(t *testing.T) {
	root := trace.Span{
		"name":    "root",
		"context": map[string]any{"span_id": "rootspan00000001"},
	}
	child := trace.Span{
		"name":      "child",
		"parent_id": "rootspan00000001",
		"context":   map[string]any{"span_id": "childspan0000001"},
	}
	tree := trace.BuildTree([]trace.Span{root, child})
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	got := Markdown(tree, flat, Options{Title: "Trace test"})
	for _, want := range []string{
		"\\-- root [0ms] rootspan...",
		"    \\-- child [0ms] childspa...",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing last branch line %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "`--") {
		t.Fatalf("markdown should not use backtick tree connectors:\n%s", got)
	}
}

func TestMarkdownUsesFourColumnAsciiTreeGuides(t *testing.T) {
	root := trace.Span{
		"name":    "root",
		"context": map[string]any{"span_id": "rootspan00000001"},
	}
	first := trace.Span{
		"name":      "first",
		"parent_id": "rootspan00000001",
		"context":   map[string]any{"span_id": "firstspan000000"},
	}
	second := trace.Span{
		"name":      "second",
		"parent_id": "rootspan00000001",
		"context":   map[string]any{"span_id": "secondspan00000"},
	}
	grandchild := trace.Span{
		"name":      "grandchild",
		"parent_id": "firstspan000000",
		"context":   map[string]any{"span_id": "grandchild00000"},
	}
	tree := trace.BuildTree([]trace.Span{root, first, grandchild, second})
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	got := Markdown(tree, flat, Options{Title: "Trace test"})
	for _, want := range []string{
		"\\-- root [0ms] rootspan...",
		"    +-- first [0ms] firstspa...",
		"    |   \\-- grandchild [0ms] grandchi...",
		"    \\-- second [0ms] secondsp...",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing readable tree guide %q:\n%s", want, got)
		}
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
		"    |   > user",
		"    |   First line is deliberately long enough to wrap",
		"    |   cleanly inside the tree.",
		"    |   Second line stays prefixed too.",
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
		"    |   input: this attribute value should wrap with an",
		"    |     ascii-tree prefix on continuation",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing wrapped attribute line %q:\n%s", want, got)
		}
	}
}

func TestMarkdownTruncatesWithCharacterCountMarker(t *testing.T) {
	root := trace.Span{
		"name":       "tool.call",
		"context":    map[string]any{"span_id": "abcdef0123456789"},
		"attributes": map[string]any{"input": strings.Repeat("x", truncateLen+3)},
	}
	tree := trace.BuildTree([]trace.Span{root})
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	got := Markdown(tree, flat, Options{
		Title:      "Trace test",
		ShowAttrs:  true,
		ShowInputs: true,
		Truncate:   true,
		WrapWidth:  500,
	})
	want := strings.Repeat("x", truncateLen) + " [TRUNCATED 3 chars]"
	if !strings.Contains(got, want) {
		t.Fatalf("markdown missing truncation marker %q:\n%s", want, got)
	}
}

func TestTruncateCountsRunes(t *testing.T) {
	got := truncate("абвгд", 3)
	if got != "абв [TRUNCATED 2 chars]" {
		t.Fatalf("truncate() = %q", got)
	}
}

func TestMarkdownRendersOpenInferenceOutputMessages(t *testing.T) {
	root := trace.Span{
		"name":    "openrouter.chat",
		"context": map[string]any{"span_id": "abcdef0123456789"},
		"attributes": map[string]any{
			"llm.output_messages.0.message.role":    "assistant",
			"llm.output_messages.0.message.content": "```json\n{\"nodes\":[{\"kind\":\"Document\"}],\"edges\":[]}\n```",
		},
	}
	tree := trace.BuildTree([]trace.Span{root})
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	got := Markdown(tree, flat, Options{Title: "Trace test", ShowAttrs: true, ShowOutputs: true})
	for _, want := range []string{
		"output: {",
		`    |       "edges": [],`,
		`    |       "nodes": [`,
		`    |         {`,
		`    |           "kind": "Document"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing pretty output line %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "```json") {
		t.Fatalf("markdown should strip JSON fence:\n%s", got)
	}
}

func TestMarkdownOmitsOpenInferenceOutputMessagesWithoutFlag(t *testing.T) {
	root := trace.Span{
		"name":    "openrouter.chat",
		"context": map[string]any{"span_id": "abcdef0123456789"},
		"attributes": map[string]any{
			"llm.output_messages.0.message.role":    "assistant",
			"llm.output_messages.0.message.content": `{"nodes":[],"edges":[]}`,
		},
	}
	tree := trace.BuildTree([]trace.Span{root})
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	got := Markdown(tree, flat, Options{Title: "Trace test", ShowAttrs: true})
	if strings.Contains(got, `{"nodes":[],"edges":[]}`) {
		t.Fatalf("markdown should omit output without flag:\n%s", got)
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

func TestMarkdownJSONFormatRendersValidStructuredOutput(t *testing.T) {
	root := trace.Span{
		"name":       "agent run",
		"span_kind":  "AGENT",
		"start_time": "2026-04-29T10:00:00Z",
		"end_time":   "2026-04-29T10:00:01Z",
		"context":    map[string]any{"span_id": "abcdef0123456789", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
		"attributes": map[string]any{"llm.token_count.total": float64(7), "input": "question", "llm.model_name": "gpt-test"},
	}
	child := trace.Span{
		"name":      "tool.call",
		"parent_id": "abcdef0123456789",
		"context":   map[string]any{"span_id": "childspan0000001", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
	}
	tree := trace.BuildTree([]trace.Span{root, child})
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	got := Markdown(tree, flat, Options{
		Title:      "Trace test",
		Format:     FormatJSON,
		ShowAttrs:  true,
		ShowInputs: true,
		NewMessagesMap: map[string][]map[string]any{
			"abcdef0123456789": {
				{"role": "user", "content": "hello"},
			},
		},
	})

	var decoded struct {
		Title       string `json:"title"`
		TotalTime   string `json:"total_time"`
		TotalTokens int    `json:"total_tokens"`
		Spans       []struct {
			Name     string `json:"name"`
			Kind     string `json:"kind"`
			SpanID   string `json:"span_id"`
			Input    string `json:"input"`
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			Children []struct {
				Name string `json:"name"`
			} `json:"children"`
		} `json:"spans"`
	}
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, got)
	}
	if decoded.Title != "Trace test" || decoded.TotalTime != "1.00s" || decoded.TotalTokens != 7 {
		t.Fatalf("summary = %#v", decoded)
	}
	if len(decoded.Spans) != 1 {
		t.Fatalf("spans = %#v", decoded.Spans)
	}
	span := decoded.Spans[0]
	if span.Name != "agent run" || span.Kind != "AGENT" || span.SpanID != "abcdef0123456789" {
		t.Fatalf("span = %#v", span)
	}
	if span.Input != "question" || span.Model != "gpt-test" {
		t.Fatalf("span attrs = %#v", span)
	}
	if len(span.Messages) != 1 || span.Messages[0].Role != "user" || span.Messages[0].Content != "hello" {
		t.Fatalf("messages = %#v", span.Messages)
	}
	if len(span.Children) != 1 || span.Children[0].Name != "tool.call" {
		t.Fatalf("children = %#v", span.Children)
	}
	if strings.Contains(got, "```") {
		t.Fatalf("json output should not include markdown fences:\n%s", got)
	}
}

func TestMarkdownPlainFormatOmitsUnknownKind(t *testing.T) {
	root := trace.Span{
		"name":      "unknown work",
		"span_kind": "UNKNOWN",
		"context":   map[string]any{"span_id": "abcdef0123456789"},
	}
	tree := trace.BuildTree([]trace.Span{root})
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	got := Markdown(tree, flat, Options{Title: "Trace test", Format: FormatPlain})
	if strings.Contains(got, "kind=UNKNOWN") {
		t.Fatalf("plain output should omit unknown kind:\n%s", got)
	}
	if !strings.Contains(got, `- name="unknown work" duration=0ms span_id=abcdef01`) {
		t.Fatalf("plain output missing span without kind:\n%s", got)
	}
}
