// Package mvu is the Model-View-Update runtime for a Gio application: a Gio
// window, an Elm-shaped reducer over it, and rx observables as the wiring
// between the two. It imports nothing else in the organization.
//
// An application writes four things: a Model type, message types, an Init
// returning the seed model and a startup command, and an Update reducing a
// message onto a model. [Run] is the whole application for a single window;
// [Loop] is the reducer alone, for applications that own their rendering.
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
// Component code hands a message to the loop with [MessageOp]:
//
//	mvu.MessageOp{Message: SelectItem{ID: id}}.Add(gtx.Ops)
//
// The collector is keyed on the exact *op.Ops the current frame is being
// recorded into, and an Add against any other buffer is dropped silently — no
// panic, no error, just a message that never arrives. A component whose body is
// recorded into a private op.Ops, as a caching layer does, therefore cannot
// emit: emit from the component that owns gtx.Ops, never from inside a cached
// recording.
//
// # Platform handles arrive as view events
//
// A callback that fires outside any frame — an OS drag callback, a native
// notification — has no gtx.Ops, so MessageOp cannot carry its message; the
// only correct path is a channel of its own, wrapped in rx.Recv and merged
// into [Loop]'s messages. The handles such a callback needs come from the
// window: [Window.ViewEvents] forwards Gio's [app.ViewEvent] values (the
// native view and layer on macOS, and so on), the one event class
// [Window.Render] forwards beyond DestroyEvent and FrameEvent. The first view
// event arrives before the first frame and is buffered until subscribed, so a
// subscriber attaching in ordinary application order never misses it; the
// full delivery contract — single subscription, buffer of four, keep-latest
// on overflow, completion on destroy — is on the method. Applications without
// a platform adapter never call it.
//
// # The models observable carries the current model
//
// [Loop] returns the models observable and the command runner. Models is a
// replay-latest multicast: every subscriber observes the model in force the
// moment it attaches, so a layer topology needs no consumer count and no
// Publish().AutoConnect(N) of its own. Subscribe it as many times as the
// topology has consumers, at whatever time they attach.
//
// Late subscription is not a corner case. A consumer handed an observable of
// observables flattens it, and so re-subscribes everything it combines the
// inner one with every time the outer one emits. A window that feeds its
// model into such a consumer therefore re-subscribes the model mid-flight,
// and a stream without replay would leave that consumer with no model at all
// until the next message.
//
// Models conflates: a consumer that falls behind converges on the newest
// model and never blocks the loop. A fact whose every occurrence is
// load-bearing belongs in a message, not in the model stream.
//
// An application that still wraps models in Publish().AutoConnect(N) keeps
// that gate's arithmetic: N must equal the number of cold subscriptions the
// topology makes, too high never connects at all — the window's messages are
// never drained, and because that channel holds exactly one MessageOp the
// event goroutine blocks on the second one it tries to hand over and the
// window stops painting — and neither failure logs anything. The gate buys
// nothing the models observable does not already give, so a topology that
// gains a consumer is better off dropping it than re-counting.
//
// # The window owns its Option boundary
//
// Applying window options after construction makes Gio rebuild the native
// window's configuration, which can silently undo any adjustment made
// directly to the native handle. [Window.Option] therefore forwards to the
// underlying window and then notifies every func registered with
// [Window.OnConfigure]: once after the first frame — covering construction
// options and Gio's own initial configuration — and again after every later
// Option call. A platform adapter that pokes the native window registers a handler
// there and re-asserts its adjustment on each notification. The raw handle
// from [Window.Window] stays available, but options applied through it
// bypass the notification.
//
// # The window can remember its frame
//
// [RememberFrame] is one opt-in call at window construction, given the
// application's name, after which the window reopens where and how the user
// last left it. The frame — the size and position together — is kept as JSON
// in the OS config directory under that name, written after a change stands
// still rather than per frame, and nothing is written until the first change
// after launch. A missing or damaged file, or a frame that no longer lands on
// any screen, leaves the options passed to [NewWindow] standing.
//
// Position is remembered on macOS, where it is read from and applied to the
// native window. Every other platform remembers the size alone: neither Gio's
// [app.Config] nor any window option carries a position there, so there is
// nothing to read or to apply, and a position written on macOS is loaded and
// ignored.
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
// Runnable programs, from a bare window upwards, live in the example module,
// github.com/vibrantgio/mvu/example.
package mvu
