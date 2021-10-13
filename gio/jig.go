//go:build ignore
// +build ignore

package gio

import (
	"gioui.org/op"
	_ "github.com/reactivego/rx/generic"
)

//jig:type Event = event.Event

func generate_rx() {
	_ = GoroutineScheduler()
	_ = MakeTrampolineScheduler()

	// ObservableFrameEvent
	{
		var o ObservableFrameEvent
		o.Subscribe(nil)
		PrintlnFrameEvent()
	}

	// ObservableCallOp
	{
		FromCallOp()
		JustCallOp(op.CallOp{})
		CombineLatestCallOp()
		DeferCallOp(nil)
		var o ObservableCallOp
		o.CombineLatestWith()
	}

	// ObservableCallOpSlice
	{
		var o ObservableCallOpSlice
		o.MapCallOp(nil)
	}
}
