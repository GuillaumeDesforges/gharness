# Testing conventions

## Scope

We test where bugs realistically hide:

- Translations between our types and a vendor's wire shape (role
  renames, default-value insertion, content extraction).
- Error paths a caller is going to depend on.

We do **not** test that the standard library works, or that one-line
defaults fall back. Those are stdlib's job, or covered transitively by
real tests.

## Layout

Tests live next to the code (`foo.go` ↔ `foo_test.go`) and exercise the
public interface. If a test needs to reach into private types, that's a
sign the public surface is missing something.

## Given / when / then

Each test is structured as three labeled phases, separated by blank
lines:

- `// given` — set up dependencies, inputs, fakes.
- `// when` — invoke the subject. Exactly the line(s) under test.
- `// then` — assert. Errors from the subject's call are checked here,
  not floating between phases.

The discipline is the point: it forces an honest answer to "what is
this test actually checking?".
