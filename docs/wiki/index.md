# Wiki catalog

The wiki captures **specification, design decisions, and conceptual
knowledge** about gharness — *not* implementation reference. The code
is the source of truth for how things are built; this wiki is the
source of truth for what they must do and why they look the way they
do. Pages are rewritten freely as understanding improves.

For the history of *how* we got here, read `docs/logs/`.

## Pages

### Plan
- [Roadmap](roadmap.md) — staged plan from core domain to extensions.

### Design
- [LLM contract](llm.md) — what a language model is in this project,
  what the interface promises, and what's deferred.
- [Providers](providers.md) — what a provider must do, and the shape
  decisions that govern all three.

### Conventions
- [Testing](testing.md) — what we test, what we don't, and the
  given/when/then discipline.

## Conventions for this catalog

- Every wiki page has one entry here, ≤120 chars.
- Update same-commit when you add, rename, or retire a page.
- Categories grow organically; split or rename when shape changes.
