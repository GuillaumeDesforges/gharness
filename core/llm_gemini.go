package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const geminiDefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

type Gemini struct {
	APIKey  string
	Model   string
	BaseURL string       // optional; defaults to geminiDefaultBaseURL
	Client  *http.Client // optional; defaults to http.DefaultClient
}

var _ LLM = (*Gemini)(nil)

type geminiFunctionCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

type geminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiFunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type geminiToolBlock struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations"`
}

type geminiRequest struct {
	Contents []geminiContent   `json:"contents"`
	Tools    []geminiToolBlock `json:"tools,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

func (g *Gemini) Complete(ctx context.Context, messages []Message, tools []Tool) (Message, error) {
	base := g.BaseURL
	if base == "" {
		base = geminiDefaultBaseURL
	}
	client := g.Client
	if client == nil {
		client = http.DefaultClient
	}

	contents, err := geminiEncodeMessages(messages)
	if err != nil {
		return Message{}, err
	}

	body := geminiRequest{Contents: contents}
	if len(tools) > 0 {
		decls := make([]geminiFunctionDeclaration, len(tools))
		for i, t := range tools {
			decls[i] = geminiFunctionDeclaration{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Schema,
			}
		}
		body.Tools = []geminiToolBlock{{FunctionDeclarations: decls}}
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return Message{}, fmt.Errorf("gemini: marshal: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", base, g.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return Message{}, fmt.Errorf("gemini: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("gemini: do: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Message{}, fmt.Errorf("gemini: read body: %w", err)
	}

	if resp.StatusCode/100 != 2 {
		return Message{}, fmt.Errorf("gemini: status %d: %s", resp.StatusCode, raw)
	}

	var out geminiResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return Message{}, fmt.Errorf("gemini: decode: %w", err)
	}
	if out.Error != nil {
		return Message{}, fmt.Errorf("gemini: %s: %s", out.Error.Status, out.Error.Message)
	}
	if len(out.Candidates) == 0 {
		return Message{}, fmt.Errorf("gemini: no candidates in response")
	}

	reply := Message{Role: RoleAssistant}
	callIdx := 0
	for _, p := range out.Candidates[0].Content.Parts {
		if p.Text != "" {
			reply.Content += p.Text
		}
		if p.FunctionCall != nil {
			reply.ToolCalls = append(reply.ToolCalls, ToolCall{
				ID:    fmt.Sprintf("%s-%d", p.FunctionCall.Name, callIdx),
				Name:  p.FunctionCall.Name,
				Input: p.FunctionCall.Args,
			})
			callIdx++
		}
	}
	return reply, nil
}

// geminiEncodeMessages translates gharness messages into Gemini's content
// shape. Gemini calls the assistant role "model" and represents tool
// results as functionResponse parts inside a user-role content. Gemini
// does not carry call IDs on the wire, so RoleTool messages are matched
// to the original tool_use by function name — looked up from prior
// assistant turns.
func geminiEncodeMessages(messages []Message) ([]geminiContent, error) {
	contents := make([]geminiContent, 0, len(messages))
	for i, m := range messages {
		switch m.Role {
		case RoleTool:
			name, ok := geminiLookupCallName(messages[:i], m.ToolCallID)
			if !ok {
				return nil, fmt.Errorf("gemini: tool result references unknown call ID %q", m.ToolCallID)
			}
			contents = append(contents, geminiContent{
				Role: "user",
				Parts: []geminiPart{{
					FunctionResponse: &geminiFunctionResponse{
						Name:     name,
						Response: map[string]any{"content": m.Content},
					},
				}},
			})
		case RoleAssistant:
			parts := make([]geminiPart, 0, 1+len(m.ToolCalls))
			if m.Content != "" {
				parts = append(parts, geminiPart{Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				parts = append(parts, geminiPart{
					FunctionCall: &geminiFunctionCall{
						Name: tc.Name,
						Args: tc.Input,
					},
				})
			}
			contents = append(contents, geminiContent{Role: "model", Parts: parts})
		default: // RoleUser
			contents = append(contents, geminiContent{
				Role:  "user",
				Parts: []geminiPart{{Text: m.Content}},
			})
		}
	}
	return contents, nil
}

func geminiLookupCallName(prev []Message, callID string) (string, bool) {
	for _, m := range prev {
		for _, tc := range m.ToolCalls {
			if tc.ID == callID {
				return tc.Name, true
			}
		}
	}
	return "", false
}
