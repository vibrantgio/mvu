// Package stream provides the one observable primitive for cross-component state
// that is neither a message nor a frame-local value: a slow-changing value
// several consumers observe, such as the user's persisted preferences.
//
// It is deliberately not a bus: it carries state, not events. [Value]
// conflates — a consumer that falls behind skips intermediate values and
// converges on the newest one; it is never stranded on a stale value and it
// never blocks the producer. That is right for "what are the current
// preferences" and wrong for "what happened". An event whose every occurrence
// is load-bearing belongs in a message, not here.
//
// The primitive is [rx.Observable.Behavior]. Its subscriber set is a map, so
// unsubscribing removes the entry: no subscription slot leaks and there is no
// subscriber ceiling. Its write never blocks, and its concurrent receiver
// blocks on a wake signal rather than spinning. A full arrival cycle —
// publish, deliver, and the consumer's read — measures 1.3 µs, 16 B, 1 alloc
// on an M1 Max. An rx.Subject in the same role costs 51.8 µs, leaks a
// subscription slot per unsubscribe, and lets a departed subscriber's frozen
// cursor block the producer forever — a hung application when the producer is
// the Gio frame goroutine.
//
// The source behind the Connectable does nothing except hand its observer
// back when the Connectable connects, and the producer then writes rx.Behavior's
// conflating cell directly, on its own goroutine, synchronously. Any real
// source — an rx.Subject or rx.Multicast — would put a buffer and a forwarding
// goroutine between producer and cell, costing a goroutine per stream for the
// life of the process and making the write asynchronous, so that a consumer
// subscribing in the microseconds after a write sees the previous value first.
// An idle stream costs zero goroutines; a live consumer costs one, which is rx's.
package stream

import "github.com/reactivego/rx"

// Value returns the two sides of a current-value stream: an [rx.Observer] the
// producer writes through, and an [rx.Observable] any number of consumers may
// subscribe to and unsubscribe from over the life of the process. A consumer
// observes the current value the moment it subscribes — seed until the first
// write, the newest written value after that — and then follows.
//
// Writing is `send(v, nil, false)`; completing is `send(zero, err, true)`,
// which delivers the terminal to every live consumer and to every consumer
// that subscribes afterwards, so nobody is left hanging on a dead stream.
// Unsubscribing is not a completion and delivers nothing.
//
// A consumer that falls behind converges on the newest value rather than
// receiving every intermediate one. If every value is load-bearing, this is
// the wrong mechanism.
func Value[T any](seed T) (rx.Observer[T], rx.Observable[T]) {
	// send is written by the source below and read after Connect returns, both
	// on this goroutine: rx.Connectable.Connect calls its connector — and so
	// the source — synchronously, on the caller's goroutine. Nothing here needs
	// a lock while that holds, and nothing here would be safe if it did not, so
	// the panic below asserts it.
	var send rx.Observer[T]
	source := rx.Observable[T](func(observe rx.Observer[T], _ rx.Scheduler, _ rx.Subscriber) {
		send = observe
	})
	connectable := source.Behavior(seed)
	connectable.Connector.Connect()
	if send == nil {
		panic("mvu/stream: rx.Connectable.Connect did not subscribe its source synchronously")
	}
	return send, connectable.Observable
}
