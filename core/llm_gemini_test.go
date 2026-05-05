package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeminiRenamesAssistantRoleToModel(t *testing.T) {
	// given
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]}}]}`)
	}))
	defer srv.Close()
	g := &Gemini{APIKey: "test", Model: "gemini-2.0-flash", BaseURL: srv.URL}
	msgs := []Message{
		{Role: RoleUser, Content: "hi"},
		{Role: RoleAssistant, Content: "hello"},
	}

	// when
	_, err := g.Complete(context.Background(), msgs)

	// then
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	var got geminiRequest
	if err := json.Unmarshal(capturedBody, &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got.Contents) != 2 {
		t.Fatalf("got %d contents, want 2", len(got.Contents))
	}
	if got.Contents[0].Role != "user" {
		t.Errorf("user role: got %q, want %q", got.Contents[0].Role, "user")
	}
	if got.Contents[1].Role != "model" {
		t.Errorf("assistant role: got %q, want %q", got.Contents[1].Role, "model")
	}
}
