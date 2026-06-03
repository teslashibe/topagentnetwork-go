package topagentnetwork

import (
	"context"
	"encoding/json"
	"fmt"
)

// This file provides typed helpers for TAN's conversation/messaging domain.
// They wrap the raw GraphQL surface (accountConversations, conversationById,
// replyToConversationV5, markConversationMessagesAsReadV5) so callers get
// reliable read/write of conversations without hand-composing GraphQL.

// ConversationParticipant is one party in a conversation. Type is the
// GraphQL __typename ("Agent" or "AnonymousConversationParticipant").
type ConversationParticipant struct {
	Type        string `json:"type"`
	ID          string `json:"id,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

// ConversationMessage is a single message in a conversation thread.
type ConversationMessage struct {
	ID     string                   `json:"id"`
	Text   string                   `json:"text,omitempty"`
	SentAt string                   `json:"sentAt,omitempty"`
	From   *ConversationParticipant `json:"from,omitempty"`
}

// ConversationSummary is the compact view returned when listing conversations.
type ConversationSummary struct {
	ID          string               `json:"id"`
	Anonymous   bool                 `json:"anonymous"`
	ReadOnly    bool                 `json:"readOnly"`
	LastMessage *ConversationMessage `json:"lastMessage,omitempty"`
}

// ConversationList is one page of conversations.
type ConversationList struct {
	TotalCount    int                   `json:"totalCount"`
	Conversations []ConversationSummary `json:"conversations"`
	HasNextPage   bool                  `json:"hasNextPage"`
	EndCursor     string                `json:"endCursor,omitempty"`
}

// Conversation is a full thread with participants and messages.
type Conversation struct {
	ID           string                    `json:"id"`
	Anonymous    bool                      `json:"anonymous"`
	ReadOnly     bool                      `json:"readOnly"`
	Participants []ConversationParticipant `json:"participants,omitempty"`
	Messages     []ConversationMessage     `json:"messages,omitempty"`
}

// participantFragment selects identity fields across the participant union.
const participantFragment = `from { __typename ... on Agent { id displayName } ... on AnonymousConversationParticipant { __typename } }`

// rawParticipant decodes the __typename-tagged participant union.
type rawParticipant struct {
	Typename    string `json:"__typename"`
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

func (r *rawParticipant) convert() *ConversationParticipant {
	if r == nil || r.Typename == "" {
		return nil
	}
	return &ConversationParticipant{Type: r.Typename, ID: r.ID, DisplayName: r.DisplayName}
}

type rawMessage struct {
	ID     string          `json:"id"`
	Text   string          `json:"text"`
	SentAt string          `json:"sentAt"`
	From   *rawParticipant `json:"from"`
}

func (r *rawMessage) convert() *ConversationMessage {
	if r == nil {
		return nil
	}
	return &ConversationMessage{ID: r.ID, Text: r.Text, SentAt: r.SentAt, From: r.From.convert()}
}

const listConversationsQuery = `query AccountConversations($first: Int, $after: String) {
  accountConversations(first: $first, after: $after) {
    totalCount
    pageInfo { hasNextPage endCursor }
    edges { node { id anonymous readOnly lastMessage { id text sentAt ` + participantFragment + ` } } }
  }
}`

// ListConversations returns one page of the signed-in agent's conversations,
// most recent first. first defaults to 20 (clamped to 1..100); pass the
// previous page's EndCursor as after to paginate.
func (c *Client) ListConversations(ctx context.Context, first int, after string) (*ConversationList, error) {
	if first <= 0 {
		first = 20
	}
	if first > 100 {
		first = 100
	}
	vars := map[string]any{"first": first}
	if after != "" {
		vars["after"] = after
	}
	resp, err := c.Query(ctx, GraphQLRequest{OperationName: "AccountConversations", Query: listConversationsQuery, Variables: vars})
	if err != nil {
		return nil, err
	}
	var raw struct {
		AccountConversations struct {
			TotalCount int `json:"totalCount"`
			PageInfo   struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Edges []struct {
				Node struct {
					ID          string      `json:"id"`
					Anonymous   bool        `json:"anonymous"`
					ReadOnly    bool        `json:"readOnly"`
					LastMessage *rawMessage `json:"lastMessage"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"accountConversations"`
	}
	if err := decodeData(resp, &raw); err != nil {
		return nil, err
	}
	out := &ConversationList{
		TotalCount:  raw.AccountConversations.TotalCount,
		HasNextPage: raw.AccountConversations.PageInfo.HasNextPage,
		EndCursor:   raw.AccountConversations.PageInfo.EndCursor,
	}
	for _, e := range raw.AccountConversations.Edges {
		out.Conversations = append(out.Conversations, ConversationSummary{
			ID:          e.Node.ID,
			Anonymous:   e.Node.Anonymous,
			ReadOnly:    e.Node.ReadOnly,
			LastMessage: e.Node.LastMessage.convert(),
		})
	}
	return out, nil
}

const getConversationQuery = `query ConversationById($id: ID!, $messages: Int) {
  conversationById(id: $id) {
    __typename
    ... on ConversationV5 {
      id anonymous readOnly
      participants(first: 25) { edges { node { __typename ... on Agent { id displayName } ... on AnonymousConversationParticipant { __typename } } } }
      messages(first: $messages) { edges { node { id text sentAt ` + participantFragment + ` } } }
    }
    ... on NotAuthorizedError { __typename }
  }
}`

// GetConversation fetches a single conversation thread by id, including up to
// `messages` most-recent messages (defaults to 50, clamped to 1..200) and its
// participants.
func (c *Client) GetConversation(ctx context.Context, id string, messages int) (*Conversation, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: conversation id is required", ErrInvalidParams)
	}
	if messages <= 0 {
		messages = 50
	}
	if messages > 200 {
		messages = 200
	}
	resp, err := c.Query(ctx, GraphQLRequest{
		OperationName: "ConversationById",
		Query:         getConversationQuery,
		Variables:     map[string]any{"id": id, "messages": messages},
	})
	if err != nil {
		return nil, err
	}
	var raw struct {
		ConversationByID struct {
			Typename     string `json:"__typename"`
			ID           string `json:"id"`
			Anonymous    bool   `json:"anonymous"`
			ReadOnly     bool   `json:"readOnly"`
			Participants struct {
				Edges []struct {
					Node rawParticipant `json:"node"`
				} `json:"edges"`
			} `json:"participants"`
			Messages struct {
				Edges []struct {
					Node rawMessage `json:"node"`
				} `json:"edges"`
			} `json:"messages"`
		} `json:"conversationById"`
	}
	if err := decodeData(resp, &raw); err != nil {
		return nil, err
	}
	cv := raw.ConversationByID
	if cv.Typename == "NotAuthorizedError" {
		return nil, fmt.Errorf("%w: not authorized to view conversation %q", ErrForbidden, id)
	}
	if cv.ID == "" {
		return nil, fmt.Errorf("%w: conversation %q not found", ErrNotFound, id)
	}
	out := &Conversation{ID: cv.ID, Anonymous: cv.Anonymous, ReadOnly: cv.ReadOnly}
	for _, e := range cv.Participants.Edges {
		if p := e.Node.convert(); p != nil {
			out.Participants = append(out.Participants, *p)
		}
	}
	for _, e := range cv.Messages.Edges {
		if m := e.Node.convert(); m != nil {
			out.Messages = append(out.Messages, *m)
		}
	}
	return out, nil
}

const replyConversationMutation = `mutation ReplyToConversation($input: ReplyToConversationV5Input!) {
  replyToConversationV5(input: $input) {
    __typename
    ... on ConversationReply { message { id text sentAt ` + participantFragment + ` } }
    ... on ConversationIsReadOnlyError { __typename }
  }
}`

// ReplyToConversation posts a text reply to an existing conversation and
// returns the created message. It fails if the conversation is read-only.
//
// This sends a real message to the other participant(s) — there is no delete.
func (c *Client) ReplyToConversation(ctx context.Context, conversationID, text string) (*ConversationMessage, error) {
	if conversationID == "" {
		return nil, fmt.Errorf("%w: conversation id is required", ErrInvalidParams)
	}
	if text == "" {
		return nil, fmt.Errorf("%w: reply text is required", ErrInvalidParams)
	}
	resp, err := c.Query(ctx, GraphQLRequest{
		OperationName: "ReplyToConversation",
		Query:         replyConversationMutation,
		Variables:     map[string]any{"input": map[string]any{"conversationId": conversationID, "text": text}},
	})
	if err != nil {
		return nil, err
	}
	var raw struct {
		Reply struct {
			Typename string      `json:"__typename"`
			Message  *rawMessage `json:"message"`
		} `json:"replyToConversationV5"`
	}
	if err := decodeData(resp, &raw); err != nil {
		return nil, err
	}
	if raw.Reply.Typename == "ConversationIsReadOnlyError" {
		return nil, fmt.Errorf("%w: conversation %q is read-only", ErrForbidden, conversationID)
	}
	return raw.Reply.Message.convert(), nil
}

const markReadMutation = `mutation MarkConversationRead($input: MarkConversationMessagesAsReadV5Input!) {
  markConversationMessagesAsReadV5(input: $input) {
    __typename
    ... on ConversationMessagesMarkedAsRead { __typename }
  }
}`

// MarkConversationRead marks messages in a conversation as read up to and
// including lastReadMessageID.
func (c *Client) MarkConversationRead(ctx context.Context, conversationID, lastReadMessageID string) error {
	if conversationID == "" || lastReadMessageID == "" {
		return fmt.Errorf("%w: conversationId and lastReadMessageId are required", ErrInvalidParams)
	}
	_, err := c.Query(ctx, GraphQLRequest{
		OperationName: "MarkConversationRead",
		Query:         markReadMutation,
		Variables:     map[string]any{"input": map[string]any{"conversationId": conversationID, "lastReadMessageId": lastReadMessageID}},
	})
	return err
}

// decodeData re-marshals a GraphQL response's data payload into out.
func decodeData(resp *GraphQLResponse, out any) error {
	b, err := json.Marshal(resp.Data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("%w: malformed response payload: %v", ErrRequestFailed, err)
	}
	return nil
}
