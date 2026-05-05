# LLM contract

A language model in this project is a function over conversation: given
the dialogue so far, return the assistant's next reply. The `LLM`
interface in `core/llm.go` is the whole spec.

## Specification

- Takes the **entire conversation** as input on every call. No session
  state lives behind the interface.
- Returns the assistant's next utterance as text. One reply per call;
  not a stream, not a structured turn.
- Conversation roles are `user` and `assistant`. No `system` role yet
  (see *Deferred*).

## Design decisions

- **Pure function of context.** Statefulness lives above the interface,
  never on it. Sessions, memory, caching stay composable.
- **Reply is text, not a structured `Message`.** The reply is always
  assistant-role; making the role implicit keeps the asymmetry honest.
  The asymmetry is a signal that the LLM is a primitive, not a peer of
  the conversation type.
- **Model bound at construction.** A given `LLM` instance is a handle
  for one model. The ambition's `"provider/model"` string addresses
  *which handle to build*, not *what to ask each call*.

## Deferred

- **System prompt.** Configuration-shaped, not turn-shaped; defer.
- **Tools, multi-block content.** Stage 2+ of the roadmap.
- **Streaming.** Single-shot is the primitive; streaming layered above
  only when a real consumer needs it.
