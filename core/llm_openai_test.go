package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIEmptyChoicesReturnsError(t *testing.T) {
	// given
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[]}`)
	}))
	defer srv.Close()
	o := &OpenAI{APIKey: "test", Model: "gpt-x", BaseURL: srv.URL}
	msgs := []Message{{Role: RoleUser, Content: "hi"}}

	// when
	_, err := o.Complete(context.Background(), msgs, nil)

	// then
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no choices") {
		t.Errorf("error: got %v, want to contain %q", err, "no choices")
	}
}

func TestOpenAIToolCallArgumentsAreJSONString(t *testing.T) {
	// given
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":""}}]}`)
	}))
	defer srv.Close()
	o := &OpenAI{APIKey: "test", Model: "gpt-x", BaseURL: srv.URL}
	msgs := []Message{
		{Role: RoleUser, Content: "go"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{
			{ID: "call_1", Name: "f", Input: json.RawMessage(`{"x":1}`)},
		}},
		{Role: RoleTool, ToolCallID: "call_1", Content: "done"},
	}

	// when
	_, err := o.Complete(context.Background(), msgs, nil)

	// then
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	// Decode raw to assert arguments is a JSON string, not a JSON object.
	var raw struct {
		Messages []struct {
			Role      string `json:"role"`
			ToolCalls []struct {
				Function struct {
					Arguments json.RawMessage `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(capturedBody, &raw); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	args := raw.Messages[1].ToolCalls[0].Function.Arguments
	if len(args) == 0 || args[0] != '"' {
		t.Errorf("arguments wire form: got %s, want a JSON string", string(args))
	}
	var unquoted string
	if err := json.Unmarshal(args, &unquoted); err != nil {
		t.Fatalf("arguments not a valid JSON string: %v (raw: %s)", err, string(args))
	}
	if unquoted != `{"x":1}` {
		t.Errorf("arguments contents: got %q, want %q", unquoted, `{"x":1}`)
	}
}

func TestOpenAIDecodesToolCalls(t *testing.T) {
	// given
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[
			{"id":"call_1","type":"function","function":{"name":"f","arguments":"{\"x\":1}"}}
		]}}]}`)
	}))
	defer srv.Close()
	o := &OpenAI{APIKey: "test", Model: "gpt-x", BaseURL: srv.URL}
	msgs := []Message{{Role: RoleUser, Content: "hi"}}

	// when
	out, err := o.Complete(context.Background(), msgs, nil)

	// then
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("tool calls: got %d, want 1", len(out.ToolCalls))
	}
	tc := out.ToolCalls[0]
	if tc.ID != "call_1" || tc.Name != "f" || string(tc.Input) != `{"x":1}` {
		t.Errorf("tool call: got %+v", tc)
	}
}
