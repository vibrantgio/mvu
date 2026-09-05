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
// subscribe the model observable from a per-row factory, which attaches after
// the seed has fired — and let a test count the subscriptions instead of
// tuning the number by hand.
//
// What enters the count is subscriptions to models, nothing else. The
// window's channel-backed streams — [Window.Messages] and [Window.ViewEvents]
// — are outside the multicast: messages is an input Loop subscribes exactly
// once, and view events flow to a platform adapter without touching the loop
// at all. Subscribing ViewEvents, or merging another rx.Recv-backed message
// source into Loop's input, changes N by exactly zero; only a new subscriber
// of the models observable moves it.
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
