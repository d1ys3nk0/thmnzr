package render

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/d1ys3nk0/thmnzr/internal/trace"
)

const truncateLen = 200
const defaultWrapWidth = 100

type Format string

const (
	FormatASCII Format = "ascii"
	FormatPlain Format = "plain"
)

type Options struct {
	Title          string
	Format         Format
	ShowAttrs      bool
	ShowOutputs    bool
	ShowInputs     bool
	Truncate       bool
	WrapWidth      int
	NewMessagesMap map[string][]map[string]any
}

func Markdown(tree trace.Tree, flatSpans []trace.FlatSpan, opts Options) string {
	if opts.Format == FormatPlain {
		return renderPlain(tree, flatSpans, opts)
	}

	children := tree.Children
	spans := make([]trace.Span, 0, len(flatSpans))
	for _, flat := range flatSpans {
		spans = append(spans, flat.Span)
	}

	startTime, endTime, totalMS := traceTimes(spans)
	rootSpans := children[trace.RootID]
	totalTokens := 0
	for _, span := range rootSpans {
		totalTokens += totalTokensInTree(children, span, map[string]bool{})
	}

	title := opts.Title
	if title == "" {
		title = "Agent Trace"
	}

	lines := []string{fmt.Sprintf("# %s\n", title), "Summary:"}
	lines = append(lines, fmt.Sprintf("  Total time: %s", formatTimeMS(totalMS)))
	if totalTokens > 0 {
		lines = append(lines, fmt.Sprintf("  Total tokens: %d", totalTokens))
	}
	if !startTime.IsZero() {
		lines = append(lines, fmt.Sprintf("  Started: %s", startTime.Format("2006-01-02 15:04:05")))
	}
	if !endTime.IsZero() {
		lines = append(lines, fmt.Sprintf("  Finished: %s", endTime.Format("2006-01-02 15:04:05")))
	}
	lines = append(lines, "", "```")

	for i, span := range rootSpans {
		lines = append(lines, renderNode(span, children, 0, i == len(rootSpans)-1, "", opts, map[string]bool{})...)
	}
	lines = append(lines, "```")

	return strings.Join(lines, "\n")
}

func renderNode(span trace.Span, children map[string][]trace.Span, depth int, isLast bool, prefix string, opts Options, visited map[string]bool) []string {
	name := trace.GetName(span)
	if name == "" {
		name = "unnamed"
	}
	kind := trace.GetSpanKind(span)
	spanID := trace.GetID(span)
	totalTime, totalTokens := computeSubtreeStats(span, children, map[string]bool{})
	status := trace.GetStatusCode(span)
	if status == "UNSET" || status == "OK" {
		status = ""
	}

	statusString := ""
	if status != "" {
		statusString = " " + status
	}
	metrics := formatMetrics(totalTime, totalTokens) + statusString

	treeChar := "+-- "
	if depth == 0 {
		treeChar = "+-- "
	} else if isLast {
		treeChar = "`-- "
	}
	nodePrefix := ""
	if depth >= 2 {
		nodePrefix = prefix
	}

	line := fmt.Sprintf("%s%s%s %s", nodePrefix, treeChar, name, metrics)
	if kind != "" {
		line += fmt.Sprintf(" [%s]", kind)
	}
	if spanID != "" {
		line += fmt.Sprintf(" %s...", prefixString(spanID, 8))
	}
	lines := []string{line}

	childCont := ""
	if depth == 0 {
		if isLast {
			childCont = "   "
		} else {
			childCont = "|  "
		}
	} else {
		childCont = prefix
		if isLast {
			childCont += "   "
		} else {
			childCont += "|  "
		}
	}

	if opts.ShowAttrs {
		for _, attrLine := range formatAttrs(span, opts.ShowOutputs, opts.ShowInputs, opts.Truncate) {
			lines = appendWrappedTreeText(lines, fmt.Sprintf("%s|  ", childCont), "", "  ", attrLine, opts.WrapWidth)
		}
	}

	if opts.NewMessagesMap != nil {
		for _, msg := range opts.NewMessagesMap[spanID] {
			role := stringValue(msg["role"])
			if role == "" {
				role = "unknown"
			}
			content := contentString(msg["content"])
			if opts.Truncate {
				content = truncate(content, 150)
			}
			label := fmt.Sprintf("-> %s: ", role)
			lines = appendWrappedTreeText(lines, fmt.Sprintf("%s|  ", childCont), label, strings.Repeat(" ", utf8.RuneCountInString(label)), content, opts.WrapWidth)
		}
	}

	if spanID != "" {
		if visited[spanID] {
			return lines
		}
		visited[spanID] = true
	}
	childSpans := children[spanID]
	for i, child := range childSpans {
		lines = append(lines, renderNode(child, children, depth+1, i == len(childSpans)-1, childCont, opts, visited)...)
	}

	return lines
}

func renderPlain(tree trace.Tree, flatSpans []trace.FlatSpan, opts Options) string {
	children := tree.Children
	spans := make([]trace.Span, 0, len(flatSpans))
	for _, flat := range flatSpans {
		spans = append(spans, flat.Span)
	}

	startTime, endTime, totalMS := traceTimes(spans)
	rootSpans := children[trace.RootID]
	totalTokens := 0
	for _, span := range rootSpans {
		totalTokens += totalTokensInTree(children, span, map[string]bool{})
	}

	title := opts.Title
	if title == "" {
		title = "Agent Trace"
	}

	lines := []string{"trace: " + title}
	lines = append(lines, "total_time: "+formatTimeMS(totalMS))
	if totalTokens > 0 {
		lines = append(lines, fmt.Sprintf("total_tokens: %d", totalTokens))
	}
	if !startTime.IsZero() {
		lines = append(lines, "started: "+startTime.Format("2006-01-02 15:04:05"))
	}
	if !endTime.IsZero() {
		lines = append(lines, "finished: "+endTime.Format("2006-01-02 15:04:05"))
	}
	lines = append(lines, "", "spans:")

	for _, span := range rootSpans {
		lines = append(lines, renderPlainNode(span, children, 0, opts, map[string]bool{})...)
	}

	return strings.Join(lines, "\n")
}

func renderPlainNode(span trace.Span, children map[string][]trace.Span, depth int, opts Options, visited map[string]bool) []string {
	name := trace.GetName(span)
	if name == "" {
		name = "unnamed"
	}
	kind := trace.GetSpanKind(span)
	spanID := trace.GetID(span)
	totalTime, totalTokens := computeSubtreeStats(span, children, map[string]bool{})
	status := trace.GetStatusCode(span)

	indent := strings.Repeat("  ", depth)
	parts := []string{
		fmt.Sprintf("name=%q", name),
		"duration=" + formatTimeMS(totalTime),
	}
	if totalTokens > 0 {
		parts = append(parts, fmt.Sprintf("tokens=%d", totalTokens))
	}
	if kind != "" {
		parts = append(parts, "kind="+kind)
	}
	if status != "" && status != "UNSET" && status != "OK" {
		parts = append(parts, "status="+status)
	}
	if spanID != "" {
		parts = append(parts, "span_id="+prefixString(spanID, 8))
	}

	lines := []string{indent + "- " + strings.Join(parts, " ")}
	detailPrefix := indent + "  "
	if opts.ShowAttrs {
		for _, attrLine := range formatAttrs(span, opts.ShowOutputs, opts.ShowInputs, opts.Truncate) {
			lines = appendWrappedTreeText(lines, detailPrefix, "", "  ", attrLine, opts.WrapWidth)
		}
	}
	if opts.NewMessagesMap != nil {
		for _, msg := range opts.NewMessagesMap[spanID] {
			role := stringValue(msg["role"])
			if role == "" {
				role = "unknown"
			}
			content := contentString(msg["content"])
			if opts.Truncate {
				content = truncate(content, 150)
			}
			label := fmt.Sprintf("message role=%s content=", role)
			lines = appendWrappedTreeText(lines, detailPrefix, label, strings.Repeat(" ", utf8.RuneCountInString(label)), content, opts.WrapWidth)
		}
	}

	if spanID != "" {
		if visited[spanID] {
			return lines
		}
		visited[spanID] = true
	}
	for _, child := range children[spanID] {
		lines = append(lines, renderPlainNode(child, children, depth+1, opts, visited)...)
	}
	return lines
}

func appendWrappedTreeText(lines []string, prefix, label, continuationIndent, content string, width int) []string {
	width = normalizedWrapWidth(width)
	text := strings.ReplaceAll(content, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	paragraphs := strings.Split(text, "\n")
	for i, paragraph := range paragraphs {
		if i == 0 {
			lines = appendWrappedLine(lines, prefix+label, prefix+continuationIndent, paragraph, width)
			continue
		}
		lines = appendWrappedLine(lines, prefix+continuationIndent, prefix+continuationIndent, paragraph, width)
	}
	return lines
}

func appendWrappedLine(lines []string, firstPrefix, nextPrefix, text string, width int) []string {
	available := width - utf8.RuneCountInString(firstPrefix)
	if available < 1 {
		available = 1
	}
	chunks := wrapText(text, available)
	if len(chunks) == 0 {
		return append(lines, firstPrefix)
	}
	lines = append(lines, firstPrefix+chunks[0])
	for _, chunk := range chunks[1:] {
		available := width - utf8.RuneCountInString(nextPrefix)
		if available < 1 {
			available = 1
		}
		for _, nestedChunk := range wrapText(chunk, available) {
			lines = append(lines, nextPrefix+nestedChunk)
		}
	}
	return lines
}

func wrapText(text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	words := strings.Fields(text)
	lines := []string{}
	current := ""
	for _, word := range words {
		for utf8.RuneCountInString(word) > width {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			head, tail := splitRunes(word, width)
			lines = append(lines, head)
			word = tail
		}
		if current == "" {
			current = word
			continue
		}
		if utf8.RuneCountInString(current)+1+utf8.RuneCountInString(word) <= width {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func splitRunes(text string, count int) (string, string) {
	runes := []rune(text)
	return string(runes[:count]), string(runes[count:])
}

func normalizedWrapWidth(width int) int {
	if width <= 0 {
		return defaultWrapWidth
	}
	if width < 40 {
		return 40
	}
	return width
}

func formatAttrs(span trace.Span, showOutputs, showInputs, shouldTruncate bool) []string {
	attrs := trace.GetAttributes(span)
	lines := []string{}
	if showInputs {
		if input := trace.GetInput(span); input != nil {
			value := valueString(input)
			if shouldTruncate {
				value = truncate(value, truncateLen)
			}
			lines = append(lines, "input: "+value)
		}
	}
	if showOutputs {
		if output := trace.GetOutput(span); output != nil {
			value := valueString(output)
			if shouldTruncate {
				value = truncate(value, truncateLen)
			}
			lines = append(lines, "output: "+value)
		}
	}
	if model := firstString(stringValue(attrs["llm.model_name"]), stringValue(attrs["model_name"])); model != "" {
		lines = append(lines, "model: "+model)
	}
	return lines
}

func computeSubtreeStats(span trace.Span, children map[string][]trace.Span, visited map[string]bool) (float64, int) {
	spanID := trace.GetID(span)
	if spanID != "" {
		if visited[spanID] {
			return 0, 0
		}
		visited[spanID] = true
	}
	totalTime := 0.0
	if duration, ok := trace.GetDurationMS(span); ok {
		totalTime = duration
	}
	totalTokens := trace.GetTokenCount(span)
	for _, child := range children[spanID] {
		childTime, childTokens := computeSubtreeStats(child, children, visited)
		totalTime += childTime
		totalTokens += childTokens
	}
	return totalTime, totalTokens
}

func totalTokensInTree(children map[string][]trace.Span, span trace.Span, visited map[string]bool) int {
	_, tokens := computeSubtreeStats(span, children, visited)
	return tokens
}

func formatMetrics(totalTime float64, totalTokens int) string {
	parts := []string{formatTimeMS(totalTime)}
	if totalTokens > 0 {
		parts = append(parts, fmt.Sprintf("%d", totalTokens))
	}
	return "[" + strings.Join(parts, " | ") + "]"
}

func traceTimes(spans []trace.Span) (time.Time, time.Time, float64) {
	starts := []time.Time{}
	ends := []time.Time{}
	for _, span := range spans {
		if start, err := time.Parse(time.RFC3339Nano, stringValue(span["start_time"])); err == nil {
			starts = append(starts, start)
		}
		if end, err := time.Parse(time.RFC3339Nano, stringValue(span["end_time"])); err == nil {
			ends = append(ends, end)
		}
	}
	if len(starts) == 0 || len(ends) == 0 {
		return time.Time{}, time.Time{}, 0
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i].Before(starts[j]) })
	sort.Slice(ends, func(i, j int) bool { return ends[i].Before(ends[j]) })
	start := starts[0]
	end := ends[len(ends)-1]
	return start, end, float64(end.Sub(start).Microseconds()) / 1000
}

func formatTimeMS(ms float64) string {
	if ms >= 1000 {
		return fmt.Sprintf("%.2fs", ms/1000)
	}
	return fmt.Sprintf("%.0fms", ms)
}

func valueString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any, map[string]any:
		data, err := json.Marshal(typed)
		if err == nil {
			return string(data)
		}
	}
	return fmt.Sprint(value)
}

func contentString(value any) string {
	if parts, ok := value.([]any); ok {
		result := []string{}
		for _, part := range parts {
			if obj, ok := part.(map[string]any); ok {
				result = append(result, valueString(obj["text"]))
			}
		}
		return strings.Join(result, " ")
	}
	return valueString(value)
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

func prefixString(value string, length int) string {
	if len(value) <= length {
		return value
	}
	return value[:length]
}

func firstString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	if raw, ok := value.(string); ok {
		return raw
	}
	return ""
}
