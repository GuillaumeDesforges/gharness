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
	_, err := g.Complete(context.Background(), msgs, nil)

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

func TestGeminiTranslatesToolResultsToFunctionResponse(t *testing.T) {
	// given — Gemini has no call IDs on the wire; the provider must look up
	// the function name from the prior assistant turn that issued the call.
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]}}]}`)
	}))
	defer srv.Close()
	g := &Gemini{APIKey: "test", Model: "gemini-2.0-flash", BaseURL: srv.URL}
	msgs := []Message{
		{Role: RoleUser, Content: "go"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{
			{ID: "f-0", Name: "f", Input: json.RawMessage(`{}`)},
		}},
		{Role: RoleTool, ToolCallID: "f-0", Content: "result"},
	}

	// when
	_, err := g.Complete(context.Background(), msgs, nil)

	// then
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	var got geminiRequest
	if err := json.Unmarshal(capturedBody, &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got.Contents) != 3 {
		t.Fatalf("contents: got %d, want 3", len(got.Contents))
	}
	last := got.Contents[2]
	if last.Role != "user" {
		t.Errorf("tool-result role: got %q, want user", last.Role)
	}
	if len(last.Parts) != 1 || last.Parts[0].FunctionResponse == nil {
		t.Fatalf("expected one functionResponse part, got %+v", last.Parts)
	}
	fr := last.Parts[0].FunctionResponse
	if fr.Name != "f" {
		t.Errorf("functionResponse.name: got %q, want %q (looked up from prior tool_use)", fr.Name, "f")
	}
}

func TestGeminiSynthesizesCallIDs(t *testing.T) {
	// given — Gemini responses don't carry call IDs; the provider invents
	// one per call so the agent can match results back.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"candidates":[{"content":{"role":"model","parts":[
			{"functionCall":{"name":"f","args":{"x":1}}}
		]}}]}`)
	}))
	defer srv.Close()
	g := &Gemini{APIKey: "test", Model: "gemini-2.0-flash", BaseURL: srv.URL}
	msgs := []Message{{Role: RoleUser, Content: "hi"}}

	// when
	out, err := g.Complete(context.Background(), msgs, nil)

	// then
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("tool calls: got %d, want 1", len(out.ToolCalls))
	}
	if out.ToolCalls[0].ID == "" {
		t.Errorf("expected synthesized call ID, got empty string")
	}
}
