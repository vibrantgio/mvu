package mvu

import (
	"fmt"

	"github.com/reactivego/x"
)

type Runner interface {
	Messages() x.Observable[any]
	Run(commands x.Observable[Command]) x.Subscription
}

func NewRunner() Runner {
	return &run{messages: make(chan any, 1)}
}

type run struct {
	messages chan any
}

func (r *run) Messages() x.Observable[any] {
	return x.Recv(r.messages)
}

func (r *run) Run(commands x.Observable[Command]) x.Subscription {
	return x.MergeMap(commands, func(cmd Command) x.Observable[any] {
		return cmd.Pipe(
			x.Send(r.messages),
			x.CatchError(func(err error, caught x.Observable[any]) x.Observable[any] {
				fmt.Println("Command Error:", err)
				return x.Empty[any]()
			}),
		)
	}).Go()
}
