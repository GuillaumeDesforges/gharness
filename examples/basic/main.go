// Basic end-to-end smoke test against a real Anthropic API key.
//
// Usage:
//
//	ANTHROPIC_API_KEY=sk-ant-... go run ./examples/basic
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/GuillaumeDesforges/gharness/core"
)

func main() {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY is not set")
		os.Exit(1)
	}

	llm := &core.Anthropic{
		APIKey: key,
		Model:  "claude-opus-4-7",
	}

	reply, err := llm.Complete(context.Background(), []core.Message{
		{Role: core.RoleUser, Content: "In one short sentence, what is gharness?"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(reply)
}
