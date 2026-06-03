# topagentnetwork-go

Private Go client + MCP server for [topagentnetwork.app](https://topagentnetwork.app)
("TAN"). There is no public API. The website is a Next.js SPA that
talks to a GraphQL backend at `https://api.topagentnetwork.app/graphql`.
Schema is large and useful: **79 queries** (incl. `currentlyLoggedInAgent`,
`sessionV5`, `messageBroadcasts`, `alertsForCurrentUser`,
`accountConversations`) and **140 mutations** — counts confirmed via live
introspection 2026-06-03. Introspection is **enabled** in production, so
the full schema is discoverable at runtime (see `topagentnetwork_introspect`).

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

## Library methods (`*tan.Client`)

| Area         | Method                                                     | Wraps (GraphQL)                       |
|--------------|------------------------------------------------------------|---------------------------------------|
| Auth         | `New`, `AuthSnapshot`                                       | —                                     |
| Auth         | `Login`                                                     | `currentlyLoggedInAgent`              |
| Auth         | `GetMe`                                                     | `currentlyLoggedInAgent`              |
| GraphQL      | `Query` (generic — any query/mutation)                     | arbitrary                             |
| Schema       | `Introspect(typeName)`, `IntrospectSchema()`               | `__type` / `__schema`                 |
| Conversation | `ListConversations(first, after)`                          | `accountConversations`                |
| Conversation | `GetConversation(id, messages)`                            | `conversationById`                    |
| Conversation | `ReplyToConversation(id, text)`                            | `replyToConversationV5`               |
| Conversation | `MarkConversationRead(id, lastReadMessageID)`              | `markConversationMessagesAsReadV5`    |

## MCP tools

All tools are prefixed `topagentnetwork_`. The generic `query` + `introspect`
pair makes **every** one of the 79 queries / 140 mutations reachable; the
typed conversation tools wrap the most common flows for reliability.

| Tool                                  | Kind  | Description                                                        |
|---------------------------------------|-------|-------------------------------------------------------------------|
| `topagentnetwork_login`               | read  | Validate the pasted cookie via `currentlyLoggedInAgent`.          |
| `topagentnetwork_get_me`              | read  | Return the authenticated agent's profile.                         |
| `topagentnetwork_introspect`          | read  | Introspect the real schema (fields, args, SDL types like `[ID!]!`). Omit `type_name` for all root fields. |
| `topagentnetwork_query`               | r/w   | Run a raw GraphQL query/mutation. Introspect first for valid fields. |
| `topagentnetwork_list_conversations`  | read  | List the agent's conversations (recent first) with last message.  |
| `topagentnetwork_get_conversation`    | read  | Get one conversation thread: participants + recent messages.      |
| `topagentnetwork_reply_conversation`  | write | Send a text reply in a conversation. **Real message; not undoable.** |
| `topagentnetwork_mark_conversation_read` | write | Mark a conversation read up to a message id.                   |

## Tested status

Validated live against `api.topagentnetwork.app/graphql` on 2026-06-03
(account `margie@pardeeproperties.com`). "Fully tested" = executed live with
a real response; "Schema-validated" = query/input shape confirmed via
introspection but not executed (write side effects).

| Tool / method                            | Status            | Evidence                                              |
|------------------------------------------|-------------------|-------------------------------------------------------|
| `topagentnetwork_introspect`             | ✅ Fully tested    | Root + `__type` introspection returned 79/140 fields. |
| `topagentnetwork_query`                  | ✅ Fully tested    | Used for all live read/write probes below.            |
| `topagentnetwork_list_conversations`     | ✅ Fully tested    | Returned 33 conversations with last message.          |
| `topagentnetwork_get_conversation`       | ✅ Fully tested    | Returned a thread with 2 participants + messages.     |
| `topagentnetwork_mark_conversation_read` | ✅ Fully tested    | `ConversationMessagesMarkedAsRead` returned.          |
| `topagentnetwork_reply_conversation`     | ✅ Fully tested    | Sent a real reply (msg `e9ecb0df…`); returned `ConversationReply`. |
| `topagentnetwork_login` / `_get_me`      | ⚠️ Untested (v0.2) | Pre-existing; not re-run this round.                  |

Unit test coverage: `introspect_test.go` pins SDL rendering (incl. `[ID!]!`)
and the root-Query digest. Run `go test ./...`.

## TODO

- [ ] Capture programmatic login HAR from a real browser session and
      replicate the `/api/login` flow (cookies + headers required).
- [ ] Live-test `topagentnetwork_login` / `topagentnetwork_get_me` and the
      broadcast write tools (create/delete) against a safe account.
- [ ] Typed wrappers for more high-value queries:
      `alertsForCurrentUser`, `notificationsForCurrentUserV5`,
      `messageBroadcastsForCurrentUser`, `currentAccountV5`,
      `clientStatsForCurrentAccountV5`.

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
