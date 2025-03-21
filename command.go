package mvu

import (
	"fmt"

	"github.com/reactivego/rx"
)

const should_trace = true

// Command
type Command struct{ rx.Observable[Message] }

func (cmd Command) Trace(name string) Command {
	if should_trace {
		fmt.Println("Executing:", name)
		return Command{cmd.Finally(func(err error) {
			if err == nil {
				fmt.Println("Completed:", name)
			} else {
				fmt.Println("Failed:", name, err)
			}
		})}
	}
	return cmd
}

func Do(command func() (Message, error)) Command {
	runner := rx.Create[Message](func(index int) (Next Message, Err error, Done bool) {
		if index == 0 {
			msg, err := command()
			if err != nil {
				return nil, err, true
			}
			if msg != nil {
				return msg, nil, false
			}
		}
		return nil, nil, true
	})
	return Command{runner}
}

func DoNothing() Command {
	nothing := func() (Message, error) { return nil, nil }
	return Do(nothing)
}

func DoConcurrent(cmds ...Command) Command {
	return Command{rx.MergeMap(rx.From(cmds...), func(cmd Command) rx.Observable[any] {
		return cmd.Observable
	})}
}

func DoSequence(cmds ...Command) Command {
	return Command{rx.ConcatMap(rx.From(cmds...), func(cmd Command) rx.Observable[any] {
		return cmd.Observable
	})}
}
