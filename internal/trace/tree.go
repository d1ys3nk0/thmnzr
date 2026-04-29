package trace

import (
	"encoding/json"
	"strings"
)

const RootID = "__root__"

type Tree struct {
	Children map[string][]Span
	Nodes    map[string]Span
}

type FlatSpan struct {
	Span  Span
	Depth int
}

type IndexedSpan struct {
	Index int
	Span  Span
}

func BuildTree(spans []Span) Tree {
	children := map[string][]Span{}
	nodes := map[string]Span{}

	for _, span := range spans {
		if id := GetID(span); id != "" {
			nodes[id] = span
		}
	}

	for _, span := range spans {
		parentID := GetParentID(span)
		if parentID != "" {
			if _, ok := nodes[parentID]; ok {
				children[parentID] = append(children[parentID], span)
				continue
			}
		}
		children[RootID] = append(children[RootID], span)
	}

	return Tree{Children: children, Nodes: nodes}
}

func FlattenTree(children map[string][]Span, nodeID string) []FlatSpan {
	return flattenTree(children, nodeID, 0, map[string]bool{})
}

func FindLLMSpansChronological(spans []Span) []IndexedSpan {
	result := []IndexedSpan{}
	for i, span := range spans {
		kind := strings.ToUpper(GetSpanKind(span))
		name := strings.ToUpper(GetName(span))
		if strings.Contains(kind, "LLM") ||
			strings.Contains(name, "CHAT") ||
			strings.Contains(name, "COMPLETION") ||
			strings.Contains(name, "MESSAGE") {
			result = append(result, IndexedSpan{Index: i, Span: span})
		}
	}
	return result
}

func DeduplicateMessages(spans []IndexedSpan) map[int][]map[string]any {
	result := map[int][]map[string]any{}
	seen := map[string]bool{}

	for _, indexed := range spans {
		messages := GetLLMMessages(indexed.Span)
		result[indexed.Index] = []map[string]any{}
		for _, message := range messages {
			key := stableJSON(message)
			if seen[key] {
				continue
			}
			seen[key] = true
			result[indexed.Index] = append(result[indexed.Index], message)
		}
	}

	return result
}

func flattenTree(children map[string][]Span, nodeID string, depth int, visited map[string]bool) []FlatSpan {
	result := []FlatSpan{}
	for _, child := range children[nodeID] {
		id := GetID(child)
		if id != "" {
			if visited[id] {
				continue
			}
			visited[id] = true
		}
		result = append(result, FlatSpan{Span: child, Depth: depth})
		result = append(result, flattenTree(children, id, depth+1, visited)...)
	}
	return result
}

func stableJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}
