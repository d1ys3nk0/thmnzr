package phoenix

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/d1ys3nk0/thmnzr/internal/trace"
)

const defaultLimit = 10000

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type spansResponse struct {
	Data       []trace.Span `json:"data"`
	NextCursor string       `json:"next_cursor"`
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func NewClientWithHTTP(baseURL, apiKey string, httpClient *http.Client) *Client {
	client := NewClient(baseURL, apiKey)
	client.httpClient = httpClient
	return client
}

func (c *Client) GetSpan(projectID, spanID string) (trace.Span, bool, error) {
	spans, err := c.getSpans(projectID, map[string][]string{"span_id": {spanID}}, 1)
	if err != nil {
		return nil, false, err
	}
	if len(spans) == 0 {
		return nil, false, nil
	}
	return spans[0], true, nil
}

func (c *Client) GetTraceSpans(projectID, traceID string) ([]trace.Span, error) {
	return c.getSpans(projectID, map[string][]string{"trace_id": {traceID}}, defaultLimit)
}

func (c *Client) getSpans(projectID string, params map[string][]string, limit int) ([]trace.Span, error) {
	allSpans := []trace.Span{}
	cursor := ""
	pageSize := min(100, limit)

	for len(allSpans) < limit {
		remaining := limit - len(allSpans)
		currentPageSize := min(pageSize, remaining)

		query := url.Values{}
		query.Set("limit", fmt.Sprintf("%d", currentPageSize))
		for key, values := range params {
			for _, value := range values {
				query.Add(key, value)
			}
		}
		if cursor != "" {
			query.Set("cursor", cursor)
		}

		endpoint := fmt.Sprintf("%s/v1/projects/%s/spans?%s", c.baseURL, url.PathEscape(projectID), query.Encode())
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		if c.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("phoenix returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}

		var decoded spansResponse
		if err := json.Unmarshal(body, &decoded); err != nil {
			return nil, err
		}
		allSpans = append(allSpans, decoded.Data...)
		if decoded.NextCursor == "" || len(decoded.Data) == 0 {
			break
		}
		cursor = decoded.NextCursor
	}

	return allSpans[:min(len(allSpans), limit)], nil
}
