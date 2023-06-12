package vibrant

import (
	"sync"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/layout"

	"github.com/reactivego/x"
)

type Click struct {
	*gesture.Click
	Events  []gesture.ClickEvent
	Hovered bool
	Pressed bool
}

func (window *Window) Click() x.Observable[Click] {
	click := struct {
		sync.Mutex
		Map map[*gesture.Click][]event.Event
	}{Map: make(map[*gesture.Click][]event.Event)}
	return func(observe x.Observer[Click], scheduler x.Scheduler, subscriber x.Subscriber) {
		channel := make(chan any, 5)
		x.AsObservable[Click](x.FromChan(channel))(observe, scheduler, subscriber)
		tag := new(gesture.Click)
		channel <- Click{Click: tag}
		click.Lock()
		click.Map[tag] = nil
		click.Unlock()
		handler := NewHandler(
			func(gtx layout.Context) {
				var all []event.Event
				for k := range click.Map {
					all = append(all, gtx.Events(k)...)
				}
				if n := len(all); n > 0 {
					for k := range click.Map {
						click.Map[k] = all
					}
				}
				if subscriber.Subscribed() {
					if events := click.Map[tag]; events != nil {
						c := Click{
							Click:   tag,
							Events:  tag.Events(EventQueue(events)),
							Hovered: tag.Hovered(),
							Pressed: tag.Pressed(),
						}
						select {
						case channel <- c:
							click.Map[tag] = nil
						default:
							panic("Click: Channel Overflow")
						}
					}
				}
			}, func() {
				close(channel)
			})
		window.Append(handler)
		subscriber.OnUnsubscribe(func() { window.Delete(handler) })
	}
}
