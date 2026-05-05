package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnthropicConcatenatesTextBlocksOnly(t *testing.T) {
	// given
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"content":[
			{"type":"text","text":"hello "},
			{"type":"tool_use","id":"x","name":"y","input":{}},
			{"type":"text","text":"world"}
		]}`)
	}))
	defer srv.Close()
	a := &Anthropic{APIKey: "test", Model: "claude-x", BaseURL: srv.URL}
	msgs := []Message{{Role: RoleUser, Content: "hi"}}

	// when
	out, err := a.Complete(context.Background(), msgs, nil)

	// then
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if out.Content != "hello world" {
		t.Errorf("content: got %q, want %q", out.Content, "hello world")
	}
	if len(out.ToolCalls) != 1 || out.ToolCalls[0].ID != "x" || out.ToolCalls[0].Name != "y" {
		t.Errorf("tool calls: got %+v, want one call ID=x Name=y", out.ToolCalls)
	}
}

func TestAnthropicDefaultMaxTokens(t *testing.T) {
	// given
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"content":[{"type":"text","text":"ok"}]}`)
	}))
	defer srv.Close()
	a := &Anthropic{APIKey: "test", Model: "claude-x", BaseURL: srv.URL} // MaxTokens unset
	msgs := []Message{{Role: RoleUser, Content: "hi"}}

	// when
	_, err := a.Complete(context.Background(), msgs, nil)

	// then
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	var got anthropicRequest
	if err := json.Unmarshal(capturedBody, &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.MaxTokens != anthropicDefaultMaxTokens {
		t.Errorf("MaxTokens: got %d, want %d", got.MaxTokens, anthropicDefaultMaxTokens)
	}
}

func TestAnthropicMergesConsecutiveToolResults(t *testing.T) {
	// given
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"content":[{"type":"text","text":"ok"}]}`)
	}))
	defer srv.Close()
	a := &Anthropic{APIKey: "test", Model: "claude-x", BaseURL: srv.URL}
	msgs := []Message{
		{Role: RoleUser, Content: "go"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{
			{ID: "a", Name: "f", Input: json.RawMessage(`{}`)},
			{ID: "b", Name: "g", Input: json.RawMessage(`{}`)},
		}},
		{Role: RoleTool, ToolCallID: "a", Content: "result-a"},
		{Role: RoleTool, ToolCallID: "b", Content: "result-b"},
	}

	// when
	_, err := a.Complete(context.Background(), msgs, nil)

	// then
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	var got anthropicRequest
	if err := json.Unmarshal(capturedBody, &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if n := len(got.Messages); n != 3 {
		t.Fatalf("messages: got %d, want 3 (user, assistant, merged-user)", n)
	}
	last := got.Messages[2]
	if last.Role != "user" {
		t.Errorf("last role: got %q, want user", last.Role)
	}
	if len(last.Content) != 2 {
		t.Fatalf("merged tool_result blocks: got %d, want 2", len(last.Content))
	}
	for i, b := range last.Content {
		if b.Type != "tool_result" {
			t.Errorf("block %d type: got %q, want tool_result", i, b.Type)
		}
	}
}
