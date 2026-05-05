# LLM contract

A language model in this project is a function over conversation and
tools: given the dialogue so far and the capabilities the model is
allowed to call, return the assistant's next turn. The `LLM` interface
in `core/llm.go` is the whole spec.

## Specification

- Takes the **entire conversation** plus a per-call list of tools. No
  session state lives behind the interface.
- Returns one assistant turn per call. The turn is a `Message`: it may
  carry text, tool calls, or both, and is always assistant-role.
- Conversation roles are `user`, `assistant`, and `tool`. A `tool` turn
  is a result returned to the model and references the call it answers
  by ID. No `system` role yet (see *Deferred*).

## Design decisions

- **Pure function of context.** Statefulness lives above the interface,
  never on it. Sessions, memory, caching stay composable.
- **Reply is a `Message`, not text.** Once assistant turns can carry
  tool calls alongside text, the original "text in roles, plain text
  out" asymmetry stops being honest. Returning a `Message` makes
  appending the reply to history a one-step operation and keeps the
  shape symmetric with input.
- **Tools are per-call.** Skills (Stage 2) will attach and detach tools
  per agent turn; binding tools at provider construction would force
  Stage 2 to reshape the contract. Per-call now, no migration later.
- **`Tool` is a struct, not an interface.** Tool authors write inline
  values; a stateful tool closes over its state in `Execute`. The same
  "three is too few for a base type" logic that left the providers
  unabstracted applies. An interface can be lifted over this later
  without breaking callers if the polymorphism ever earns its keep.
- **Tool input as `json.RawMessage`, output as `string`.** Inputs
  round-trip the model's own JSON without re-encoding; outputs travel
  back to the model as text and stay text-shaped at the boundary. JSON
  outputs serialize themselves on the way out.
- **Schema as `map[string]any`.** Stdlib-only and wire-shaped; matches
  what providers want to send and what a future reflective wrapper
  produces. A typed schema library would break the stdlib-only rule.
- **Call IDs are opaque.** Provider-issued and gharness-internal: the
  agent uses them to match a result back to the call that produced it.
  Vendors that don't carry IDs natively (Gemini) get synthesized ones
  inside the provider.
- **Model bound at construction.** A given `LLM` instance is a handle
  for one model. The ambition's `"provider/model"` string addresses
  *which handle to build*, not *what to ask each call*.

## Deferred

- **System prompt.** Configuration-shaped, not turn-shaped; defer.
- **Streaming.** Single-shot is the primitive; streaming layered above
  only when a real consumer needs it.
- **Generic tool helper** (`NewTool[In, Out]`). The erased shape was
  designed to wrap cleanly, but reflective schema generation is
  non-trivial code with no concrete need yet.
