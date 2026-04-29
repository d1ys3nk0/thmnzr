package trace

import "testing"

func TestBuildAndFlattenTree(t *testing.T) {
	root := Span{"name": "root", "context": map[string]any{"span_id": "a", "trace_id": "trace"}}
	child := Span{"name": "child", "parent_id": "a", "context": map[string]any{"span_id": "b"}}
	orphan := Span{"name": "orphan", "parent_id": "missing", "context": map[string]any{"span_id": "c"}}

	tree := BuildTree([]Span{root, child, orphan})
	if len(tree.Children[RootID]) != 2 {
		t.Fatalf("root children = %d", len(tree.Children[RootID]))
	}
	flat := FlattenTree(tree.Children, RootID)
	if len(flat) != 3 {
		t.Fatalf("flat length = %d", len(flat))
	}
	if GetName(flat[1].Span) != "child" || flat[1].Depth != 1 {
		t.Fatalf("unexpected child entry: %#v", flat[1])
	}
}

func TestDeduplicateMessages(t *testing.T) {
	first := Span{
		"name": "ChatCompletion",
		"attributes": map[string]any{
			"input": map[string]any{"messages": []any{
				map[string]any{"role": "user", "content": "hello"},
			}},
		},
	}
	second := Span{
		"name": "ChatCompletion",
		"attributes": map[string]any{
			"input": map[string]any{"messages": []any{
				map[string]any{"role": "user", "content": "hello"},
				map[string]any{"role": "assistant", "content": "hi"},
			}},
		},
	}

	got := DeduplicateMessages(FindLLMSpansChronological([]Span{first, second}))
	if len(got[0]) != 1 {
		t.Fatalf("first new messages = %d", len(got[0]))
	}
	if len(got[1]) != 1 || got[1][0]["role"] != "assistant" {
		t.Fatalf("second new messages = %#v", got[1])
	}
}
