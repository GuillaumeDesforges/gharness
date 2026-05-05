# Providers

A *provider* is a concrete `LLM` that translates the project's abstract
conversation contract into one vendor's API. Three exist today: OpenAI,
Anthropic, Gemini.

## Specification

A provider must:
- Satisfy the [LLM contract](llm.md).
- Translate our `Role`, `Message`, and `Tool` shape to the vendor's
  wire format and the response back, hiding wire-format quirks entirely
  behind `Complete`. Callers never need to know what the vendor renames
  our roles to, how it nests content, whether it carries call IDs
  natively, or whether tool arguments travel as objects or strings.
- Surface vendor errors as plain `error` values prefixed with the
  provider name. A caller can route on the prefix without binding to a
  custom error type.

## Design decisions

- **Stdlib HTTP only.** No vendor SDKs. Consumers pick up zero
  transitive dependencies; we pay the cost of simple HTTP+JSON, which
  is a cost we understand.
- **Same shape across providers, duplicated not abstracted.** Three is
  too few for a base type to earn its keep. Reopened if providers grow
  or their configuration drifts in lockstep.
- **Vendor quirks live inside the provider, not the contract.** Renamed
  roles, mandatory fields, content-block arrays, batch-tool-result
  requirements, ID-less tool calls, JSON-string-encoded arguments — all
  absorbed by `Complete`. Cost: a few lines of translation per
  provider. Value: the rest of the system never learns vendor names.
- **Required-but-vendor-specific knobs default on zero.** When a vendor
  requires a field our abstract contract doesn't (e.g. a token cap),
  the provider exposes a typed field with a sensible default, so the
  zero-value struct stays usable. The quirk does not bleed into the
  cross-provider mental model.
- **Synthesize what the vendor doesn't carry.** Some vendors don't put
  call IDs on the wire; the provider invents them so the agent's
  call↔result matching contract holds across all backends.
- **Compile-time interface assertion per provider.** A
  `var _ LLM = (*X)(nil)` line catches silent contract drift; one of
  the cheapest guards we have.
