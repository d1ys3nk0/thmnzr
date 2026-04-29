package input

import (
	"encoding/base64"
	"net/url"
	"regexp"
	"strings"
)

var (
	traceIDPattern = regexp.MustCompile(`(?i)\b([a-f0-9]{32})\b`)
	spanIDPattern  = regexp.MustCompile(`(?i)\b([a-f0-9]{16})\b`)
	urlPattern     = regexp.MustCompile(`https?://[^\s]+`)
)

type Parsed struct {
	URL              string
	ProjectID        string
	ProjectIDDecoded string
	TraceID          string
	SpanID           string
}

func Parse(raw string) Parsed {
	raw = strings.TrimSpace(raw)
	if match := urlPattern.FindString(raw); match != "" {
		return ParsePhoenixURL(match)
	}
	if traceID := ExtractTraceID(raw); traceID != "" {
		return Parsed{TraceID: traceID}
	}
	return Parsed{}
}

func ParsePhoenixURL(raw string) Parsed {
	parsedURL, err := url.Parse(raw)
	if err != nil {
		return Parsed{URL: raw}
	}

	segments := []string{}
	for _, segment := range strings.Split(strings.Trim(parsedURL.Path, "/"), "/") {
		if segment != "" {
			segments = append(segments, segment)
		}
	}

	result := Parsed{URL: raw}
	for i, segment := range segments {
		switch segment {
		case "projects":
			if i+1 < len(segments) {
				result.ProjectID = segments[i+1]
				result.ProjectIDDecoded = decodeBase64OrSelf(result.ProjectID)
			}
		case "spans":
			if i+1 < len(segments) {
				result.SpanID = segments[i+1]
			}
		case "traces":
			if i+1 < len(segments) {
				result.TraceID = segments[i+1]
			}
		}
	}

	if result.TraceID == "" {
		result.TraceID = ExtractTraceID(parsedURL.Path)
	}

	return result
}

func ExtractTraceID(raw string) string {
	match := traceIDPattern.FindStringSubmatch(raw)
	if len(match) < 2 {
		return ""
	}
	return strings.ToLower(match[1])
}

func ExtractSpanID(raw string) string {
	match := spanIDPattern.FindStringSubmatch(raw)
	if len(match) < 2 {
		return ""
	}
	return strings.ToLower(match[1])
}

func decodeBase64OrSelf(raw string) string {
	padding := len(raw) % 4
	if padding > 0 {
		raw += strings.Repeat("=", 4-padding)
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return strings.TrimRight(raw, "=")
	}
	return string(decoded)
}
