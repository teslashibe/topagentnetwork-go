package mcp

import (
	"context"

	tan "github.com/teslashibe/topagentnetwork-go"
	"github.com/teslashibe/mcptool"
)

// QueryInput is the typed input for topagentnetwork_query.
type QueryInput struct {
	OperationName string         `json:"operation_name,omitempty" jsonschema:"description=Optional GraphQL operation name"`
	Query         string         `json:"query" jsonschema:"description=Raw GraphQL query/mutation string,required"`
	Variables     map[string]any `json:"variables,omitempty" jsonschema:"description=Optional variables map"`
}

func query(ctx context.Context, c *tan.Client, in QueryInput) (any, error) {
	return c.Query(ctx, tan.GraphQLRequest{
		OperationName: in.OperationName,
		Query:         in.Query,
		Variables:     in.Variables,
	})
}

var queryTools = []mcptool.Tool{
	mcptool.Define[*tan.Client, QueryInput](
		"topagentnetwork_query",
		"Execute an arbitrary GraphQL query/mutation against api.topagentnetwork.app/graphql. 78 queries and 139 mutations available; see TAN introspection for the schema.",
		"Query",
		query,
	),
}
