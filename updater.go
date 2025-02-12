package mvu

import "github.com/reactivego/x"

type Updater[Model any] interface {
	Update(update UpdateFunc[Model], messages ...x.Observable[any]) (models x.Observable[Model], commands x.Observable[Command])
}

func NewUpdater[Model any](initialModel Model, initialCommand Command) Updater[Model] {
	return &updater[Model]{initialModel: initialModel, initialCommand: initialCommand}
}

type updater[Model any] struct {
	initialModel   Model
	initialCommand Command
}

type UpdateFunc[Model any] func(model Model, message any) (Model, Command)

func (u *updater[Model]) Update(update UpdateFunc[Model], messages ...x.Observable[any]) (models x.Observable[Model], commands x.Observable[Command]) {
	type State struct {
		Model   Model
		Command Command
	}
	updater := x.Scan(x.Merge(messages...), State{Model: u.initialModel}, func(state State, message any) State {
		model, command := update(state.Model, message)
		return State{Model: model, Command: command}
	}).Publish().AutoConnect(2)
	models = x.Map(updater, func(state State) Model { return state.Model }).StartWith(u.initialModel)
	commands = x.Map(updater, func(state State) Command { return state.Command }).StartWith(u.initialCommand)
	return
}
