package topagentnetwork

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TAN's GraphQL backend (api.topagentnetwork.app) does NOT accept the
// session cookie directly — it requires an `Authorization: Bearer <jwt>`
// header. The JWT is a short-lived (~1h) RS256 access token minted by the
// Next.js web app: POST topagentnetwork.app/api/refresh with the session
// cookies returns {"accessToken": "<jwt>"} (and rotates the session cookie,
// which we deliberately ignore — the original cookie keeps working, so we
// avoid racing the browser/extension that owns it).
//
// We cache the token and refresh it shortly before expiry. This is what
// makes cookie-synced sessions actually usable against the API.

// tokenSkew refreshes the access token this long before its real expiry so
// an in-flight request never races the boundary.
const tokenSkew = 60 * time.Second

type refreshResponse struct {
	AccessToken string `json:"accessToken"`
}

// ensureToken returns a valid bearer token, minting (or refreshing) one via
// /api/refresh when the cache is empty or near expiry.
func (c *Client) ensureToken(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.accessToken != "" && time.Now().Before(c.tokenExp.Add(-tokenSkew)) {
		return c.accessToken, nil
	}
	tok, err := c.mintToken(ctx)
	if err != nil {
		return "", err
	}
	c.accessToken = tok
	c.tokenExp = jwtExpiry(tok)
	return tok, nil
}

// invalidateToken drops the cached token so the next call re-mints. Called
// when the API rejects the current token (401/unauthenticated) despite it
// not having locally expired.
func (c *Client) invalidateToken() {
	c.tokenMu.Lock()
	c.accessToken = ""
	c.tokenExp = time.Time{}
	c.tokenMu.Unlock()
}

// mintToken exchanges the session cookies for a fresh access token. It does
// not go through doRetried/setCommonHeaders to avoid attaching a (possibly
// stale) bearer to the refresh call itself.
func (c *Client) mintToken(ctx context.Context) (string, error) {
	c.authMu.RLock()
	cookie := c.auth.CookieHeader
	c.authMu.RUnlock()
	if cookie == "" {
		return "", ErrUnauthorized
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webBaseURL+refreshPath, bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", webBaseURL)
	req.Header.Set("Referer", webBaseURL+"/")
	req.Header.Set("Cookie", cookie)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return "", &HTTPError{StatusCode: resp.StatusCode, Body: truncate(string(raw), 256)}
	}
	var rr refreshResponse
	if err := json.Unmarshal(raw, &rr); err != nil {
		return "", fmt.Errorf("%w: malformed refresh response: %v", ErrRequestFailed, err)
	}
	if rr.AccessToken == "" {
		// A 200 with no token means the session cookie is no longer valid
		// (TAN returns an empty/anonymous payload rather than a 401).
		return "", ErrUnauthorized
	}
	return rr.AccessToken, nil
}

// jwtExpiry parses the `exp` claim from a JWT without verifying its
// signature (we only need the expiry for cache timing; the server is the
// real authority). Falls back to a conservative 5-minute TTL when the claim
// can't be read, so a malformed token still gets retried promptly.
func jwtExpiry(token string) time.Time {
	fallback := time.Now().Add(5 * time.Minute)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fallback
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fallback
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return fallback
	}
	return time.Unix(claims.Exp, 0)
}
