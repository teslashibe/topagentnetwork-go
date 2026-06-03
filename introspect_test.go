package topagentnetwork

import (
	"encoding/json"
	"testing"
)

func strptr(s string) *string { return &s }

// TestRenderTypeRef pins the SDL rendering of wrapped types. The [ID!]! case
// is the exact arg shape the agent kept omitting on Query.nodes (issue #56),
// so it must render with both the inner and outer "!".
func TestRenderTypeRef(t *testing.T) {
	cases := []struct {
		name string
		ref  *rawTypeRef
		want string
	}{
		{
			name: "scalar",
			ref:  &rawTypeRef{Kind: "SCALAR", Name: strptr("String")},
			want: "String",
		},
		{
			name: "non_null scalar",
			ref:  &rawTypeRef{Kind: "NON_NULL", OfType: &rawTypeRef{Kind: "SCALAR", Name: strptr("ID")}},
			want: "ID!",
		},
		{
			name: "list of non_null id, non_null (nodes ids arg)",
			ref: &rawTypeRef{
				Kind: "NON_NULL",
				OfType: &rawTypeRef{
					Kind:   "LIST",
					OfType: &rawTypeRef{Kind: "NON_NULL", OfType: &rawTypeRef{Kind: "SCALAR", Name: strptr("ID")}},
				},
			},
			want: "[ID!]!",
		},
		{
			name: "nil",
			ref:  nil,
			want: "Unknown",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := renderTypeRef(tc.ref); got != tc.want {
				t.Fatalf("renderTypeRef = %q, want %q", got, tc.want)
			}
		})
	}
}

// canned __type payload for the root Query type, modeled on TAN's real
// schema: `nodes` takes a required `ids: [ID!]!` and returns `[Node]!`.
// There is deliberately NO `posts` field and NO `first` arg — the things
// the agent hallucinated in issue #56.
const queryTypeIntrospection = `{
  "name": "Query",
  "kind": "OBJECT",
  "description": "Root query type",
  "fields": [
    {
      "name": "nodes",
      "description": "Fetch nodes by id",
      "args": [
        {"name": "ids", "description": "node ids", "type": {"kind": "NON_NULL", "name": null, "ofType": {"kind": "LIST", "name": null, "ofType": {"kind": "NON_NULL", "name": null, "ofType": {"kind": "SCALAR", "name": "ID", "ofType": null}}}}}
      ],
      "type": {"kind": "NON_NULL", "name": null, "ofType": {"kind": "LIST", "name": null, "ofType": {"kind": "OBJECT", "name": "Node", "ofType": null}}}
    },
    {
      "name": "currentlyLoggedInAgent",
      "description": "The signed-in agent",
      "args": [],
      "type": {"kind": "OBJECT", "name": "Agent", "ofType": null}
    }
  ],
  "inputFields": null,
  "enumValues": null
}`

func TestConvertType_RootQuery(t *testing.T) {
	var raw any
	if err := json.Unmarshal([]byte(queryTypeIntrospection), &raw); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	rt, err := decodeRawType(raw)
	if err != nil {
		t.Fatalf("decodeRawType: %v", err)
	}
	got := convertType(rt)

	if got.Name != "Query" || got.Kind != "OBJECT" {
		t.Fatalf("type header = %q/%q, want Query/OBJECT", got.Name, got.Kind)
	}

	byName := map[string]IntrospectedField{}
	for _, f := range got.Fields {
		byName[f.Name] = f
	}

	nodes, ok := byName["nodes"]
	if !ok {
		t.Fatal("expected `nodes` field in digest")
	}
	if nodes.Type != "[Node]!" {
		t.Errorf("nodes return type = %q, want [Node]!", nodes.Type)
	}
	if len(nodes.Args) != 1 || nodes.Args[0].Name != "ids" || nodes.Args[0].Type != "[ID!]!" {
		t.Errorf("nodes args = %+v, want one `ids: [ID!]!`", nodes.Args)
	}

	// Regression guard: the fields the agent hallucinated must NOT appear.
	if _, bad := byName["posts"]; bad {
		t.Error("digest unexpectedly contains hallucinated `posts` field")
	}
	for _, a := range nodes.Args {
		if a.Name == "first" {
			t.Error("digest unexpectedly contains hallucinated `first` arg")
		}
	}
}
