package mvu

import (
	"fmt"

	"gioui.org/app"
	"gioui.org/layout"

	"github.com/reactivego/rx"
)

type Runtime[Model any] struct {
	Window *Window
	Init   func() (Model, Command)
	Update func(model Model, message Message) (Model, Command)
	View   func(model Model) layout.Widget
}

func NewRuntime[Model any](options ...app.Option) *Runtime[Model] {
	return &Runtime[Model]{Window: NewWindow(options...)}
}

func (r *Runtime[Model]) Run(layers ...rx.Observable[layout.Widget]) error {
	initialModel, initialCommand := r.Init()
	messages := make(chan Message, 1)
	type State struct {
		Model   Model
		Command Command
	}
	updater := rx.Scan(rx.Merge(r.Window.Messages(), rx.Recv(messages)), State{Model: initialModel}, func(state State, message Message) State {
		model, command := r.Update(state.Model, message)
		return State{Model: model, Command: command}
	}).Publish().AutoConnect(2)
	models := rx.Map(updater, func(state State) Model { return state.Model }).StartWith(initialModel)
	commands := rx.Map(updater, func(state State) Command { return state.Command }).StartWith(initialCommand)
	runner := rx.MergeMap(commands, func(cmd Command) rx.Observable[any] {
		return cmd.Pipe(
			rx.Send(messages),
			rx.CatchError(func(err error, caught rx.Observable[any]) rx.Observable[any] {
				fmt.Println("Command Error:", err)
				return rx.Empty[any]()
			}),
		)
	}).Go()
	defer func() { runner.Unsubscribe(); runner.Wait() }()
	return r.Window.Render(append(layers, rx.Map(models, r.View))...).Wait()
}
