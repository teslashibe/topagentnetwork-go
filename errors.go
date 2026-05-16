package topagentnetwork

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors.
var (
	ErrInvalidAuth         = errors.New("topagentnetwork: missing or invalid auth credentials (paste a browser cookie into Auth.CookieHeader)")
	ErrUnauthorized        = errors.New("topagentnetwork: unauthorized (session expired — re-paste cookies from browser)")
	ErrForbidden           = errors.New("topagentnetwork: forbidden")
	ErrNotFound            = errors.New("topagentnetwork: not found")
	ErrRateLimited         = errors.New("topagentnetwork: rate limited")
	ErrInvalidParams       = errors.New("topagentnetwork: invalid parameters")
	ErrRequestFailed       = errors.New("topagentnetwork: request failed")
	ErrLoginFailed         = errors.New("topagentnetwork: login failed")
	ErrLoginNotImplemented = errors.New("topagentnetwork: programmatic email/password login is not yet supported; paste a browser session cookie into Auth.CookieHeader (see package docs)")
)

// HTTPError is returned for unexpected non-2xx HTTP responses.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("topagentnetwork: HTTP %d: %s", e.StatusCode, e.Body)
}

// GraphQLErrors wraps one or more GraphQLError entries.
type GraphQLErrors struct {
	Errors []GraphQLError
}

func (g *GraphQLErrors) Error() string {
	msgs := make([]string, 0, len(g.Errors))
	for _, e := range g.Errors {
		msgs = append(msgs, e.Message)
	}
	return "topagentnetwork: graphql: " + strings.Join(msgs, "; ")
}
