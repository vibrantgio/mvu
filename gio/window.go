package gio

import (
	"log"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/op"

	"github.com/reactivego/scheduler"
)

const kLogEvents = false

// Window handles the events of a single gioui app window.
type Window struct {
	*app.Window
	Handlers
}

type Option = app.Option

func NewWindow(options ...app.Option) *Window {
	return &Window{Window: app.NewWindow(options...)}
}

func (w *Window) FrameEvents() ObservableFrameEvent {
	observable := func(observe FrameEventObserver, scheduler Scheduler, subscriber Subscriber) {
		w.Handlers.FrameEvents()(observe, scheduler, subscriber)
		w.Invalidate()
	}
	return observable
}

type CallOp = op.CallOp

func (w *Window) Frame(loading CallOp, layers ...ObservableCallOp) Subscription {
	launchScreen := CallOpSlice{loading}
	invalidate := func(cos CallOpSlice) interface{} {
		w.Invalidate()
		return cos
	}
	content := CombineLatestCallOp(layers...).StartWith(launchScreen).Map(invalidate).SubscribeOn(scheduler.Goroutine)

	frames := FromChanEvent(w.Events()).AsObservable()

	var ops op.Ops
	subcription := frames.WithLatestFrom(content).Subscribe(func(next Slice, err error, done bool) {
		ops.Reset()
		switch {
		case !done:
			if kLogEvents {
				log.Printf("event: %T %v\n", next[0], next[0])
			}
			switch ev := next[0].(type) {
			case system.FrameEvent:
				w.Handlers.Dispatch(&ops, ev)
				for _, c := range next[1].(CallOpSlice) {
					c.Add(&ops)
				}
				ev.Frame(&ops)
				// log.Printf("concurrency: %v\n", scheduler.Goroutine)
			case system.DestroyEvent:
				if ev.Err != nil {
					log.Fatal(err)
				}
			default:
				// log.Printf("event: %T %v\n", ev, ev)
			}
		case err != nil:
			log.Printf("error: %v\n", err)
		default:
			log.Printf("complete: %v\n", scheduler.Goroutine)
		}
	})
	return subcription
}
