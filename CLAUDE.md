# gharness

A Go SDK for building AI agent harnesses.

## Ambition

A composable, lean core that other Go programs can embed to run agentic loops.
Provider-agnostic: models are addressed as `"provider/model"` strings (e.g.
`"anthropic/claude-opus-4-7"`, `"openai/gpt-5"`). The harness does not bind to
any single SDK.

## Principles

- **Composable** — small interfaces, orthogonal pieces, no hidden coupling.
  Every layer must be usable on its own and replaceable from the outside.
- **Lean** — no feature unless something concrete needs it. Three similar
  lines beat a premature abstraction. Delete aggressively.

## Roadmap

Core domain → skills → MCP → memory. Each stage must leave the previous
one untouched and independently usable. Details and stage-by-stage scope
live in [`docs/wiki/roadmap.md`](docs/wiki/roadmap.md).

## Knowledge layer (Karpathy LLM Wiki)

This project follows Andrej Karpathy's LLM Wiki pattern
(https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f),
adapted for an engineering codebase. Three layers:

- **Schema** — this file (`CLAUDE.md`). How knowledge is structured and
  what workflows to follow.
- **Wiki** (`docs/wiki/`) — **specification, design decisions, and
  conceptual knowledge.** Topic-oriented pages, rewritten freely. **Not**
  implementation reference (header names, JSON paths, restated signatures,
  field tables) — the code is here for that. Catalog:
  [`docs/wiki/index.md`](docs/wiki/index.md).
- **Raw** — `docs/logs/` plus git history. Decisions live in the daily
  log (one file per day, `YYYY-MM-DD.md`, append-only `##` sections);
  implementation diffs live in git. Both are immutable; the wiki is
  derived from them.

### Operations

- **Ingest** — when something non-trivial happens (decision, design,
  implementation, dead end), append a `##` section to today's log
  **and** update the wiki pages it touches. Log = *what happened and
  why*; wiki = *what is true now*.
- **Query** — start any session by reading `docs/wiki/index.md`, then
  skim recent `docs/logs/` entries for in-flight context.
- **Lint** — periodically check the wiki for stale claims, dead links,
  and orphan pages.

### Commit workflow

**Update the wiki immediately before committing.** Code that changes
contract, behavior, or convention must arrive with the wiki updated in
the same commit. The wiki must reflect committed state — no perpetual
lag.

## Working agreements

- Edit existing files before creating new ones.
- No comments that restate the code. Comments explain *why* something
  surprising is the way it is.
- Tests live next to the code (`foo.go` ↔ `foo_test.go`) and exercise the
  public interface, not internals.
- The core depends only on the standard library until a real need forces
  otherwise. Document any added dependency in `docs/logs/`.
