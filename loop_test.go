package mvu

import (
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reactivego/rx"
)

// Add is the test message: it adds N to the int model.
type Add struct{ N int }

// collect subscribes models and returns a snapshot function plus the
// subscription.
func collect(models rx.Observable[int]) (func() []int, rx.Subscription) {
	var mu sync.Mutex
	var seen []int
	sub := models.Subscribe(rx.GoroutineContext(), func(next int, err error, done bool) {
		if !done {
			mu.Lock()
			seen = append(seen, next)
			mu.Unlock()
		}
	})
	snapshot := func() []int {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(seen)
	}
	return snapshot, sub
}

// await polls snapshot until cond holds or the timeout expires.
func await(t *testing.T, snapshot func() []int, cond func([]int) bool) []int {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if seen := snapshot(); cond(seen) {
			return seen
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition not met before timeout; models seen: %v", snapshot())
	return nil
}

func last(seen []int) int { return seen[len(seen)-1] }

// TestLoopEmitsSeedFirst asserts the models observable starts with the seed
// before any message arrives.
func TestLoopEmitsSeedFirst(t *testing.T) {
	in := make(chan Message, 1)
	update := func(m int, msg Message) (int, Command) { return m, DoNothing() }

	init := func() (int, Command) { return 42, DoNothing() }
	models, runner := Loop(rx.Recv(in), init, update)
	defer func() { runner.Unsubscribe(); runner.Wait() }()
	snapshot, sub := collect(models)
	defer sub.Unsubscribe()

	seen := await(t, snapshot, func(seen []int) bool { return len(seen) > 0 })
	if seen[0] != 42 {
		t.Fatalf("first emission = %d, want seed 42", seen[0])
	}
}

// TestLoopReducesExternalMessages asserts messages from the input observable
// flow through update.
func TestLoopReducesExternalMessages(t *testing.T) {
	in := make(chan Message, 8)
	update := func(m int, msg Message) (int, Command) {
		if add, ok := msg.(Add); ok {
			return m + add.N, DoNothing()
		}
		return m, DoNothing()
	}

	init := func() (int, Command) { return 0, DoNothing() }
	models, runner := Loop(rx.Recv(in), init, update)
	defer func() { runner.Unsubscribe(); runner.Wait() }()
	snapshot, sub := collect(models)
	defer sub.Unsubscribe()

	in <- Add{N: 1}
	in <- Add{N: 2}

	seen := await(t, snapshot, func(seen []int) bool { return len(seen) > 0 && last(seen) == 3 })
	if last(seen) != 3 {
		t.Fatalf("model = %d, want 3", last(seen))
	}
}

// TestLoopRunsInitialCommand asserts the initial command's message feeds back
// into update without any external message.
func TestLoopRunsInitialCommand(t *testing.T) {
	in := make(chan Message, 1)
	update := func(m int, msg Message) (int, Command) {
		if add, ok := msg.(Add); ok {
			return m + add.N, DoNothing()
		}
		return m, DoNothing()
	}
	initial := Do(func() (Message, error) { return Add{N: 7}, nil })

	init := func() (int, Command) { return 0, initial }
	models, runner := Loop(rx.Recv(in), init, update)
	defer func() { runner.Unsubscribe(); runner.Wait() }()
	snapshot, sub := collect(models)
	defer sub.Unsubscribe()

	await(t, snapshot, func(seen []int) bool { return len(seen) > 0 && last(seen) == 7 })
}

// TestLoopFeedsBackUpdateCommands asserts a command returned by update emits
// messages that reach update again (the chain 1 → 10 → 100 terminates when
// update stops returning commands).
func TestLoopFeedsBackUpdateCommands(t *testing.T) {
	in := make(chan Message, 1)
	update := func(m int, msg Message) (int, Command) {
		add, ok := msg.(Add)
		if !ok {
			return m, DoNothing()
		}
		next := m + add.N
		if add.N < 100 {
			chained := add.N * 10
			return next, Do(func() (Message, error) { return Add{N: chained}, nil })
		}
		return next, DoNothing()
	}

	init := func() (int, Command) { return 0, DoNothing() }
	models, runner := Loop(rx.Recv(in), init, update)
	defer func() { runner.Unsubscribe(); runner.Wait() }()
	snapshot, sub := collect(models)
	defer sub.Unsubscribe()

	in <- Add{N: 1}

	await(t, snapshot, func(seen []int) bool { return len(seen) > 0 && last(seen) == 111 })
}

// TestLoopSurvivesCommandError asserts a failing command is contained: the
// loop keeps reducing messages that arrive afterwards.
func TestLoopSurvivesCommandError(t *testing.T) {
	in := make(chan Message, 8)
	update := func(m int, msg Message) (int, Command) {
		if add, ok := msg.(Add); ok {
			return m + add.N, DoNothing()
		}
		return m, DoNothing()
	}
	initial := Do(func() (Message, error) { return nil, errors.New("boom") })

	init := func() (int, Command) { return 0, initial }
	models, runner := Loop(rx.Recv(in), init, update)
	defer func() { runner.Unsubscribe(); runner.Wait() }()
	snapshot, sub := collect(models)
	defer sub.Unsubscribe()

	in <- Add{N: 5}

	await(t, snapshot, func(seen []int) bool { return len(seen) > 0 && last(seen) == 5 })
}

// TestLoopSubscribesMessagesExactlyOnce counts the cold subscriptions Loop
// makes to its messages input, pinning the AutoConnect arithmetic doc.go calls
// load-bearing: the internal Publish().AutoConnect(2) — models and commands —
// must connect exactly once, subscribing the merged message sources exactly
// once. Both drifts are silent (doc.go): too high never connects, so the
// count stays 0 and the await below times out with no message reduced; too
// low connects early or double-subscribes, and the count leaves 1. This is
// the test that counts instead of hand-tuning — anyone adding a subscriber
// path inside Loop (the way ViewEvents added one beside it on Window) must
// keep this passing, not adjust the constant to match.
func TestLoopSubscribesMessagesExactlyOnce(t *testing.T) {
	in := make(chan Message, 8)
	var subscriptions atomic.Int32
	base := rx.Recv(in)
	counted := rx.Observable[Message](func(observe rx.Observer[Message], scheduler rx.Scheduler, subscriber rx.Subscriber) {
		subscriptions.Add(1)
		base(observe, scheduler, subscriber)
	})
	update := func(m int, msg Message) (int, Command) {
		if add, ok := msg.(Add); ok {
			return m + add.N, DoNothing()
		}
		return m, DoNothing()
	}

	init := func() (int, Command) { return 0, DoNothing() }
	models, runner := Loop(counted, init, update)
	defer func() { runner.Unsubscribe(); runner.Wait() }()
	snapshot, sub := collect(models)
	defer sub.Unsubscribe()

	in <- Add{N: 3}

	// A reduced message proves the loop connected and is draining.
	await(t, snapshot, func(seen []int) bool { return len(seen) > 0 && last(seen) == 3 })

	if got := subscriptions.Load(); got != 1 {
		t.Fatalf("Loop subscribed its messages input %d times; want exactly 1 (AutoConnect arithmetic drifted)", got)
	}
}
