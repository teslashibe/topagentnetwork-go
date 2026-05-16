package topagentnetwork

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

// Query executes an arbitrary GraphQL query/mutation against
// /graphql and returns the parsed response envelope. If the response
// carries `errors`, a *GraphQLErrors is returned.
//
// This is the foundational primitive — every other typed helper on
// Client is implemented in terms of Query.
func (c *Client) Query(ctx context.Context, op GraphQLRequest) (*GraphQLResponse, error) {
	body, err := json.Marshal(op)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}
	raw, _, err := c.doRetried(ctx, http.MethodPost, apiBaseURL+graphqlPath, body, "application/json")
	if err != nil {
		return nil, err
	}
	var out GraphQLResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%w: malformed graphql response: %v", ErrRequestFailed, err)
	}
	if len(out.Errors) > 0 {
		for _, ge := range out.Errors {
			m := strings.ToLower(ge.Message)
			if strings.Contains(m, "unauthorized") || strings.Contains(m, "unauthenticated") || strings.Contains(m, "not authenticated") {
				return &out, ErrUnauthorized
			}
		}
		return &out, &GraphQLErrors{Errors: out.Errors}
	}
	return &out, nil
}

func (c *Client) doRetried(ctx context.Context, method, rawURL string, body []byte, contentType string) ([]byte, int, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			wait := c.backoff(attempt)
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(wait):
			}
		}
		raw, status, err := c.doRequest(ctx, method, rawURL, body, contentType)
		if err == nil {
			return raw, status, nil
		}
		lastErr = err
		if errors.Is(err, ErrRateLimited) {
			continue
		}
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode >= 500 {
			continue
		}
		return nil, status, err
	}
	return nil, 0, lastErr
}

func (c *Client) doRequest(ctx context.Context, method, rawURL string, body []byte, contentType string) ([]byte, int, error) {
	c.waitForGap(ctx)
	if ctx.Err() != nil {
		return nil, 0, ctx.Err()
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	c.setCommonHeaders(req, contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("%w: reading body: %v", ErrRequestFailed, err)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return raw, resp.StatusCode, nil
	case http.StatusUnauthorized:
		return nil, resp.StatusCode, ErrUnauthorized
	case http.StatusForbidden:
		return nil, resp.StatusCode, ErrForbidden
	case http.StatusNotFound:
		return nil, resp.StatusCode, ErrNotFound
	case http.StatusTooManyRequests:
		c.gapMu.Lock()
		if earliest := time.Now().Add(60 * time.Second); c.lastReqAt.Before(earliest) {
			c.lastReqAt = earliest
		}
		c.gapMu.Unlock()
		return nil, resp.StatusCode, fmt.Errorf("%w: 429", ErrRateLimited)
	default:
		return nil, resp.StatusCode, &HTTPError{StatusCode: resp.StatusCode, Body: truncate(string(raw), 256)}
	}
}

func (c *Client) setCommonHeaders(req *http.Request, contentType string) {
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", webBaseURL)
	req.Header.Set("Referer", webBaseURL+"/")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	c.authMu.RLock()
	if c.auth.CookieHeader != "" {
		req.Header.Set("Cookie", c.auth.CookieHeader)
	}
	c.authMu.RUnlock()
}

func (c *Client) backoff(attempt int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempt-1))) * c.retryBase
}

func (c *Client) waitForGap(ctx context.Context) {
	c.gapMu.Lock()
	now := time.Now()
	next := c.lastReqAt.Add(c.minGap)
	if now.After(next) {
		next = now
	}
	c.lastReqAt = next
	c.gapMu.Unlock()
	if wait := time.Until(next); wait > 0 {
		select {
		case <-ctx.Done():
		case <-time.After(wait):
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
