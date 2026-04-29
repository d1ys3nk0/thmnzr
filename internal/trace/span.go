package trace

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Span map[string]any

func GetID(span Span) string {
	return firstString(
		nestedString(span, "context", "span_id"),
		stringValue(span["span_id"]),
		stringValue(span["id"]),
	)
}

func GetParentID(span Span) string {
	return firstString(
		stringValue(span["parent_id"]),
		nestedString(span, "context", "parent_span_id"),
	)
}

func GetTraceID(span Span) string {
	return firstString(
		nestedString(span, "context", "trace_id"),
		stringValue(span["trace_id"]),
		stringValue(span["traceId"]),
	)
}

func GetSpanKind(span Span) string {
	return firstString(
		nestedString(span, "openinference", "span", "kind"),
		stringValue(span["span_kind"]),
		stringValue(span["kind"]),
	)
}

func GetName(span Span) string {
	return stringValue(span["name"])
}

func GetStatusCode(span Span) string {
	status := stringValue(span["status_code"])
	if status == "" {
		return "UNSET"
	}
	return status
}

func GetAttributes(span Span) map[string]any {
	attrs, ok := span["attributes"].(map[string]any)
	if !ok || attrs == nil {
		return map[string]any{}
	}
	return attrs
}

func GetInput(span Span) any {
	attrs := GetAttributes(span)
	return firstAny(attrs["input"], attrs["llm.input"], attrs["input.value"])
}

func GetOutput(span Span) any {
	attrs := GetAttributes(span)
	if output := firstAny(attrs["output"], attrs["llm.output"], attrs["output.value"]); output != nil {
		return output
	}
	return outputMessagesContent(flatAttributeMessages(attrs, "llm.output_messages."))
}

func GetDurationMS(span Span) (float64, bool) {
	start := stringValue(span["start_time"])
	end := stringValue(span["end_time"])
	if start == "" || end == "" {
		return 0, false
	}
	t0, err := parseTime(start)
	if err != nil {
		return 0, false
	}
	t1, err := parseTime(end)
	if err != nil {
		return 0, false
	}
	return float64(t1.Sub(t0).Microseconds()) / 1000, true
}

func GetTokenCount(span Span) int {
	attrs := GetAttributes(span)
	return intValue(firstAny(attrs["llm.token_count.total"], attrs["token_count"]))
}

func GetLLMMessages(span Span) []map[string]any {
	attrs := GetAttributes(span)
	if messages := flatAttributeMessages(attrs, "llm.input_messages."); len(messages) > 0 {
		return messages
	}

	input := GetInput(span)
	if raw, ok := input.(string); ok {
		var decoded any
		if json.Unmarshal([]byte(raw), &decoded) == nil {
			input = decoded
		}
	}
	if obj, ok := input.(map[string]any); ok {
		if messages, ok := asMessageList(obj["messages"]); ok {
			return messages
		}
		if messages, ok := asMessageList(obj["prompt"]); ok {
			return messages
		}
	}
	if messages, ok := asMessageList(input); ok {
		return messages
	}
	return nil
}

func flatAttributeMessages(attrs map[string]any, prefix string) []map[string]any {
	type indexedMessage struct {
		index   int
		message map[string]any
	}

	byIndex := map[int]map[string]any{}
	for key, value := range attrs {
		remainder, ok := strings.CutPrefix(key, prefix)
		if !ok {
			continue
		}
		indexRaw, fieldPath, ok := strings.Cut(remainder, ".message.")
		if !ok {
			continue
		}
		index, err := strconv.Atoi(indexRaw)
		if err != nil {
			continue
		}
		message := byIndex[index]
		if message == nil {
			message = map[string]any{}
			byIndex[index] = message
		}
		message[fieldPath] = value
	}

	messages := make([]indexedMessage, 0, len(byIndex))
	for index, message := range byIndex {
		messages = append(messages, indexedMessage{index: index, message: message})
	}
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].index < messages[j].index
	})

	result := make([]map[string]any, 0, len(messages))
	for _, item := range messages {
		result = append(result, item.message)
	}
	return result
}

func outputMessagesContent(messages []map[string]any) any {
	if len(messages) == 0 {
		return nil
	}
	if len(messages) == 1 {
		if content := messages[0]["content"]; content != nil {
			return content
		}
	}
	return messages
}

func stringValue(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return ""
	}
}

func intValue(v any) int {
	switch value := v.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		i, _ := value.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(value)
		return i
	default:
		return 0
	}
}

func nestedString(root map[string]any, path ...string) string {
	var current any = root
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = obj[key]
	}
	return stringValue(current)
}

func firstString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstAny(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func parseTime(raw string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, raw)
}

func asMessageList(value any) ([]map[string]any, bool) {
	rawList, ok := value.([]any)
	if !ok {
		return nil, false
	}
	messages := make([]map[string]any, 0, len(rawList))
	for _, item := range rawList {
		message, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		messages = append(messages, message)
	}
	return messages, true
}
