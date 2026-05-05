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

## Auto-documentation

Following Karpathy's philosophy
(https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f), this
project documents itself as it grows:

- `docs/logs/` — append-only log of decisions, knowledge gained, and dead
  ends. One file per day, named `YYYY-MM-DD.md`; multiple entries within a
  day are `##` sections. Write the *why*, not just the *what*. Past days
  are never edited; corrections go in a new day's entry.
- `docs/wiki/` — distilled, current-state knowledge. Topic-oriented pages
  (`design.md`, `providers.md`, `testing.md`, ...). Rewrite freely as
  understanding improves.

When you make a non-trivial decision, learn something non-obvious, or change
direction, add a log entry. When the wiki is wrong or out of date, fix it.

Start any session by reading [`docs/wiki/index.md`](docs/wiki/index.md) for
current-state knowledge, then skim recent entries in `docs/logs/` for
context on in-flight work.

## Working agreements

- Edit existing files before creating new ones.
- No comments that restate the code. Comments explain *why* something
  surprising is the way it is.
- Tests live next to the code (`foo.go` ↔ `foo_test.go`) and exercise the
  public interface, not internals.
- The core depends only on the standard library until a real need forces
  otherwise. Document any added dependency in `docs/logs/`.
