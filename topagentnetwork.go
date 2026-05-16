// Package topagentnetwork is a Go client + MCP tool surface for
// topagentnetwork.app ("TAN"). There is no public API.
//
// The website is a Next.js front-end at topagentnetwork.app that talks
// to a GraphQL backend at https://api.topagentnetwork.app/graphql. The
// public schema exposes 78 queries (incl. currentlyLoggedInAgent,
// sessionV5, alertsForCurrentUser, messageBroadcasts, etc.) and 139
// mutations.
//
// Authentication
//
// TAN's web app authenticates via a Next.js /api/login proxy that
// sets opaque first-party session cookies on topagentnetwork.app.
// Programmatically reproducing that proxy is non-trivial (the proxy
// returns HTTP 500 when called outside the browser, suggesting a
// CSRF/initial-cookie requirement we have not yet captured). For now
// this package takes the AllTrails-style fallback: paste the
// authenticated cookies from a real browser session into Auth and
// reuse them. The cookie-paste path is fully working — every GraphQL
// query/mutation becomes reachable once cookies are populated.
//
// To pull cookies: DevTools → Application → Cookies →
// https://topagentnetwork.app — copy the values whose names start
// with "_tan", "session", or similar (paste them whole into
// Auth.CookieHeader; the package treats CookieHeader as opaque).
package topagentnetwork

import (
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"
)

const (
	webBaseURL       = "https://topagentnetwork.app"
	apiBaseURL       = "https://api.topagentnetwork.app"
	graphqlPath      = "/graphql"
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	defaultRetries   = 3
	defaultRetryBase = 500 * time.Millisecond
)

// Client communicates with the TAN GraphQL backend.
type Client struct {
	auth       Auth
	httpClient *http.Client
	userAgent  string
	maxRetries int
	retryBase  time.Duration
	minGap     time.Duration

	gapMu     sync.Mutex
	lastReqAt time.Time

	authMu sync.RWMutex
}

// Option configures a Client.
type Option func(*Client)

// WithUserAgent overrides the default browser User-Agent string.
func WithUserAgent(ua string) Option { return func(c *Client) { c.userAgent = ua } }

// WithRetry sets the maximum retry count and base backoff duration.
func WithRetry(maxRetries int, base time.Duration) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
		c.retryBase = base
	}
}

// WithHTTPClient overrides the default http.Client. Nil is ignored.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithMinRequestGap sets the minimum time between consecutive requests.
// Defaults to 400ms.
func WithMinRequestGap(d time.Duration) Option {
	return func(c *Client) { c.minGap = d }
}

// New creates a new client. Auth.CookieHeader must be supplied (paste
// from a logged-in browser session — see package doc).
func New(auth Auth, opts ...Option) (*Client, error) {
	if auth.CookieHeader == "" && (auth.Email == "" || auth.Password == "") {
		return nil, ErrInvalidAuth
	}
	jar, _ := cookiejar.New(nil)
	c := &Client{
		auth:       auth,
		httpClient: &http.Client{Timeout: 30 * time.Second, Jar: jar},
		userAgent:  defaultUserAgent,
		maxRetries: defaultRetries,
		retryBase:  defaultRetryBase,
		minGap:     400 * time.Millisecond,
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// AuthSnapshot returns the current cached auth credentials.
func (c *Client) AuthSnapshot() Auth {
	c.authMu.RLock()
	defer c.authMu.RUnlock()
	return c.auth
}
