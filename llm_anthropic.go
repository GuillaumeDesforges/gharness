package gharness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (a *Anthropic) Complete(ctx context.Context, messages []Message) (string, error) {
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
		Messages:  make([]anthropicMessage, len(messages)),
	}
	for i, m := range messages {
		body.Messages[i] = anthropicMessage{Role: string(m.Role), Content: m.Content}
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("anthropic: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/messages", bytes.NewReader(buf))
	if err != nil {
		return "", fmt.Errorf("anthropic: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: do: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("anthropic: read body: %w", err)
	}

	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("anthropic: status %d: %s", resp.StatusCode, raw)
	}

	var out anthropicResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("anthropic: decode: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("anthropic: %s: %s", out.Error.Type, out.Error.Message)
	}
	var sb strings.Builder
	for _, block := range out.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	return sb.String(), nil
}
