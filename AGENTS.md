# AGENTS.md — mvu

The Model-View-Update runtime every Vibrant Gio application is built on: a
Gio window driven by an MVU loop whose views are
`rx.Observable[layout.Widget]` layers — `NewWindow`, `Loop` and `Run`,
`Message` and `Command`, and `MessageOp` for sending a message from inside
a layout.

**Layer.** Tier 0 of ADR-001's stack, `mvu → spectrum → prism → pulse →
cadence → markdown`, and its base: it imports nothing else in the
organization, only `gioui.org` and `github.com/reactivego/rx`. prism,
spectrum and pulse require it directly; cadence only indirectly, through
them.

**Read the canonical guide before you write code against this module.** It is
the organization's one agent guide — the module inventory with current tags,
the application skeleton, the MVU loop and rx semantics, typography, and the
pitfalls that are not guessable. It lives exactly once, in `vibrantgio/.github`,
and this file links it rather than copying it:

    https://raw.githubusercontent.com/vibrantgio/.github/master/llms.txt

**Modules.** `github.com/vibrantgio/mvu` at the repository root, and one
nested module: `example/` (`github.com/vibrantgio/mvu/example`).
Nested-module tags carry the directory as a prefix — `example/v0.4.4`, not
`v0.4.4`.

**Build and test.** From the repository root, and again inside each nested
module directory — `./...` does not cross a module boundary:

    go build ./... && go test ./...
