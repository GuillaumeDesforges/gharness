package core

import (
	"context"
	"encoding/json"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is one turn in a conversation.
//
// Field meaning depends on Role. ToolCalls is set on assistant turns that
// invoke tools; ToolCallID is set on tool-result turns and references the
// call this turn answers. Invariants are not enforced by the type — providers
// translate fields that don't apply to a given role into the empty case.
type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

// Tool is a capability the model can invoke. Definitions are values; a
// stateful implementation closes over its state in Execute.
type Tool struct {
	Name        string
	Description string
	Schema      map[string]any
	Execute     func(ctx context.Context, input json.RawMessage) (string, error)
}

// ToolCall is the model's request to invoke a tool. ID is provider-issued
// and opaque; the agent uses it to match the result back to the call.
type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// LLM produces the next assistant turn given a conversation and the tools
// the model is allowed to call. Returned Message always has Role
// RoleAssistant; len(ToolCalls) > 0 indicates the caller should execute
// tools and continue.
type LLM interface {
	Complete(ctx context.Context, messages []Message, tools []Tool) (Message, error)
}
