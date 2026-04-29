package phoenix

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/d1ys3nk0/thmnzr/internal/trace"
)

func TestGetTraceSpansPaginatesAndAuthenticates(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("missing auth header")
		}
		if r.URL.Path != "/v1/projects/project-1/spans" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("trace_id") != "trace-1" {
			t.Fatalf("trace_id = %s", r.URL.Query().Get("trace_id"))
		}
		resp := spansResponse{Data: []trace.Span{{"name": "one"}}}
		if requests == 1 {
			resp.NextCursor = "next"
		} else if r.URL.Query().Get("cursor") != "next" {
			t.Fatalf("cursor = %s", r.URL.Query().Get("cursor"))
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret")
	spans, err := client.GetTraceSpans("project-1", "trace-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 2 {
		t.Fatalf("spans = %d", len(spans))
	}
	if requests != 2 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestGetSpanNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(spansResponse{Data: []trace.Span{}})
	}))
	defer server.Close()

	_, ok, err := NewClient(server.URL, "").GetSpan("project", "span")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected span to be missing")
	}
}
