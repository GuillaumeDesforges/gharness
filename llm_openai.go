package gharness

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

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiRequest struct {
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
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

func (o *OpenAI) Complete(ctx context.Context, messages []Message) (string, error) {
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
		body.Messages[i] = openaiMessage{Role: string(m.Role), Content: m.Content}
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("openai: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return "", fmt.Errorf("openai: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai: do: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("openai: read body: %w", err)
	}

	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("openai: status %d: %s", resp.StatusCode, raw)
	}

	var out openaiResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("openai: decode: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("openai: %s: %s", out.Error.Type, out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("openai: no choices in response")
	}
	return out.Choices[0].Message.Content, nil
}
