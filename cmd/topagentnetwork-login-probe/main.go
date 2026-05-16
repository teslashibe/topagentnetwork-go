// Command topagentnetwork-login-probe validates the pasted browser
// cookie by calling currentlyLoggedInAgent.
package main

import (
	"context"
	"fmt"
	"os"

	tan "github.com/teslashibe/topagentnetwork-go"
)

func main() {
	auth := tan.Auth{
		Email:        os.Getenv("TOPAGENTNETWORK_EMAIL"),
		Password:     os.Getenv("TOPAGENTNETWORK_PASSWORD"),
		CookieHeader: os.Getenv("TOPAGENTNETWORK_COOKIE_HEADER"),
	}
	c, err := tan.New(auth)
	if err != nil {
		fmt.Fprintln(os.Stderr, "init:", err)
		os.Exit(1)
	}
	user, err := c.Login(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "login:", err)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "If you see 'login not implemented', paste your browser cookie:")
		fmt.Fprintln(os.Stderr, "  1. Log in at https://topagentnetwork.app in a real browser")
		fmt.Fprintln(os.Stderr, "  2. DevTools → Network → POST /api/login → copy the entire `cookie` request header")
		fmt.Fprintln(os.Stderr, "  3. export TOPAGENTNETWORK_COOKIE_HEADER='<paste>'")
		os.Exit(1)
	}
	fmt.Printf("logged in as %s %s (%s) id=%s\n", user.FirstName, user.LastName, user.Email, user.ID)
}
