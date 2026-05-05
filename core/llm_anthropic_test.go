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
	out, err := a.Complete(context.Background(), msgs)

	// then
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if out != "hello world" {
		t.Errorf("got %q, want %q", out, "hello world")
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
	_, err := a.Complete(context.Background(), msgs)

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
