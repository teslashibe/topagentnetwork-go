package topagentnetwork

import (
	"context"
	"encoding/json"
	"fmt"
)

// SchemaDigest is a compact, agent-friendly view of TAN's root GraphQL
// operations. It exists because the only way callers learn the real schema
// is introspection — guessing field names (Relay-style `posts`/`edges`/
// `first`, or omitting required args like `nodes(ids: [ID!]!)`) produces
// queries the server rejects.
type SchemaDigest struct {
	QueryType    *IntrospectedType `json:"queryType,omitempty"`
	MutationType *IntrospectedType `json:"mutationType,omitempty"`
}

// IntrospectedType is a compact description of one GraphQL type: its fields
// (for OBJECT types) and/or input fields (for INPUT_OBJECT types).
type IntrospectedType struct {
	Name        string              `json:"name"`
	Kind        string              `json:"kind"`
	Description string              `json:"description,omitempty"`
	Fields      []IntrospectedField `json:"fields,omitempty"`
	InputFields []IntrospectedField `json:"inputFields,omitempty"`
	EnumValues  []string            `json:"enumValues,omitempty"`
}

// IntrospectedField is one field (or input field) with its rendered GraphQL
// type and arguments. Type is rendered in SDL form, e.g. "[Listing!]!".
type IntrospectedField struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Type        string            `json:"type"`
	Args        []IntrospectedArg `json:"args,omitempty"`
}

// IntrospectedArg is one argument on a field, with its rendered GraphQL
// type. A trailing "!" marks a required argument.
type IntrospectedArg struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
}

// rootIntrospectionQuery discovers the actual names of the root operation
// types (usually "Query"/"Mutation", but never assume).
const rootIntrospectionQuery = `query IntrospectRoot {
  __schema { queryType { name } mutationType { name } }
}`

// typeIntrospectionQuery fetches a single named type. The nested ofType
// chain unwraps NON_NULL/LIST wrappers (GraphQL nests up to a handful deep
// for things like [ID!]!); five levels is comfortably more than TAN uses.
const typeIntrospectionQuery = `query IntrospectType($name: String!) {
  __type(name: $name) {
    name
    kind
    description
    fields(includeDeprecated: true) {
      name
      description
      args { name description type { ...TypeRef } }
      type { ...TypeRef }
    }
    inputFields { name description type { ...TypeRef } }
    enumValues(includeDeprecated: true) { name }
  }
}
fragment TypeRef on __Type {
  kind name
  ofType { kind name ofType { kind name ofType { kind name ofType { kind name ofType { kind name } } } } }
}`

type rawTypeRef struct {
	Kind   string      `json:"kind"`
	Name   *string     `json:"name"`
	OfType *rawTypeRef `json:"ofType"`
}

type rawArg struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Type        rawTypeRef `json:"type"`
}

type rawField struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Args        []rawArg   `json:"args"`
	Type        rawTypeRef `json:"type"`
}

type rawType struct {
	Name        string     `json:"name"`
	Kind        string     `json:"kind"`
	Description string     `json:"description"`
	Fields      []rawField `json:"fields"`
	InputFields []rawArg   `json:"inputFields"`
	EnumValues  []struct {
		Name string `json:"name"`
	} `json:"enumValues"`
}

// IntrospectSchema returns the root Query and Mutation fields with their
// arguments, so callers can build valid queries without guessing.
func (c *Client) IntrospectSchema(ctx context.Context) (*SchemaDigest, error) {
	queryName, mutationName, err := c.rootTypeNames(ctx)
	if err != nil {
		return nil, err
	}
	digest := &SchemaDigest{}
	if queryName != "" {
		if digest.QueryType, err = c.Introspect(ctx, queryName); err != nil {
			return nil, err
		}
	}
	if mutationName != "" {
		if digest.MutationType, err = c.Introspect(ctx, mutationName); err != nil {
			return nil, err
		}
	}
	return digest, nil
}

// Introspect returns a compact description of a single GraphQL type (e.g.
// "Query", "Mutation", or an object/input type name). Empty typeName
// defaults to "Query".
func (c *Client) Introspect(ctx context.Context, typeName string) (*IntrospectedType, error) {
	if typeName == "" {
		typeName = "Query"
	}
	resp, err := c.Query(ctx, GraphQLRequest{
		OperationName: "IntrospectType",
		Query:         typeIntrospectionQuery,
		Variables:     map[string]any{"name": typeName},
	})
	if err != nil {
		return nil, err
	}
	rawData, ok := resp.Data["__type"]
	if !ok || rawData == nil {
		return nil, fmt.Errorf("%w: type %q does not exist in the TAN schema", ErrNotFound, typeName)
	}
	rt, err := decodeRawType(rawData)
	if err != nil {
		return nil, err
	}
	return convertType(rt), nil
}

func (c *Client) rootTypeNames(ctx context.Context) (queryName, mutationName string, err error) {
	resp, err := c.Query(ctx, GraphQLRequest{
		OperationName: "IntrospectRoot",
		Query:         rootIntrospectionQuery,
	})
	if err != nil {
		return "", "", err
	}
	schema, ok := resp.Data["__schema"].(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("%w: introspection disabled or unavailable", ErrRequestFailed)
	}
	queryName = rootTypeName(schema["queryType"])
	mutationName = rootTypeName(schema["mutationType"])
	return queryName, mutationName, nil
}

func rootTypeName(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	name, _ := m["name"].(string)
	return name
}

func decodeRawType(v any) (*rawType, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	var rt rawType
	if err := json.Unmarshal(b, &rt); err != nil {
		return nil, fmt.Errorf("%w: malformed introspection payload: %v", ErrRequestFailed, err)
	}
	return &rt, nil
}

func convertType(rt *rawType) *IntrospectedType {
	out := &IntrospectedType{
		Name:        rt.Name,
		Kind:        rt.Kind,
		Description: rt.Description,
	}
	for _, f := range rt.Fields {
		field := IntrospectedField{
			Name:        f.Name,
			Description: f.Description,
			Type:        renderTypeRef(&f.Type),
		}
		for _, a := range f.Args {
			field.Args = append(field.Args, IntrospectedArg{
				Name:        a.Name,
				Description: a.Description,
				Type:        renderTypeRef(&a.Type),
			})
		}
		out.Fields = append(out.Fields, field)
	}
	for _, f := range rt.InputFields {
		out.InputFields = append(out.InputFields, IntrospectedField{
			Name:        f.Name,
			Description: f.Description,
			Type:        renderTypeRef(&f.Type),
		})
	}
	for _, e := range rt.EnumValues {
		out.EnumValues = append(out.EnumValues, e.Name)
	}
	return out
}

// renderTypeRef flattens GraphQL's nested NON_NULL/LIST wrappers into SDL
// notation, e.g. NON_NULL(LIST(NON_NULL(ID))) -> "[ID!]!". This is exactly
// the signal callers need: it makes required args (the trailing "!") and
// list shapes visible, so they stop omitting things like nodes(ids: [ID!]!).
func renderTypeRef(t *rawTypeRef) string {
	if t == nil {
		return "Unknown"
	}
	switch t.Kind {
	case "NON_NULL":
		return renderTypeRef(t.OfType) + "!"
	case "LIST":
		return "[" + renderTypeRef(t.OfType) + "]"
	default:
		if t.Name != nil && *t.Name != "" {
			return *t.Name
		}
		return "Unknown"
	}
}
