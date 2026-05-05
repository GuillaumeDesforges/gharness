package core

import (
	"context"
	"fmt"
)

// Agent runs a conversation forward by calling an LLM, executing any tool
// calls it requests, and feeding results back until some implementation-
// defined stop condition is met. Run takes the conversation so far and
// returns the conversation including everything the agent appended.
type Agent interface {
	Run(ctx context.Context, history []Message) ([]Message, error)
}

// FixedPointAgent loops Complete-and-execute until the assistant returns a
// turn with no tool calls. MaxSteps caps the number of LLM calls; zero
// means no cap. Tool execution errors and unknown-tool calls are not
// bubbled — they become RoleTool messages so the model can recover.
type FixedPointAgent struct {
	LLM      LLM
	Tools    []Tool
	MaxSteps int
}

var _ Agent = (*FixedPointAgent)(nil)

func (a *FixedPointAgent) Run(ctx context.Context, history []Message) ([]Message, error) {
	tools := indexTools(a.Tools)
	for step := 0; a.MaxSteps == 0 || step < a.MaxSteps; step++ {
		reply, err := a.LLM.Complete(ctx, history, a.Tools)
		if err != nil {
			return history, err
		}
		history = append(history, reply)
		if len(reply.ToolCalls) == 0 {
			return history, nil
		}
		for _, call := range reply.ToolCalls {
			history = append(history, executeToolCall(ctx, tools, call))
		}
	}
	return history, nil
}

func indexTools(tools []Tool) map[string]Tool {
	m := make(map[string]Tool, len(tools))
	for _, t := range tools {
		m[t.Name] = t
	}
	return m
}

func executeToolCall(ctx context.Context, tools map[string]Tool, call ToolCall) Message {
	tool, ok := tools[call.Name]
	if !ok {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("unknown tool: %s", call.Name),
		}
	}
	out, err := tool.Execute(ctx, call.Input)
	if err != nil {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    err.Error(),
		}
	}
	return Message{
		Role:       RoleTool,
		ToolCallID: call.ID,
		Content:    out,
	}
}
