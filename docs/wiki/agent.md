# Agent

An *agent* drives a conversation forward by repeatedly calling an
[`LLM`](llm.md) and executing the tools it asks for. The `Agent`
interface in `core/agent.go` is the contract; `FixedPointAgent` is the
basic implementation that ships with the core.

## Specification

- `Agent.Run(ctx, history) ([]Message, error)` consumes the conversation
  so far and returns the conversation including every turn the agent
  appended (assistant replies and tool results).
- The agent owns no conversation state. History is passed in and
  returned. Same rule the LLM contract follows: statefulness lives above
  the interface.
- On error, partial history is returned alongside. The transcript at the
  moment of failure is the most useful diagnostic the caller has.

## FixedPointAgent

Runs the conversation to a fixed point: loop `LLM.Complete` and execute
any tool calls it returns, stopping when the assistant returns a turn
with no tool calls. `MaxSteps` caps the number of LLM calls; zero means
no cap.

Tool failures (execution error or unknown tool name) do not abort the
run. The failure becomes a `RoleTool` message whose content is an error
string, and the loop continues so the model can self-correct.

## Design decisions

- **`Agent` is an interface; `Tool` and providers stayed concrete.** The
  agent is the top of the stack — wrapping it (tracing, budget caps,
  alternate orchestrations) is a natural use case, and the interface
  gives users a seam they can replace without forking the loop. The same
  argument doesn't apply to `Tool` (data + a function) or to providers
  (three is too few for a base type).
- **Fixed-point semantics.** "The assistant returned no tool calls" is
  how the model signals it's done. Any other stop rule (turn count,
  content match, sentinel tool) is a policy that belongs above the core,
  not inside it.
- **`MaxSteps` zero means infinite.** The cap is a safety net, not a
  required choice. Forcing every caller to pick a number for a problem
  they don't have yet would be ceremony. A consumer who wants a hard
  budget sets one; the loop respects it.
- **Tool errors become `RoleTool` messages, not Go errors.** Models
  hallucinate tool names and call tools with bad arguments. Feeding the
  failure back as a tool result lets the model recover within the same
  run; bubbling it up forces the caller to invent a recovery loop the
  agent already has.
- **No `Step` method.** A single-tick variant is speculative surface
  until a UI or test genuinely needs to drive iteration externally. Add
  it when something forces it.
- **No system prompt field.** Configuration-shaped, like at the LLM
  layer; defer until Stage 2 (skills) actually needs it.
- **Compile-time interface assertion.** `var _ Agent =
  (*FixedPointAgent)(nil)` — same cheap insurance the providers carry.

## Deferred

- Streaming and step events / observability hooks.
- Parallel tool dispatch when the model returns multiple calls in one
  turn.
- Sentinel error / structured stop reason on `MaxSteps` exhaustion. The
  caller can detect it by inspecting the last message's `ToolCalls`.
