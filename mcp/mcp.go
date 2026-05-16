// Package mcp exposes topagentnetwork-go as a set of MCP tools.
package mcp

import "github.com/teslashibe/mcptool"

// Provider implements [mcptool.Provider] for topagentnetwork-go.
type Provider struct{}

// Platform returns "topagentnetwork".
func (Provider) Platform() string { return "topagentnetwork" }

// Tools returns every MCP tool, in registration order.
func (Provider) Tools() []mcptool.Tool {
	out := make([]mcptool.Tool, 0, len(authTools)+len(queryTools))
	out = append(out, authTools...)
	out = append(out, queryTools...)
	return out
}
