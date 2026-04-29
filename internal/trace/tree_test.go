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

func TestGetLLMMessagesFromOpenInferenceAttributes(t *testing.T) {
	span := Span{
		"name": "openrouter.chat",
		"attributes": map[string]any{
			"llm.input_messages.1.message.role":    "user",
			"llm.input_messages.1.message.content": "extract this",
			"llm.input_messages.0.message.role":    "system",
			"llm.input_messages.0.message.content": "follow instructions",
		},
	}

	got := GetLLMMessages(span)
	if len(got) != 2 {
		t.Fatalf("messages = %#v", got)
	}
	if got[0]["role"] != "system" || got[0]["content"] != "follow instructions" {
		t.Fatalf("first message = %#v", got[0])
	}
	if got[1]["role"] != "user" || got[1]["content"] != "extract this" {
		t.Fatalf("second message = %#v", got[1])
	}
}
