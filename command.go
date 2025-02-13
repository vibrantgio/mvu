package mvu

import (
	"fmt"

	"github.com/reactivego/x"
)

const should_trace = true

// Command

type Command struct{ x.Observable[any] }

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

func DoNothing() Command {
	return Command{x.Empty[any]()}
}

func Concurrent(cmds ...Command) Command {
	return Command{x.MergeMap(x.From(cmds...), func(cmd Command) x.Observable[any] {
		return cmd.Observable
	})}
}

func Sequence(cmds ...Command) Command {
	return Command{x.ConcatMap(x.From(cmds...), func(cmd Command) x.Observable[any] {
		return cmd.Observable
	})}
}
