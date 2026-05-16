package topagentnetwork

// Auth carries the credentials needed to talk to the TAN GraphQL API.
//
// Two modes are defined:
//
//  1. CookieHeader (working today) — paste the raw Cookie header from a
//     logged-in browser session at https://topagentnetwork.app. The
//     package treats it as opaque and forwards it on every request.
//
//  2. Email + Password (placeholder) — reserved for the future
//     programmatic login path. Today the website's /api/login proxy
//     returns HTTP 500 to non-browser callers (likely CSRF/initial
//     cookie requirement). Setting these fields without CookieHeader
//     will cause Login() to return ErrLoginNotImplemented; populate
//     CookieHeader instead.
type Auth struct {
	Email        string
	Password     string
	CookieHeader string
}

// User mirrors a subset of TAN's "currentlyLoggedInAgent" GraphQL type.
// Real schema fields are richer; expand as needed.
type User struct {
	ID        string `json:"id,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Email     string `json:"email,omitempty"`
	Slug      string `json:"slug,omitempty"`
	AgentType string `json:"agentType,omitempty"`
}

// GraphQLRequest is the shape posted to /graphql.
type GraphQLRequest struct {
	OperationName string         `json:"operationName,omitempty"`
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
}

// GraphQLResponse mirrors the standard GraphQL response envelope.
type GraphQLResponse struct {
	Data   map[string]any   `json:"data,omitempty"`
	Errors []GraphQLError   `json:"errors,omitempty"`
}

// GraphQLError is one entry in a GraphQL `errors` array.
type GraphQLError struct {
	Message    string         `json:"message"`
	Path       []any          `json:"path,omitempty"`
	Extensions map[string]any `json:"extensions,omitempty"`
}
