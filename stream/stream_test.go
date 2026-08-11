package stream_test

import (
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/mvu/stream"
)

// observe subscribes to obs and returns the subscription plus two channels.
// Both sends are non-blocking, so a caller that stops reading never wedges
// the receiver.
func observe[T any](obs rx.Observable[T]) (rx.Subscription, <-chan T, <-chan error) {
	values := make(chan T, 64)
	errs := make(chan error, 1)
	sub := obs.Subscribe(rx.GoroutineContext(), func(next T, err error, done bool) {
		if err != nil {
			select {
			case errs <- err:
			default:
			}
			return
		}
		if !done {
			select {
			case values <- next:
			default:
			}
		}
	})
	return sub, values, errs
}

func await[T any](t *testing.T, what string, values <-chan T, errs <-chan error) T {
	t.Helper()
	select {
	case v := <-values:
		return v
	case err := <-errs:
		t.Fatalf("%s: subscription failed: %v", what, err)
	case <-time.After(3 * time.Second):
		t.Fatalf("%s: timed out waiting for delivery", what)
	}
	var zero T
	return zero
}

// TestASubscriberSeesTheSeed is the replay half of the contract: subscribing
// to a stream nobody has written to yet still delivers a value.
func TestASubscriberSeesTheSeed(t *testing.T) {
	_, obs := stream.Value("seed")

	sub, values, errs := observe(obs)
	defer sub.Unsubscribe()
	if got := await(t, "the seed", values, errs); got != "seed" {
		t.Errorf("got %q, want %q", got, "seed")
	}
}

// TestALateSubscriberSeesTheCurrentValue is the other half: a write with
// nobody listening is not lost, it becomes the value the next subscriber
// starts from. This is what theme/preferences needs — a settings screen
// saves, a window opens later, and the window must not start from the value
// the file held at launch.
func TestALateSubscriberSeesTheCurrentValue(t *testing.T) {
	send, obs := stream.Value("seed")

	send("saved", nil, false)

	sub, values, errs := observe(obs)
	defer sub.Unsubscribe()
	if got := await(t, "the late subscriber", values, errs); got != "saved" {
		t.Errorf("got %q, want %q", got, "saved")
	}
}

// TestTheWriteIsSynchronous pins the reason the source hands its observer back
// instead of going through a buffer and a forwarding goroutine: a subscriber
// attaching immediately after a write sees the written value FIRST, with no
// intervening stale emission. Through an rx.Subject or rx.Multicast source the
// same loop delivers the previous value first, and converges only afterwards.
func TestTheWriteIsSynchronous(t *testing.T) {
	for i := range 200 {
		send, obs := stream.Value(-1)
		send(i, nil, false)

		sub, values, errs := observe(obs)
		if got := await(t, "immediately after the write", values, errs); got != i {
			t.Fatalf("iteration %d: got %d, want %d", i, got, i)
		}
		sub.Unsubscribe()
	}
}

// TestUnsubscribeReleasesEverything is the G0B.1 regression, carried over from
// prism/coordination. Over a bare rx.Subject this loop dies on iteration 32
// with rx's "out of subject subscriptions", reported in a test binary against
// whichever unlucky test subscribed next; prism/coordination survived it by
// keeping its own registry, and this survives it because rx.Behavior's
// subscriber set is a map with nothing to leak.
//
// The loop deliberately runs far past any plausible ceiling with only ever one
// live subscription, because that is the shape of a long-running application:
// shells open and close, and at no instant are many subscribed at once.
func TestUnsubscribeReleasesEverything(t *testing.T) {
	send, obs := stream.Value(-1)

	before := runtime.NumGoroutine()
	for i := range 300 {
		send(i, nil, false)
		sub, values, errs := observe(obs)
		if got := await(t, "iteration", values, errs); got != i {
			t.Fatalf("iteration %d: got %d", i, got)
		}
		sub.Unsubscribe()
	}

	// A released subscription releases its goroutine too, or the leak has
	// merely moved.
	deadline := time.Now().Add(3 * time.Second)
	for runtime.NumGoroutine() > before+2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before+2 {
		t.Errorf("goroutines went %d -> %d over 300 subscribe/unsubscribe cycles", before, after)
	}
}

// TestTheProducerRunsFreeBehindAWedgedConsumer is the harsher half of the same
// defect, and the one prism/coordination did NOT fix. A bare rx.Subject pins
// its ring window to the slowest cursor, so a consumer that stops draining
// blocks the producer forever — and the wrapper kept that, because it
// delivered through a private rx.Subject per subscription. rx.Behavior's write
// conflates instead of blocking.
func TestTheProducerRunsFreeBehindAWedgedConsumer(t *testing.T) {
	send, obs := stream.Value(-1)

	// A consumer that departs...
	sub, values, errs := observe(obs)
	send(0, nil, false)
	await(t, "priming delivery", values, errs)
	sub.Unsubscribe()

	// ...and a consumer that stays and never drains.
	block := make(chan struct{})
	wedged := obs.Subscribe(rx.GoroutineContext(), func(int, error, bool) { <-block })
	defer func() { close(block); wedged.Unsubscribe() }()
	time.Sleep(50 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 10000 {
			send(i, nil, false)
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the producer blocked behind a consumer that stopped draining")
	}
}

// TestThereIsNoSubscriberCeiling. prism/coordination allowed 64 concurrent
// subscriptions and reported ErrSubscriberLimit past that, a leak detector for
// a leak this design cannot have. There is no slot, so there is no limit.
func TestThereIsNoSubscriberCeiling(t *testing.T) {
	const n = 200
	send, obs := stream.Value(0)

	subs := make([]rx.Subscription, 0, n)
	values := make([]<-chan int, 0, n)
	errs := make([]<-chan error, 0, n)
	for range n {
		sub, v, e := observe(obs)
		subs = append(subs, sub)
		values = append(values, v)
		errs = append(errs, e)
	}
	defer func() {
		for _, s := range subs {
			s.Unsubscribe()
		}
	}()

	send(7, nil, false)
	for i := range n {
		// Each consumer sees the seed and then 7, or conflates straight to 7.
		if got := await(t, "fan-out", values[i], errs[i]); got == 0 {
			got = await(t, "fan-out after the seed", values[i], errs[i])
			if got != 7 {
				t.Fatalf("subscriber %d: got %d, want 7", i, got)
			}
		} else if got != 7 {
			t.Fatalf("subscriber %d: got %d, want 7", i, got)
		}
	}
}

// TestAWriteRacingASubscriptionIsNotLost is the race a hand-rolled replay
// cannot win: read the current value, then attach, and a write that lands
// between the two leaves the new subscriber holding a stale value forever.
// rx.Behavior closes that window because the receiver reads the cell after it
// is registered, so the value is delivered rather than replayed.
func TestAWriteRacingASubscriptionIsNotLost(t *testing.T) {
	for range 200 {
		send, obs := stream.Value(0)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() { defer wg.Done(); send(1, nil, false) }()

		sub, values, errs := observe(obs)
		for got := 0; got != 1; {
			got = await(t, "convergence on the racing write", values, errs)
		}
		sub.Unsubscribe()
		wg.Wait()
	}
}

// TestCompletionReachesEverySubscriber checks the producer's done signal
// terminates every live subscription, and that a subscriber arriving after
// completion is completed rather than left hanging on a dead stream.
func TestCompletionReachesEverySubscriber(t *testing.T) {
	send, obs := stream.Value(0)

	const n = 3
	dones := make([]chan error, n)
	subs := make([]rx.Subscription, n)
	for i := range n {
		done := make(chan error, 1)
		dones[i] = done
		subs[i] = obs.Subscribe(rx.GoroutineContext(), func(_ int, err error, complete bool) {
			if complete {
				select {
				case done <- err:
				default:
				}
			}
		})
	}
	defer func() {
		for _, s := range subs {
			s.Unsubscribe()
		}
	}()

	sentinel := errors.New("producer finished")
	send(0, sentinel, true)

	for i := range n {
		select {
		case err := <-dones[i]:
			if !errors.Is(err, sentinel) {
				t.Errorf("subscriber %d completed with %v, want %v", i, err, sentinel)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("subscriber %d never completed", i)
		}
	}

	late := make(chan error, 1)
	lateSub := obs.Subscribe(rx.GoroutineContext(), func(_ int, err error, complete bool) {
		if complete {
			select {
			case late <- err:
			default:
			}
		}
	})
	defer lateSub.Unsubscribe()
	select {
	case err := <-late:
		if !errors.Is(err, sentinel) {
			t.Errorf("late subscriber completed with %v, want %v", err, sentinel)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("a subscriber arriving after completion was left hanging")
	}
}

// TestUnsubscribeDeliversNoCompletion guards the seam between leaving and
// ending: a consumer that unsubscribes must not see that as the stream
// completing, because a stream that outlives it has not completed.
func TestUnsubscribeDeliversNoCompletion(t *testing.T) {
	_, obs := stream.Value(0)

	completed := make(chan struct{}, 1)
	sub := obs.Subscribe(rx.GoroutineContext(), func(_ int, _ error, complete bool) {
		if complete {
			select {
			case completed <- struct{}{}:
			default:
			}
		}
	})
	time.Sleep(50 * time.Millisecond)
	sub.Unsubscribe()

	select {
	case <-completed:
		t.Fatal("Unsubscribe delivered a completion to the observer")
	case <-time.After(250 * time.Millisecond):
	}
}

// TestAnIdleStreamCostsNoGoroutine is the reason the source hands its observer
// back rather than being an rx.Subject the Connectable forwards from: a stream
// that nobody is watching runs nothing at all. A per-path registry that never
// shrinks — theme/preferences has one — would otherwise accumulate a
// forwarding goroutine per path for the life of the process.
func TestAnIdleStreamCostsNoGoroutine(t *testing.T) {
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	before := runtime.NumGoroutine()

	keep := make([]rx.Observable[int], 0, 50)
	for range 50 {
		_, obs := stream.Value(0)
		keep = append(keep, obs)
	}
	time.Sleep(200 * time.Millisecond)

	if after := runtime.NumGoroutine(); after > before+1 {
		t.Errorf("50 idle streams cost %d goroutines", after-before)
	}
	runtime.KeepAlive(keep)
}

// BenchmarkArrival is the number the package doc quotes: one full cycle of
// publish, deliver, and the consumer's read.
func BenchmarkArrival(b *testing.B) {
	send, obs := stream.Value(0)
	got := make(chan int, 1)
	sub := obs.Subscribe(rx.GoroutineContext(), func(v int, _ error, done bool) {
		if !done {
			select {
			case got <- v:
			default:
			}
		}
	})
	defer sub.Unsubscribe()
	<-got // the seed

	b.ReportAllocs()
	b.ResetTimer()
	for i := 1; b.Loop(); i++ {
		send(i, nil, false)
		<-got
	}
}

// BenchmarkWrite is the write on its own, with a consumer attached.
func BenchmarkWrite(b *testing.B) {
	send, obs := stream.Value(0)
	sub := obs.Subscribe(rx.GoroutineContext(), func(int, error, bool) {})
	defer sub.Unsubscribe()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		send(i, nil, false)
	}
}
