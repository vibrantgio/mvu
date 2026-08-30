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
// The returned models observable emits the seed model first (StartWith) and
// then one model per message. It is never replayed: the scan behind it is
// already multicast, so a second direct subscriber does not re-run update, but
// it is handed the seed rather than the model current at the time it attached.
// A single consumer can subscribe it directly; a layer topology with N
// consumers applies Publish().AutoConnect(N), which holds the connect back
// until all N have attached so the seed reaches every one of them. The scan
// connects — and messages start draining — when the models side is subscribed.
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
	models = rx.Map(updater, func(s state) Model { return s.model }).StartWith(seed)
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
