package vibrant

import (
	"sync"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/op"

	"github.com/reactivego/x"
)

func KeyEvents(observable x.Observable[Input]) x.Observable[key.Event] {
	return x.SwitchMap(observable, func(input Input) x.Observable[key.Event] {
		return x.From(input.KeyEvents()...)
	})
}

func EditEvents(observable x.Observable[Input]) x.Observable[key.EditEvent] {
	return x.SwitchMap(observable, func(input Input) x.Observable[key.EditEvent] {
		return x.From(input.EditEvents()...)
	})
}

func FocusEvents(observable x.Observable[Input]) x.Observable[key.FocusEvent] {
	return x.SwitchMap(observable, func(input Input) x.Observable[key.FocusEvent] {
		return x.From(input.FocusEvents()...)
	})
}

type Input struct {
	event.Tag
	event.Queue
}

// Add declares a handler ready for key events.
// Key events are in general only delivered to the
// focused key handler.
//	hint is optional and defaults to key.HintAny
func (i Input) Add(ops *op.Ops, hint ...key.InputHint) {
	key.InputOp{Tag: i.Tag, Hint: append(hint, key.HintAny)[0]}.Add(ops)
}

// Focus sets or clears the keyboard focus.
// It replaces any previous Focus in the same frame.
func (i Input) Focus(ops *op.Ops, focus bool) {
	// Tag is the new focus. The focus is cleared if Tag is nil, or if Tag
	// has no InputOp in the same frame.
	if focus {
		key.FocusOp{Tag: i.Tag}.Add(ops)
	} else {
		key.FocusOp{Tag: nil}.Add(ops)
	}
}

// SoftKeyboardOp shows or hide the on-screen keyboard, if available.
// It replaces any previous SoftKeyboardOp.
func (i Input) SoftKeyboard(ops *op.Ops, show bool) {
	key.SoftKeyboardOp{Show: show}.Add(ops)
}

func (i Input) Events() []event.Event {
	return i.Queue.Events(i.Tag)
}

func (i Input) KeyEvents() []key.Event {
	var events []key.Event
	for _, event := range i.Queue.Events(i.Tag) {
		if event, ok := event.(key.Event); ok {
			events = append(events, event)
		}
	}
	return events
}

func (i Input) EditEvents() []key.EditEvent {
	var events []key.EditEvent
	for _, event := range i.Queue.Events(i.Tag) {
		if event, ok := event.(key.EditEvent); ok {
			events = append(events, event)
		}
	}
	return events
}

func (i Input) FocusEvents() []key.FocusEvent {
	var events []key.FocusEvent
	for _, event := range i.Queue.Events(i.Tag) {
		if event, ok := event.(key.FocusEvent); ok {
			events = append(events, event)
		}
	}
	return events
}

func (window *Window) Input() x.Observable[Input] {
	input := struct {
		sync.Mutex
		Map map[event.Tag][]event.Event
	}{Map: make(map[event.Tag][]event.Event)}
	return func(observe x.Observer[Input], scheduler x.Scheduler, subscriber x.Subscriber) {
		channel := make(chan any, 5)
		x.AsObservable[Input](x.FromChan(channel))(observe, scheduler, subscriber)
		tag := Tag()
		channel <- Input{Tag: tag, Queue: EventQueue([]event.Event{})}
		input.Lock()
		input.Map[tag] = nil
		input.Unlock()
		observer := func(next event.Event, err error, done bool) {
			switch {
			case !done:
				if frame, ok := next.(system.FrameEvent); ok {
					var events []event.Event
					for k := range input.Map {
						events = append(events, frame.Queue.Events(k)...)
					}
					if n := len(events); n > 0 {
						for k := range input.Map {
							input.Map[k] = events
						}
					}
					if subscriber.Subscribed() {
						if events := input.Map[tag]; events != nil {
							select {
							case channel <- Input{Tag: tag, Queue: EventQueue(events)}:
								input.Map[tag] = nil
							default:
								panic("Input: Channel Overflow")
							}
						}
					}
				}
			case err != nil:
				select {
				case channel <- err:
					// OK
				default:
					panic("Input: Channel Overflow")
				}
				close(channel) // currently unable to forward an error
			case err == nil:
				close(channel)
			}
		}
		handler := &EventHandler{observer}
		window.Append(handler)
		subscriber.OnUnsubscribe(func() { window.Delete(handler) })
	}
}
