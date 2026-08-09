// Package stream provides the one observable primitive ADR-008 sanctions for
// its third destination — a genuine stream — and nothing else.
//
// ADR-008 sorts cross-widget state into three destinations: durable or
// app-meaningful state becomes a message and lives in the model; frame-scoped
// UI coordination becomes a plain value owned by the frame goroutine; and
// what is left over — a slow-changing value several consumers observe, like
// the user's persisted preferences — stays an observable. This package is
// that last case. It is deliberately not a bus: it carries state, not events.
//
// # Value is for state, not for events
//
// [Value] conflates. A consumer that falls behind skips intermediate values
// and converges on the newest one; it is never stranded on a stale value and
// it never blocks the producer. That is exactly right for "what are the
// current preferences" and exactly wrong for "what happened" — an event whose
// every occurrence is load-bearing is a message, which is ADR-008's first
// destination, not this one.
//
// # Why not a bare rx.Subject
//
// [rx.Subject] is the obvious spelling and it is the one this package exists
// to replace. Two defects, both measured:
//
//   - It leaks a subscription slot. Its subscription list reuses an entry only
//     once that entry's cursor has been parked, and the only code that parks a
//     cursor runs inside the subscription's own scheduled receiver task, which
//     Unsubscribe cancels before it can run. So a slot is consumed for the
//     life of the process. With rx's default subscription capacity of 32, a
//     process that opens and closes shells dies on the 33rd with "out of
//     subject subscriptions" — reported against whichever innocent caller
//     subscribed next. Measured: a subscribe/unsubscribe loop over one
//     rx.Subject fails on iteration 32, and does so with a data race the
//     detector reports inside rx itself.
//   - The departed subscription's frozen cursor pins the producer. The ring
//     buffer's window follows the slowest cursor, so once the producer has
//     written bufCap more items it blocks forever, on whatever goroutine
//     called it. For a value emitted from the Gio frame goroutine that is a
//     hung application, not a dropped signal.
//
// # Why not a wrapper around rx.Subject either
//
// github.com/vibrantgio/prism/coordination wrapped rx.Subject in a
// subscription registry to fix both of those, and it did — in 292 lines, with
// a 64-subscriber ceiling as a leak detector, and still delivering through a
// private per-subscription rx.Subject. It therefore kept the third defect: a
// live consumer that stops draining still blocks the producer. It also kept
// rx.Subject's delivery cost, because a parked rx receiver spinlocks in 50 µs
// increments waiting for the sender.
//
// [rx.Observable.Behavior] has none of that. Its subscriber set is a map, so
// unsubscribing removes the entry: there is no slot to leak and no ceiling to
// reach. Its write never blocks, and its concurrent receiver blocks on a wake
// signal rather than spinning. Measured on an M1 Max, one full arrival cycle —
// publish, deliver, and the consumer's read:
//
//	bare rx.Subject                     51.8 µs   95 B   1 alloc
//	prism/coordination.Subject (292 ln) 52.0 µs   98 B   2 allocs
//	stream.Value (this package)          1.3 µs   16 B   1 alloc
//
// The lesson worth keeping is that the primitive did not need writing. It
// needed choosing: rx shipped a lifetime-safe multicast all along, under a
// different name.
//
// # Why the source hands its observer back
//
// [rx.Observable.Behavior] is a Connectable over a source observable, and the
// obvious source — an rx.Subject, or rx.Multicast — puts a buffer and a
// forwarding goroutine between the producer and the value cell. That costs a goroutine
// per stream for as long as the process lives, and it makes the write
// asynchronous: a consumer subscribing in the microseconds after a write sees
// the previous value first (measured; it converges, but it is a stale
// emission that need not exist).
//
// So the source here does nothing at all except hand its observer back when
// the Connectable connects. The producer then writes rx.Behavior's conflating
// cell directly, on its own goroutine, synchronously. An idle stream costs
// zero goroutines; a live consumer costs one, which is rx's.
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
// receiving every intermediate one — see the package doc. If every value is
// load-bearing, this is the wrong mechanism.
func Value[T any](seed T) (rx.Observer[T], rx.Observable[T]) {
	// send is written by the source below and read after Connect returns, both
	// on this goroutine: rx.Connectable.Connect calls its connector — and so
	// the source — synchronously, on the caller's goroutine. The panic is what
	// makes that assumption an assertion rather than a comment; nothing here
	// needs a lock if it holds, and nothing here would be safe if it did not.
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
