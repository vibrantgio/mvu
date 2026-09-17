package mvu

import (
	"fmt"

	"gioui.org/layout"

	"github.com/reactivego/rx"
)

// Loop is the MVU message/command loop, independent of any window: init
// produces the seed model and initial command, and update is scanned over
// messages merged with the messages emitted by the commands update returns,
// so long-running commands (streams, sequences) feed back into the loop.
// init is called once, when Loop is; its command runs as soon as the runner
// starts.
//
// The returned models observable is a replay-latest multicast: it carries the
// current model — the seed until the first message, the newest model after
// that — and hands it to every subscriber the moment it attaches, however late
// and however many. That is what a consumer subscribing outside application
// start-up needs: a consumer that flattens an observable of observables
// re-subscribes whatever it combines the inner one with every time the outer
// one emits, and a stream without replay leaves such a re-subscriber with no
// value until the next message. No consumer count is involved.
//
// It conflates: a consumer that falls behind converges on the newest model
// rather than receiving every intermediate one, and never blocks the loop. A
// model is state, so the newest one is the whole answer; a fact whose every
// occurrence is load-bearing belongs in a message, not in the model stream.
//
// The scan connects — and messages start draining — when the first consumer
// subscribes, and disconnects when the last one leaves.
//
// The returned runner executes commands until unsubscribed. A command error
// is reported and terminates that command only, never the loop. Callers stop
// the loop with:
//
//	defer func() { runner.Unsubscribe(); runner.Wait() }()
func Loop[Model any](
	messages rx.Observable[Message],
	init func() (Model, Command),
	update func(Model, Message) (Model, Command),
) (models rx.Observable[Model], runner rx.Subscription) {
	seed, initial := init()
	feedback := make(chan Message, 1)
	type state struct {
		model   Model
		command Command
	}
	updater := rx.Scan(rx.Merge(messages, rx.Recv(feedback)), state{model: seed},
		func(s state, message Message) state {
			model, command := update(s.model, message)
			return state{model: model, command: command}
		}).Publish().AutoConnect(2)
	models = rx.Map(updater, func(s state) Model { return s.model }).Behavior(seed).RefCount()
	commands := rx.Map(updater, func(s state) Command { return s.command }).StartWith(initial)
	runner = rx.MergeMap(commands, func(cmd Command) rx.Observable[any] {
		return cmd.Pipe(
			rx.Send(feedback),
			rx.CatchError(func(err error, caught rx.Observable[any]) rx.Observable[any] {
				fmt.Println("Command Error:", err)
				return rx.Empty[any]()
			}),
		)
	}).Go()
	return models, runner
}

// Run drives a window with the MVU loop: models scanned by Loop are mapped
// through view onto a layer stacked in front of layers, and Run blocks until
// the window is destroyed.
//
// Run renders on the raw mvu Window and view receives only the Model. An
// application whose layers need more than the Model composes [Loop] with its
// own rendering instead.
func Run[Model any](
	w *Window,
	init func() (Model, Command),
	update func(Model, Message) (Model, Command),
	view func(Model) layout.Widget,
	layers ...rx.Observable[layout.Widget],
) error {
	models, runner := Loop(w.Messages(), init, update)
	defer func() { runner.Unsubscribe(); runner.Wait() }()
	return w.Render(append(layers, rx.Map(models, view))...).Wait()
}
