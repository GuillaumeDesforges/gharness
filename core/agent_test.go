package core

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type scriptedLLM struct {
	replies []Message
	calls   int
}

func (s *scriptedLLM) Complete(_ context.Context, _ []Message, _ []Tool) (Message, error) {
	if s.calls >= len(s.replies) {
		return Message{}, errors.New("scriptedLLM: out of replies")
	}
	reply := s.replies[s.calls]
	s.calls++
	return reply, nil
}

type erroringLLM struct{ err error }

func (e *erroringLLM) Complete(_ context.Context, _ []Message, _ []Tool) (Message, error) {
	return Message{}, e.err
}

func TestFixedPointAgentStopsWhenAssistantHasNoToolCalls(t *testing.T) {
	// given
	llm := &scriptedLLM{replies: []Message{
		{Role: RoleAssistant, Content: "hi back"},
	}}
	agent := &FixedPointAgent{LLM: llm}

	// when
	history, err := agent.Run(context.Background(), []Message{
		{Role: RoleUser, Content: "hi"},
	})

	// then
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("history length: got %d, want 2", len(history))
	}
	if history[1].Content != "hi back" {
		t.Errorf("final content: got %q, want %q", history[1].Content, "hi back")
	}
}

func TestFixedPointAgentExecutesToolCallAndLoops(t *testing.T) {
	// given
	executed := false
	llm := &scriptedLLM{replies: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{
			{ID: "c1", Name: "echo", Input: json.RawMessage(`{}`)},
		}},
		{Role: RoleAssistant, Content: "done"},
	}}
	agent := &FixedPointAgent{
		LLM: llm,
		Tools: []Tool{{
			Name: "echo",
			Execute: func(_ context.Context, _ json.RawMessage) (string, error) {
				executed = true
				return "ok", nil
			},
		}},
	}

	// when
	history, err := agent.Run(context.Background(), []Message{
		{Role: RoleUser, Content: "go"},
	})

	// then
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !executed {
		t.Fatal("tool was not executed")
	}
	if len(history) != 4 {
		t.Fatalf("history length: got %d, want 4", len(history))
	}
	tr := history[2]
	if tr.Role != RoleTool || tr.ToolCallID != "c1" || tr.Content != "ok" {
		t.Errorf("tool result: got %+v, want RoleTool/c1/ok", tr)
	}
	if history[3].Content != "done" {
		t.Errorf("final content: got %q, want %q", history[3].Content, "done")
	}
}

func TestFixedPointAgentToolErrorBecomesToolResult(t *testing.T) {
	// given
	llm := &scriptedLLM{replies: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Name: "boom"}}},
		{Role: RoleAssistant, Content: "recovered"},
	}}
	agent := &FixedPointAgent{
		LLM: llm,
		Tools: []Tool{{
			Name: "boom",
			Execute: func(_ context.Context, _ json.RawMessage) (string, error) {
				return "", errors.New("kaboom")
			},
		}},
	}

	// when
	history, err := agent.Run(context.Background(), nil)

	// then
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	tr := history[1]
	if tr.Role != RoleTool || tr.Content != "kaboom" {
		t.Errorf("tool result: got %+v, want RoleTool with err.Error() as content", tr)
	}
}

func TestFixedPointAgentUnknownToolBecomesToolResult(t *testing.T) {
	// given
	llm := &scriptedLLM{replies: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Name: "missing"}}},
		{Role: RoleAssistant, Content: "recovered"},
	}}
	agent := &FixedPointAgent{LLM: llm}

	// when
	history, err := agent.Run(context.Background(), nil)

	// then
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	tr := history[1]
	if tr.Role != RoleTool || !strings.Contains(tr.Content, "missing") {
		t.Errorf("tool result: got %+v, want RoleTool mentioning the tool name", tr)
	}
}

func TestFixedPointAgentMaxStepsCapsLoop(t *testing.T) {
	// given
	llm := &scriptedLLM{replies: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Name: "noop"}}},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c2", Name: "noop"}}},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c3", Name: "noop"}}},
	}}
	agent := &FixedPointAgent{
		LLM: llm,
		Tools: []Tool{{
			Name:    "noop",
			Execute: func(_ context.Context, _ json.RawMessage) (string, error) { return "", nil },
		}},
		MaxSteps: 2,
	}

	// when
	_, err := agent.Run(context.Background(), nil)

	// then
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if llm.calls != 2 {
		t.Errorf("LLM calls: got %d, want 2", llm.calls)
	}
}

func TestFixedPointAgentReturnsPartialHistoryOnError(t *testing.T) {
	// given
	agent := &FixedPointAgent{LLM: &erroringLLM{err: errors.New("network down")}}
	input := []Message{{Role: RoleUser, Content: "hi"}}

	// when
	history, err := agent.Run(context.Background(), input)

	// then
	if err == nil {
		t.Fatal("expected error")
	}
	if len(history) != 1 || history[0].Content != "hi" {
		t.Errorf("partial history: got %+v, want input preserved", history)
	}
}
