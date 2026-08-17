# AGENTS.md — mvu

The Model-View-Update runtime every Vibrant Gio application is built on: a
Gio window driven by an MVU loop whose views are
`rx.Observable[layout.Widget]` layers — `NewWindow`, `Loop` and `Run`,
`Message` and `Command`, `MessageOp` for sending a message from inside a
layout, and `stream.Value`, the organization's one sanctioned observable
for state that several consumers watch.

**Layer.** Tier 0 of ADR-001's stack, `mvu → theme → components → effects →
patterns → markdown`, and its base. Outside the organization it needs only
`gioui.org` and `github.com/reactivego/rx`. Its root module imports nothing
else in the organization. Its nested `mvu/example` module adds `backdrop`,
`circle`, `font`, `gradient`, `ivg`, `ivg/raster/gio`, `textdraw` and
`theme` — those edges are the nested module's and not the root's. Imported
by `components`, `effects`, `patterns` and `theme`. Outside the tier table,
also by the demo module `components/gallery` and all eight workbench
applications. Both directions are measured rather than typed —
`scripts/check-layers.sh --edges` reports the graph and
`scripts/sync-agents.sh` renders these sentences from it — so correcting
them here changes nothing.

**Read the canonical guide before you write code against this module.** It is
the organization's one agent guide — the module inventory with current tags,
the application skeleton, the MVU loop and rx semantics, typography, and the
pitfalls that are not guessable. It lives exactly once, in `vibrantgio/.github`,
and this file links it rather than copying it:

    https://raw.githubusercontent.com/vibrantgio/.github/master/llms.txt

**Modules.** `github.com/vibrantgio/mvu` at the repository root, and two
nested modules: `desktop/` (`github.com/vibrantgio/mvu/desktop`),
`example/` (`github.com/vibrantgio/mvu/example`). Nested-module tags carry
the directory as a prefix — `desktop/v0.9.0`, not `v0.9.0`.

**Build and test.** From the repository root, and again inside each nested
module directory — `./...` does not cross a module boundary:

    go build ./... && go test ./...
