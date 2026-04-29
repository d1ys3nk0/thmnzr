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
