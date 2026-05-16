# topagentnetwork-go

Private Go client + MCP server for [topagentnetwork.app](https://topagentnetwork.app)
("TAN"). There is no public API. The website is a Next.js SPA that
talks to a GraphQL backend at `https://api.topagentnetwork.app/graphql`.
Schema is large and useful: **78 queries** (incl. `currentlyLoggedInAgent`,
`sessionV5`, `messageBroadcasts`, `alertsForCurrentUser`) and **139
mutations**.

## Auth status — read this first

TAN's `/api/login` Next.js proxy returns HTTP 500 to non-browser
callers (probably a CSRF / initial-cookie requirement we have not yet
captured). Programmatic email/password login therefore returns
`ErrLoginNotImplemented`.

**Working path today: paste your browser cookie.**

1. Log in at https://topagentnetwork.app in Chrome.
2. DevTools → Network → reload → pick any request to
   `api.topagentnetwork.app/graphql` → Headers → Request Headers →
   copy the **entire** `cookie:` value.
3. `export TOPAGENTNETWORK_COOKIE_HEADER='<paste>'` (or put it in
   `~/.topagentnetwork-mcp/config.json` under `cookie_header`).

Once that's set, every GraphQL query and mutation is reachable via
the generic `topagentnetwork_query` tool.

## Quick start (library)

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    tan "github.com/teslashibe/topagentnetwork-go"
)

func main() {
    c, err := tan.New(tan.Auth{
        CookieHeader: os.Getenv("TOPAGENTNETWORK_COOKIE_HEADER"),
    })
    if err != nil { log.Fatal(err) }

    me, err := c.GetMe(context.Background())
    if err != nil { log.Fatal(err) }
    fmt.Printf("logged in as %s %s (%s)\n", me.FirstName, me.LastName, me.Email)
}
```

## Supported operations (v0.1)

| Area    | Methods                            |
|---------|------------------------------------|
| Auth    | `Login`, `GetMe`, `AuthSnapshot`   |
| GraphQL | `Query` (generic — any query/mutation) |

## TODO

- [ ] Capture programmatic login HAR from a real browser session and
      replicate the `/api/login` flow (cookies + headers required).
- [ ] Typed wrappers around the most useful queries:
      `alertsForCurrentUser`, `notificationsForCurrentUserV5`,
      `messageBroadcastsForCurrentUser`, `currentAccountV5`,
      `clientStatsForCurrentAccountV5`.
- [ ] Read-only wrappers for the `*ForCurrentUser*` / `*V5` family.

## MCP server

```bash
go install github.com/teslashibe/topagentnetwork-go/cmd/topagentnetwork-mcp@latest
```

Register with Cursor:

```json
{
  "mcpServers": {
    "topagentnetwork": { "command": "/Users/you/go/bin/topagentnetwork-mcp" }
  }
}
```

## License

Private. Internal use only.
