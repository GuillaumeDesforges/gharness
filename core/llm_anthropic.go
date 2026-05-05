package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	anthropicDefaultBaseURL   = "https://api.anthropic.com/v1"
	anthropicVersion          = "2023-06-01"
	anthropicDefaultMaxTokens = 4096
)

type Anthropic struct {
	APIKey    string
	Model     string
	MaxTokens int          // optional; defaults to anthropicDefaultMaxTokens (Anthropic requires it)
	BaseURL   string       // optional; defaults to anthropicDefaultBaseURL
	Client    *http.Client // optional; defaults to http.DefaultClient
}

var _ LLM = (*Anthropic)(nil)

type anthropicBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
}

type anthropicMessage struct {
	Role    string           `json:"role"`
	Content []anthropicBlock `json:"content"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicResponse struct {
	Content []anthropicBlock `json:"content"`
	Error   *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (a *Anthropic) Complete(ctx context.Context, messages []Message, tools []Tool) (Message, error) {
	base := a.BaseURL
	if base == "" {
		base = anthropicDefaultBaseURL
	}
	client := a.Client
	if client == nil {
		client = http.DefaultClient
	}
	maxTokens := a.MaxTokens
	if maxTokens <= 0 {
		maxTokens = anthropicDefaultMaxTokens
	}

	body := anthropicRequest{
		Model:     a.Model,
		MaxTokens: maxTokens,
		Messages:  anthropicEncodeMessages(messages),
	}
	if len(tools) > 0 {
		body.Tools = make([]anthropicTool, len(tools))
		for i, t := range tools {
			body.Tools[i] = anthropicTool{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.Schema,
			}
		}
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return Message{}, fmt.Errorf("anthropic: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/messages", bytes.NewReader(buf))
	if err != nil {
		return Message{}, fmt.Errorf("anthropic: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := client.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("anthropic: do: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Message{}, fmt.Errorf("anthropic: read body: %w", err)
	}

	if resp.StatusCode/100 != 2 {
		return Message{}, fmt.Errorf("anthropic: status %d: %s", resp.StatusCode, raw)
	}

	var out anthropicResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return Message{}, fmt.Errorf("anthropic: decode: %w", err)
	}
	if out.Error != nil {
		return Message{}, fmt.Errorf("anthropic: %s: %s", out.Error.Type, out.Error.Message)
	}

	reply := Message{Role: RoleAssistant}
	for _, b := range out.Content {
		switch b.Type {
		case "text":
			reply.Content += b.Text
		case "tool_use":
			reply.ToolCalls = append(reply.ToolCalls, ToolCall{
				ID:    b.ID,
				Name:  b.Name,
				Input: b.Input,
			})
		}
	}
	return reply, nil
}

// anthropicEncodeMessages translates gharness messages into Anthropic's
// block-list shape. Consecutive RoleTool messages collapse into a single
// user message with multiple tool_result blocks, which is what the API
// requires when the previous assistant turn produced multiple tool_use
// blocks.
func anthropicEncodeMessages(messages []Message) []anthropicMessage {
	out := make([]anthropicMessage, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case RoleTool:
			block := anthropicBlock{
				Type:      "tool_result",
				ToolUseID: m.ToolCallID,
				Content:   m.Content,
			}
			if n := len(out); n > 0 && out[n-1].Role == "user" && anthropicAllToolResults(out[n-1].Content) {
				out[n-1].Content = append(out[n-1].Content, block)
				continue
			}
			out = append(out, anthropicMessage{Role: "user", Content: []anthropicBlock{block}})
		case RoleAssistant:
			blocks := make([]anthropicBlock, 0, 1+len(m.ToolCalls))
			if m.Content != "" {
				blocks = append(blocks, anthropicBlock{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				blocks = append(blocks, anthropicBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: tc.Input,
				})
			}
			out = append(out, anthropicMessage{Role: "assistant", Content: blocks})
		default: // RoleUser
			out = append(out, anthropicMessage{
				Role:    "user",
				Content: []anthropicBlock{{Type: "text", Text: m.Content}},
			})
		}
	}
	return out
}

func anthropicAllToolResults(blocks []anthropicBlock) bool {
	for _, b := range blocks {
		if b.Type != "tool_result" {
			return false
		}
	}
	return len(blocks) > 0
}
