package mcp

import (
	"context"

	tan "github.com/teslashibe/topagentnetwork-go"
	"github.com/teslashibe/mcptool"
)

// LoginInput is the typed input for topagentnetwork_login.
type LoginInput struct{}

func login(ctx context.Context, c *tan.Client, _ LoginInput) (any, error) {
	return c.Login(ctx)
}

// GetMeInput is the typed input for topagentnetwork_get_me.
type GetMeInput struct{}

func getMe(ctx context.Context, c *tan.Client, _ GetMeInput) (any, error) {
	return c.GetMe(ctx)
}

var authTools = []mcptool.Tool{
	mcptool.Define[*tan.Client, LoginInput](
		"topagentnetwork_login",
		"Validate the pasted browser session cookie by calling the currentlyLoggedInAgent GraphQL query.",
		"Login",
		login,
	),
	mcptool.Define[*tan.Client, GetMeInput](
		"topagentnetwork_get_me",
		"Return the authenticated agent's profile via the currentlyLoggedInAgent GraphQL query.",
		"GetMe",
		getMe,
	),
}
