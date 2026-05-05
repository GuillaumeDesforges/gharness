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

const geminiDefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

type Gemini struct {
	APIKey  string
	Model   string
	BaseURL string       // optional; defaults to geminiDefaultBaseURL
	Client  *http.Client // optional; defaults to http.DefaultClient
}

var _ LLM = (*Gemini)(nil)

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
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

func (g *Gemini) Complete(ctx context.Context, messages []Message) (string, error) {
	base := g.BaseURL
	if base == "" {
		base = geminiDefaultBaseURL
	}
	client := g.Client
	if client == nil {
		client = http.DefaultClient
	}

	// Gemini calls the assistant role "model".
	contents := make([]geminiContent, len(messages))
	for i, m := range messages {
		role := string(m.Role)
		if m.Role == RoleAssistant {
			role = "model"
		}
		contents[i] = geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		}
	}

	buf, err := json.Marshal(geminiRequest{Contents: contents})
	if err != nil {
		return "", fmt.Errorf("gemini: marshal: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", base, g.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return "", fmt.Errorf("gemini: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini: do: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini: read body: %w", err)
	}

	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("gemini: status %d: %s", resp.StatusCode, raw)
	}

	var out geminiResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("gemini: decode: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("gemini: %s: %s", out.Error.Status, out.Error.Message)
	}
	if len(out.Candidates) == 0 {
		return "", fmt.Errorf("gemini: no candidates in response")
	}
	var sb strings.Builder
	for _, p := range out.Candidates[0].Content.Parts {
		sb.WriteString(p.Text)
	}
	return sb.String(), nil
}
