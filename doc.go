// Package mvu is the Model-View-Update runtime at the root of the Vibrant Gio
// stack: a Gio window, an Elm-shaped reducer over it, and rx observables as
// the wiring between the two. It is tier 0 of ADR-001 and imports nothing else
// in the organization — spectrum wraps its [Window] to scope a theme, and
// prism, pulse, cadence and markdown all draw inside a layer it drives.
//
// You write four things: a Model type, message types, an Init returning the
// seed model and a startup command, and an Update reducing a message onto a
// model. [Run] is the whole application for a single window; [Loop] is the
// reducer alone, for applications that own their rendering — wrapping the
// window in a theme, as spectrum/window does, is the usual reason.
//
//	w := mvu.NewWindow(app.Title("Counter"))
//	if err := mvu.Run(w, Init, Update, View); err != nil { ... }
//
// Side effects are [Command] values — [Do], [DoNothing], [DoConcurrent],
// [DoSequence] — and the loop runs them, feeding the messages they emit back
// into Update. One command may stream many messages, so a long-running source
// is a single command, not a goroutine. A command that fails is reported on
// stdout and torn down alone: the loop keeps reducing and later messages still
// arrive. An application with no effects returns DoNothing() everywhere and
// never notices the runner.
//
// # Messages come out of a frame, not out of a callback
//
// Widget code hands a message to the loop with [MessageOp]:
//
//	mvu.MessageOp{Message: SelectItem{ID: id}}.Add(gtx.Ops)
//
// The collector is keyed on the exact *op.Ops the current frame is being
// recorded into, and an Add against any other buffer is dropped silently — no
// panic, no error, just a message that never arrives. That is a real trap, not
// a theoretical one: a widget drawn through prism/cache.FrameCache records into
// the cache's own private op.Ops, so a MessageOp added inside that body goes
// nowhere, and on a cache hit the body does not run at all. Emit from the
// widget that owns gtx.Ops, never from inside a cached recording.
//
// # AutoConnect counts are load-bearing, and both errors are silent
//
// [Loop] returns the models observable and the command runner. Models emits the
// seed first and never replays: a subscriber that attaches later is handed that
// same seed rather than the current model — with the model already advanced to
// 5, a freshly attached subscriber was measured receiving 0. A layer topology
// with N consumers therefore multicasts with models.Publish().AutoConnect(N),
// which holds the connect back until all N have attached so the seed reaches
// every one of them.
//
// N must equal the number of cold subscriptions the topology actually makes.
// Too low and the loop connects early, so the late consumers render a zero
// Model. Too high and it never connects at all: the window's messages are never
// drained, and because that channel holds exactly one MessageOp the event
// goroutine blocks on the second one it tries to hand over, and the window
// stops painting. Neither failure logs anything. Keep N static — never
// subscribe the model observable from inside a per-row prism/keyed factory,
// which attaches after the seed has fired — and let a test count the
// subscriptions instead of tuning the number by hand.
//
// # Threading
//
// [Window.Render] reads window events and calls Frame on one goroutine, because
// Gio's frame protocol deadlocks if a flush is delivered before Frame is called.
// Layer observables are subscribed concurrently and published to that goroutine
// as an atomic snapshot, which is then invalidated to schedule a frame: heavy
// work is free to run on rx goroutines, but nothing it produces is drawn until
// the next frame event arrives. Never call Gio from a goroutine of your own.
//
// Two more things a first program needs. app.Main() must be the last call on
// the main goroutine, with the real work started as a goroutine before it. And
// a hand-built loop is stopped by its runner, not by its window:
//
//	models, runner := mvu.Loop(w.Messages(), Init, Update)
//	defer func() { runner.Unsubscribe(); runner.Wait() }()
//
// The example module — github.com/vibrantgio/mvu/example, tagged in lockstep
// with this one, so example/v0.4.3 goes with v0.4.3 — holds runnable programs
// from a bare window upwards. The organization's agent guide carries the full
// application skeleton and the rules above:
// https://raw.githubusercontent.com/vibrantgio/.github/master/llms.txt
package mvu
