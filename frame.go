package vibrant

import (
	"image"
	"sync"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/system"
	"gioui.org/unit"

	"github.com/reactivego/x"
)

type Frame struct {
	app.Config
	unit.Metric
	system.Insets
}

func (c Frame) SafeRect() image.Rectangle {
	return image.Rect(
		c.Dp(c.Left),
		c.Dp(c.Top),
		c.Size.X-c.Dp(c.Right),
		c.Size.Y-c.Dp(c.Bottom))
}

func (c Frame) DistinctFrom(f system.FrameEvent) bool {
	return c.Size != f.Size || c.Metric != f.Metric || c.Insets != f.Insets
}

func (window *Window) Frame() x.Observable[Frame] {
	var shared struct {
		sync.Mutex
		Frame
		Count int
	}
	update := func(next event.Event) (Frame, int) {
		shared.Lock()
		defer shared.Unlock()
		if event, ok := next.(app.ConfigEvent); ok {
			shared.Config = event.Config
			shared.Metric = unit.Metric{}
		} else if event, ok := next.(system.FrameEvent); ok {
			if shared.DistinctFrom(event) {
				shared.Size = event.Size
				shared.Metric = event.Metric
				shared.Insets = event.Insets
				shared.Count++
			}
		}
		return shared.Frame, shared.Count
	}
	return func(observe x.Observer[Frame], scheduler x.Scheduler, subscriber x.Subscriber) {
		channel := make(chan any, 5)
		x.AsObservable[Frame](x.FromChan(channel))(observe, scheduler, subscriber)
		frame, mycount := update(nil)
		channel <- frame
		observer := func(next event.Event, err error, done bool) {
			switch {
			case !done:
				frame, count := update(next)
				if count > mycount {
					mycount = count
					if subscriber.Subscribed() {
						select {
						case channel <- frame:
							// OK
						default:
							panic("Frame: Channel Overflow")
						}
					}
				}
			case err != nil:
				select {
				case channel <- err:
					// OK
				default:
					panic("Frame: Channel Overflow")
				}
				close(channel)
			default:
				close(channel)
			}
		}
		handler := &EventHandler{observer}
		window.Append(handler)
		subscriber.OnUnsubscribe(func() { window.Delete(handler) })
	}
}
