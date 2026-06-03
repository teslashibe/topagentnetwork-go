package mcp

import (
	"context"

	"github.com/teslashibe/mcptool"
	tan "github.com/teslashibe/topagentnetwork-go"
)

// ListConversationsInput is the input for topagentnetwork_list_conversations.
type ListConversationsInput struct {
	First int    `json:"first,omitempty" jsonschema:"description=Max conversations to return (default 20, max 100)."`
	After string `json:"after,omitempty" jsonschema:"description=Pagination cursor from a previous page's end_cursor."`
}

func listConversations(ctx context.Context, c *tan.Client, in ListConversationsInput) (any, error) {
	return c.ListConversations(ctx, in.First, in.After)
}

// GetConversationInput is the input for topagentnetwork_get_conversation.
type GetConversationInput struct {
	ID       string `json:"id" jsonschema:"description=Conversation id,required"`
	Messages int    `json:"messages,omitempty" jsonschema:"description=Max messages to return (default 50, max 200)."`
}

func getConversation(ctx context.Context, c *tan.Client, in GetConversationInput) (any, error) {
	return c.GetConversation(ctx, in.ID, in.Messages)
}

// ReplyConversationInput is the input for topagentnetwork_reply_conversation.
type ReplyConversationInput struct {
	ConversationID string `json:"conversation_id" jsonschema:"description=Conversation id to reply to,required"`
	Text           string `json:"text" jsonschema:"description=Message body to send,required"`
}

func replyConversation(ctx context.Context, c *tan.Client, in ReplyConversationInput) (any, error) {
	return c.ReplyToConversation(ctx, in.ConversationID, in.Text)
}

// MarkReadInput is the input for topagentnetwork_mark_conversation_read.
type MarkReadInput struct {
	ConversationID    string `json:"conversation_id" jsonschema:"description=Conversation id,required"`
	LastReadMessageID string `json:"last_read_message_id" jsonschema:"description=Mark read up to and including this message id,required"`
}

func markRead(ctx context.Context, c *tan.Client, in MarkReadInput) (any, error) {
	if err := c.MarkConversationRead(ctx, in.ConversationID, in.LastReadMessageID); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

var conversationTools = []mcptool.Tool{
	mcptool.Define[*tan.Client, ListConversationsInput](
		"topagentnetwork_list_conversations",
		"List the signed-in agent's TAN conversations (most recent first) with last message. Paginate via first/after.",
		"ListConversations",
		listConversations,
	),
	mcptool.Define[*tan.Client, GetConversationInput](
		"topagentnetwork_get_conversation",
		"Get a TAN conversation thread by id: participants and recent messages (text, sender, sentAt).",
		"GetConversation",
		getConversation,
	),
	mcptool.Define[*tan.Client, ReplyConversationInput](
		"topagentnetwork_reply_conversation",
		"Send a text reply in an existing TAN conversation. Sends a real message to the other party; not undoable.",
		"ReplyToConversation",
		replyConversation,
	),
	mcptool.Define[*tan.Client, MarkReadInput](
		"topagentnetwork_mark_conversation_read",
		"Mark a TAN conversation's messages read up to a given message id.",
		"MarkConversationRead",
		markRead,
	),
}
