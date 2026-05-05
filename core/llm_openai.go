package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const openaiDefaultBaseURL = "https://api.openai.com/v1"

type OpenAI struct {
	APIKey  string
	Model   string
	BaseURL string       // optional; defaults to openaiDefaultBaseURL
	Client  *http.Client // optional; defaults to http.DefaultClient
}

var _ LLM = (*OpenAI)(nil)

type openaiFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // OpenAI sends/expects a JSON string here, not an object
}

type openaiToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openaiFunctionCall `json:"function"`
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openaiToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type openaiTool struct {
	Type     string             `json:"type"`
	Function openaiToolFunction `json:"function"`
}

type openaiRequest struct {
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
	Tools    []openaiTool    `json:"tools,omitempty"`
}

type openaiResponse struct {
	Choices []struct {
		Message openaiMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (o *OpenAI) Complete(ctx context.Context, messages []Message, tools []Tool) (Message, error) {
	base := o.BaseURL
	if base == "" {
		base = openaiDefaultBaseURL
	}
	client := o.Client
	if client == nil {
		client = http.DefaultClient
	}

	body := openaiRequest{
		Model:    o.Model,
		Messages: make([]openaiMessage, len(messages)),
	}
	for i, m := range messages {
		om := openaiMessage{Role: string(m.Role), Content: m.Content}
		if m.Role == RoleTool {
			om.ToolCallID = m.ToolCallID
		}
		for _, tc := range m.ToolCalls {
			om.ToolCalls = append(om.ToolCalls, openaiToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: openaiFunctionCall{
					Name:      tc.Name,
					Arguments: string(tc.Input),
				},
			})
		}
		body.Messages[i] = om
	}
	if len(tools) > 0 {
		body.Tools = make([]openaiTool, len(tools))
		for i, t := range tools {
			body.Tools[i] = openaiTool{
				Type: "function",
				Function: openaiToolFunction{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.Schema,
				},
			}
		}
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return Message{}, fmt.Errorf("openai: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return Message{}, fmt.Errorf("openai: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("openai: do: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Message{}, fmt.Errorf("openai: read body: %w", err)
	}

	if resp.StatusCode/100 != 2 {
		return Message{}, fmt.Errorf("openai: status %d: %s", resp.StatusCode, raw)
	}

	var out openaiResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return Message{}, fmt.Errorf("openai: decode: %w", err)
	}
	if out.Error != nil {
		return Message{}, fmt.Errorf("openai: %s: %s", out.Error.Type, out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return Message{}, fmt.Errorf("openai: no choices in response")
	}

	choice := out.Choices[0].Message
	reply := Message{Role: RoleAssistant, Content: choice.Content}
	for _, tc := range choice.ToolCalls {
		reply.ToolCalls = append(reply.ToolCalls, ToolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(tc.Function.Arguments),
		})
	}
	return reply, nil
}
