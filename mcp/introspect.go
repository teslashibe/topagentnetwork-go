package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"
	tan "github.com/teslashibe/topagentnetwork-go"
)

// IntrospectInput is the typed input for topagentnetwork_introspect.
type IntrospectInput struct {
	TypeName string `json:"type_name,omitempty" jsonschema:"description=Optional GraphQL type to inspect (e.g. Query, Mutation, or an object/input type). Omit to list all root query + mutation fields with their arguments."`
}

func introspect(ctx context.Context, c *tan.Client, in IntrospectInput) (any, error) {
	if in.TypeName == "" {
		return c.IntrospectSchema(ctx)
	}
	return c.Introspect(ctx, in.TypeName)
}

var introspectTools = []mcptool.Tool{
	mcptool.Define[*tan.Client, IntrospectInput](
		"topagentnetwork_introspect",
		"Introspect TAN's real GraphQL schema (fields, args, types). Omit type_name to list all root query/mutation fields.",
		"Query",
		introspect,
	),
}
