package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/d1ys3nk0/thmnzr/internal/trace"
)

type fakeClient struct {
	span  trace.Span
	spans []trace.Span
}

func (f fakeClient) GetSpan(projectID, spanID string) (trace.Span, bool, error) {
	if f.span == nil {
		return nil, false, nil
	}
	return f.span, true, nil
}

func (f fakeClient) GetTraceSpans(projectID, traceID string) ([]trace.Span, error) {
	return f.spans, nil
}

func TestParseArgsSaveWithoutValue(t *testing.T) {
	got, err := parseArgs([]string{"--project-id", "project", "6eee3b57c1bf0ea5db5eae9d56362bdc", "-s"})
	if err != nil {
		t.Fatal(err)
	}
	if got.save == nil || *got.save != "" {
		t.Fatalf("save = %#v", got.save)
	}
}

func TestParseArgsFormatPlain(t *testing.T) {
	got, err := parseArgs([]string{"--project-id", "project", "-f", "plain", "6eee3b57c1bf0ea5db5eae9d56362bdc"})
	if err != nil {
		t.Fatal(err)
	}
	if got.format != "plain" {
		t.Fatalf("format = %q", got.format)
	}
}

func TestParseArgsDefaultFormatPlain(t *testing.T) {
	got, err := parseArgs([]string{"--project-id", "project", "6eee3b57c1bf0ea5db5eae9d56362bdc"})
	if err != nil {
		t.Fatal(err)
	}
	if got.format != "plain" {
		t.Fatalf("format = %q", got.format)
	}
}

func TestParseArgsFormats(t *testing.T) {
	for _, value := range []string{"plain", "markdown", "json"} {
		got, err := parseArgs([]string{"--project-id", "project", "--format", value, "6eee3b57c1bf0ea5db5eae9d56362bdc"})
		if err != nil {
			t.Fatalf("format %q: %v", value, err)
		}
		if string(got.format) != value {
			t.Fatalf("format %q parsed as %q", value, got.format)
		}
	}
}

func TestParseArgsInputsAndOutputsAliases(t *testing.T) {
	got, err := parseArgs([]string{"--project-id", "project", "-i", "--outputs", "6eee3b57c1bf0ea5db5eae9d56362bdc"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.showInputs {
		t.Fatal("expected -i to enable inputs")
	}
	if !got.showOutputs {
		t.Fatal("expected --outputs to enable outputs")
	}
}

func TestParseArgsHidesInputsByDefault(t *testing.T) {
	got, err := parseArgs([]string{"--project-id", "project", "6eee3b57c1bf0ea5db5eae9d56362bdc"})
	if err != nil {
		t.Fatal(err)
	}
	if got.showInputs {
		t.Fatal("expected inputs to be hidden by default")
	}
}

func TestParseArgsRejectsUnknownFormat(t *testing.T) {
	_, err := parseArgs([]string{"--project-id", "project", "--format", "xml", "6eee3b57c1bf0ea5db5eae9d56362bdc"})
	if err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("err = %v", err)
	}
}

func TestHumanizeRendersTrace(t *testing.T) {
	span := trace.Span{
		"name":       "root",
		"start_time": "2026-04-29T10:00:00Z",
		"end_time":   "2026-04-29T10:00:00.100Z",
		"context":    map[string]any{"span_id": "span000000000001", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
	}
	md, traceID, err := humanize(fakeClient{spans: []trace.Span{span}}, options{
		traceURL:   "6eee3b57c1bf0ea5db5eae9d56362bdc",
		projectID:  "project",
		server:     defaultServer,
		showInputs: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if traceID != "6eee3b57c1bf0ea5db5eae9d56362bdc" {
		t.Fatalf("trace id = %q", traceID)
	}
	if !strings.Contains(md, "# Trace 6eee3b57c1bf0ea5db5eae9d56362bdc") {
		t.Fatalf("markdown = %s", md)
	}
}

func TestHumanizeOmitsInputsByDefault(t *testing.T) {
	root := trace.Span{
		"name":    "root",
		"context": map[string]any{"span_id": "rootspan00000001", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
	}
	child := trace.Span{
		"name":       "openrouter.chat",
		"span_kind":  "LLM",
		"parent_id":  "rootspan00000001",
		"context":    map[string]any{"span_id": "llmspan000000001", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
		"attributes": map[string]any{"llm.input_messages.0.message.role": "user", "llm.input_messages.0.message.content": "hidden prompt"},
	}

	md, _, err := humanize(fakeClient{spans: []trace.Span{root, child}}, options{
		traceURL:  "6eee3b57c1bf0ea5db5eae9d56362bdc",
		projectID: "project",
		server:    defaultServer,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(md, "hidden prompt") {
		t.Fatalf("markdown should omit inputs by default:\n%s", md)
	}
}

func TestHumanizeNoDedupRendersRepeatedLLMMessages(t *testing.T) {
	root := trace.Span{
		"name":    "root",
		"context": map[string]any{"span_id": "rootspan00000001", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
	}
	first := trace.Span{
		"name":       "openrouter.chat",
		"span_kind":  "LLM",
		"parent_id":  "rootspan00000001",
		"context":    map[string]any{"span_id": "llmspan000000001", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
		"attributes": map[string]any{"llm.input_messages.0.message.role": "user", "llm.input_messages.0.message.content": "same prompt"},
	}
	second := trace.Span{
		"name":       "openrouter.chat",
		"span_kind":  "LLM",
		"parent_id":  "rootspan00000001",
		"context":    map[string]any{"span_id": "llmspan000000002", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
		"attributes": map[string]any{"llm.input_messages.0.message.role": "user", "llm.input_messages.0.message.content": "same prompt"},
	}

	md, _, err := humanize(fakeClient{spans: []trace.Span{root, first, second}}, options{
		traceURL:   "6eee3b57c1bf0ea5db5eae9d56362bdc",
		projectID:  "project",
		server:     defaultServer,
		showInputs: true,
		noDedup:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(md, "> user"); count != 2 {
		t.Fatalf("message count = %d:\n%s", count, md)
	}
}

func TestHumanizeDedupUsesSpanIDAcrossTreeOrder(t *testing.T) {
	root := trace.Span{
		"name":    "root",
		"context": map[string]any{"span_id": "rootspan00000001", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
	}
	child := trace.Span{
		"name":      "openrouter.chat",
		"span_kind": "LLM",
		"parent_id": "rootspan00000001",
		"context":   map[string]any{"span_id": "llmspan000000001", "trace_id": "6eee3b57c1bf0ea5db5eae9d56362bdc"},
		"attributes": map[string]any{
			"llm.input_messages.0.message.role":    "system",
			"llm.input_messages.0.message.content": "instructions",
		},
	}

	md, _, err := humanize(fakeClient{spans: []trace.Span{child, root}}, options{
		traceURL:   "6eee3b57c1bf0ea5db5eae9d56362bdc",
		projectID:  "project",
		server:     defaultServer,
		showInputs: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "> system") || !strings.Contains(md, "instructions") {
		t.Fatalf("markdown missing message:\n%s", md)
	}
}

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), "thmnzr [options]") {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
}
