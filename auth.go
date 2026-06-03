package topagentnetwork

import (
	"context"
	"encoding/json"
	"fmt"
)

// Login is reserved for the future programmatic email/password path.
// TAN's /api/login Next.js proxy returns HTTP 500 to non-browser
// callers (apparent CSRF/initial-cookie requirement we have not yet
// captured from a real session). Until that is reverse-engineered,
// Login always returns ErrLoginNotImplemented when invoked without a
// pre-populated Auth.CookieHeader. Populate CookieHeader from a
// browser session instead — every GraphQL endpoint becomes reachable.
func (c *Client) Login(ctx context.Context) (*User, error) {
	c.authMu.RLock()
	hasCookie := c.auth.CookieHeader != ""
	c.authMu.RUnlock()
	if !hasCookie {
		return nil, ErrLoginNotImplemented
	}
	return c.GetMe(ctx)
}

// GetMe runs the "currentlyLoggedInAgent" GraphQL query and returns a
// normalized User. If the session cookie is invalid the response is
// either GraphQL errors (surfaced as ErrUnauthorized via Query) or a
// null data block (also returned as ErrUnauthorized).
func (c *Client) GetMe(ctx context.Context) (*User, error) {
	op := GraphQLRequest{
		OperationName: "CurrentlyLoggedInAgent",
		// TAN's Agent type dropped email/slug/agentType (the API now rejects
		// them with "Cannot query field ... on type Agent"). Request only the
		// fields the current schema still exposes; GetMe's primary job is an
		// auth/liveness check, so id/firstName/lastName are sufficient.
		Query: `query CurrentlyLoggedInAgent {
  currentlyLoggedInAgent {
    id
    firstName
    lastName
  }
}`,
	}
	resp, err := c.Query(ctx, op)
	if err != nil {
		return nil, fmt.Errorf("GetMe: %w", err)
	}
	rawNode, ok := resp.Data["currentlyLoggedInAgent"]
	if !ok || rawNode == nil {
		return nil, fmt.Errorf("GetMe: %w", ErrUnauthorized)
	}
	raw, err := json.Marshal(rawNode)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	var u User
	if err := json.Unmarshal(raw, &u); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	return &u, nil
}
