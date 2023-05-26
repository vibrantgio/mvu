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

// Frame returns an observable that emits the current frame of the window.
// The frame is emitted whenever the window is resized or the system insets
// change or when the metrics change.
// For new subscribers, the current frame is emitted immediately.
func (window *Window) Frame() x.Observable[Frame] {
	var shared struct {
		sync.Mutex
		Frame
		Count int
	}
	update := func(next event.Event) (Frame, int) {
		shared.Lock()
		defer shared.Unlock()
		switch event := next.(type) {
		case app.ConfigEvent:
			event.Config.Size = shared.Size
			shared.Config = event.Config
		case system.FrameEvent:
			if shared.Metric != event.Metric || shared.Size != event.Size || shared.Insets != event.Insets {
				shared.Metric = event.Metric
				shared.Size = event.Size
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
		handler := NewHandler(func(next event.Event, done bool) {
			if done {
				close(channel)
				return
			}
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
		})
		window.Append(handler)
		subscriber.OnUnsubscribe(func() { window.Delete(handler) })
	}
}
