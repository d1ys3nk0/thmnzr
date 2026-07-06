package input

import "testing"

func TestParsePhoenixURL(t *testing.T) {
	got := Parse("https://phoenix.example.com/projects/UHJvamVjdDox/spans/0123456789abcdef")
	if got.ProjectID != "UHJvamVjdDox" {
		t.Fatalf("project id = %q", got.ProjectID)
	}
	if got.Server != "https://phoenix.example.com" {
		t.Fatalf("server = %q", got.Server)
	}
	if got.ProjectIDDecoded != "Project:1" {
		t.Fatalf("decoded project id = %q", got.ProjectIDDecoded)
	}
	if got.SpanID != "0123456789abcdef" {
		t.Fatalf("span id = %q", got.SpanID)
	}
}

func TestParseTraceURL(t *testing.T) {
	got := Parse("open https://phoenix.example.com/projects/default/traces/6eee3b57c1bf0ea5db5eae9d56362bdc")
	if got.ProjectID != "default" {
		t.Fatalf("project id = %q", got.ProjectID)
	}
	if got.TraceID != "6eee3b57c1bf0ea5db5eae9d56362bdc" {
		t.Fatalf("trace id = %q", got.TraceID)
	}
}

func TestParsePhoenixSpanURLWithTraceIDPath(t *testing.T) {
	got := Parse("https://phoenix.example.com/projects/default/spans/6eee3b57c1bf0ea5db5eae9d56362bdc?selectedSpanNodeId=U3BhbjoyMjM=")
	if got.TraceID != "6eee3b57c1bf0ea5db5eae9d56362bdc" {
		t.Fatalf("trace id = %q", got.TraceID)
	}
	if got.SpanID != "" {
		t.Fatalf("span id = %q", got.SpanID)
	}
}

func TestParseRawTraceID(t *testing.T) {
	got := Parse("6EEE3B57C1BF0EA5DB5EAE9D56362BDC")
	if got.TraceID != "6eee3b57c1bf0ea5db5eae9d56362bdc" {
		t.Fatalf("trace id = %q", got.TraceID)
	}
}
