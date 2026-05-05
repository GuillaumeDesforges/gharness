# Roadmap

Staged plan. Each stage must leave the previous one untouched and
independently usable. We do not start a stage until the previous one is
solid enough that we'd be comfortable shipping it on its own.

## Stage 1 — Core domain

The smallest useful agent harness.

**In scope**
- Message types (user, assistant, tool call, tool result).
- Tool interface (name, schema, execution).
- Provider interface — a single method that takes a conversation and tools
  and returns the next assistant turn. Models are addressed as
  `"provider/model"` strings.
- Agent loop — orchestrates provider calls and tool execution until the
  model stops requesting tools.
- Tests covering the loop's behavior with a fake provider.

**Out of scope (for this stage)**
- Streaming, retries, cost accounting, persistence, concurrency primitives,
  any real provider implementation beyond what's needed for tests.

**Done when** — the core can be embedded in a Go program, given a fake
provider and a couple of tools, and it runs a turn-based loop end-to-end
under test.

## Stage 2 — Skills

Composable bundles of capability (prompt fragments + tools + config) that
can be attached to an agent without modifying the core.

**Done when** — a skill can be defined in user code, registered with an
agent, and contributes both to the system prompt and the tool set without
the core knowing what a "skill" is.

## Stage 3 — MCP

Model Context Protocol support as an additional tool source. MCP servers
plug in through the existing tool interface; the core does not learn new
concepts.

**Done when** — an MCP server's tools are exposed to the agent loop and
indistinguishable from native tools at the core layer.

## Stage 4 — Memory

Persistent recall across runs. Likely a tool + storage interface; details
deferred until stages 1–3 inform the design.

**Done when** — the agent can store and retrieve information across
process boundaries through a swappable backend.

## Non-goals

- Locking ourselves into one provider's SDK.
- Building a CLI or UI in this repo. The harness is a library.
- Solving evaluation, tracing, or deployment. Those live above the SDK.
