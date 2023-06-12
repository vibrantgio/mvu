package vibrant

import (
	"sync"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/reactivego/x"
)

/*
func KeyEvents(observable x.Observable[Key]) x.Observable[key.Event] {
	return x.SwitchMap(observable, func(k Key) x.Observable[key.Event] {
		return x.From(k.Events()...)
	})
}

func EditEvents(observable x.Observable[Key]) x.Observable[key.EditEvent] {
	return x.SwitchMap(observable, func(k Key) x.Observable[key.EditEvent] {
		return x.From(k.EditEvents()...)
	})
}

func FocusEvents(observable x.Observable[Key]) x.Observable[key.FocusEvent] {
	return x.SwitchMap(observable, func(k Key) x.Observable[key.FocusEvent] {
		return x.From(k.FocusEvents()...)
	})
}

func SnippetEvents(observable x.Observable[Key]) x.Observable[key.SnippetEvent] {
	return x.SwitchMap(observable, func(k Key) x.Observable[key.SnippetEvent] {
		return x.From(k.SnippetEvents()...)
	})
}

func SelectionEvents(observable x.Observable[Key]) x.Observable[key.SelectionEvent] {
	return x.SwitchMap(observable, func(k Key) x.Observable[key.SelectionEvent] {
		return x.From(k.SelectionEvents()...)
	})
}
*/

type Keyboard struct {
	event.Tag
	event.Queue
}

// Input declares a handler ready for key events.
// Key events are in general only delivered to the
// focused key handler.
//
//	hint is optional and defaults to key.HintAny
func (i Keyboard) Input(ops *op.Ops, hint ...key.InputHint) {
	key.InputOp{Tag: i.Tag, Hint: append(hint, key.HintAny)[0]}.Add(ops)
}

// Focus sets or clears the keyboard focus.
// It replaces any previous Focus in the same frame.
func (i Keyboard) Focus(ops *op.Ops, focus bool) {
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
func (i Keyboard) SoftKeyboard(ops *op.Ops, show bool) {
	key.SoftKeyboardOp{Show: show}.Add(ops)
}

// Selection sets the selection range and caret.
func (i Keyboard) Selection(ops *op.Ops, r key.Range, c key.Caret) {
	key.SelectionOp{Tag: i.Tag, Range: r, Caret: c}.Add(ops)
}

// Snippet sets the snippet to be inserted at the caret position.
func (i Keyboard) Snippet(ops *op.Ops, snippet key.Snippet) {
	key.SnippetOp{Tag: i.Tag, Snippet: snippet}.Add(ops)
}

func (i Keyboard) KeyEvents() []key.Event {
	var events []key.Event
	for _, event := range i.Queue.Events(i.Tag) {
		if event, ok := event.(key.Event); ok {
			events = append(events, event)
		}
	}
	return events
}

func (i Keyboard) EditEvents() []key.EditEvent {
	var events []key.EditEvent
	for _, event := range i.Queue.Events(i.Tag) {
		if event, ok := event.(key.EditEvent); ok {
			events = append(events, event)
		}
	}
	return events
}

func (i Keyboard) FocusEvents() []key.FocusEvent {
	var events []key.FocusEvent
	for _, event := range i.Queue.Events(i.Tag) {
		if event, ok := event.(key.FocusEvent); ok {
			events = append(events, event)
		}
	}
	return events
}

func (i Keyboard) SnippetEvents() []key.SnippetEvent {
	var events []key.SnippetEvent
	for _, event := range i.Queue.Events(i.Tag) {
		if event, ok := event.(key.SnippetEvent); ok {
			events = append(events, event)
		}
	}
	return events
}

func (i Keyboard) SelectionEvents() []key.SelectionEvent {
	var events []key.SelectionEvent
	for _, event := range i.Queue.Events(i.Tag) {
		if event, ok := event.(key.SelectionEvent); ok {
			events = append(events, event)
		}
	}
	return events
}

func (window *Window) Keyboard() x.Observable[Keyboard] {
	input := struct {
		sync.Mutex
		Map map[event.Tag][]event.Event
	}{Map: make(map[event.Tag][]event.Event)}
	return func(observe x.Observer[Keyboard], scheduler x.Scheduler, subscriber x.Subscriber) {
		channel := make(chan any, 5)
		x.AsObservable[Keyboard](x.FromChan(channel))(observe, scheduler, subscriber)
		tag := new(struct{})
		channel <- Keyboard{Tag: tag, Queue: EventQueue([]event.Event{})}
		input.Lock()
		input.Map[tag] = nil
		input.Unlock()
		handler := NewHandler(
			func(gtx layout.Context) {
				var all []event.Event
				for k := range input.Map {
					all = append(all, gtx.Events(k)...)
				}
				if n := len(all); n > 0 {
					for k := range input.Map {
						input.Map[k] = all
					}
				}
				if subscriber.Subscribed() {
					if events := input.Map[tag]; events != nil {
						select {
						case channel <- Keyboard{Tag: tag, Queue: EventQueue(events)}:
							input.Map[tag] = nil
						default:
							panic("Input: Channel Overflow")
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
