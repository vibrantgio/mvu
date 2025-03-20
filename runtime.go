package mvu

import (
	"fmt"

	"gioui.org/app"
	"gioui.org/layout"

	"github.com/reactivego/x"
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

func (r *Runtime[Model]) Run(layers ...x.Observable[layout.Widget]) error {
	initialModel, initialCommand := r.Init()
	messages := make(chan Message, 1)
	type State struct {
		Model   Model
		Command Command
	}
	updater := x.Scan(x.Merge(r.Window.Messages(), x.Recv(messages)), State{Model: initialModel}, func(state State, message Message) State {
		model, command := r.Update(state.Model, message)
		return State{Model: model, Command: command}
	}).Publish().AutoConnect(2)
	models := x.Map(updater, func(state State) Model { return state.Model }).StartWith(initialModel)
	commands := x.Map(updater, func(state State) Command { return state.Command }).StartWith(initialCommand)
	runner := x.MergeMap(commands, func(cmd Command) x.Observable[any] {
		return cmd.Pipe(
			x.Send(messages),
			x.CatchError(func(err error, caught x.Observable[any]) x.Observable[any] {
				fmt.Println("Command Error:", err)
				return x.Empty[any]()
			}),
		)
	}).Go()
	defer func() { runner.Unsubscribe(); runner.Wait() }()
	return r.Window.Render(append(layers, x.Map(models, r.View))...).Wait()
}
