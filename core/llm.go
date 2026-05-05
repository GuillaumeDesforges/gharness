package core

import "context"

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role
	Content string
}

// LLM is the bare-bones interface for a language model: given a conversation,
// return the assistant's next reply as text.
type LLM interface {
	Complete(ctx context.Context, messages []Message) (string, error)
}
