package desktop

import (
	"testing"
	"time"

	"gioui.org/app"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/mvu"
)

type menuNew struct{}
type menuSettings struct{}
type menuToggle struct{}

func testWindow(t *testing.T) *mvu.Window {
	t.Helper()
	return mvu.NewWindow(app.Title("menu test"))
}

// The bridge's whole contract, stated on the one path that can be driven
// without a person and a mouse: a tag arriving from the native side posts
// that item's message, and only that item's. Everything above this — the
// declaration, the routing, the stream — is the same code on every platform;
// what differs is only whether anything ever calls in.
func TestChoosingAnItemPostsItsMessage(t *testing.T) {
	m := NewMenuBar(testWindow(t),
		MenuItem{Menu: "File", Title: "New Chat", Key: "n", Msg: menuNew{}},
		MenuItem{Title: "Settings…", Key: ",", Msg: menuSettings{}},
		MenuItem{Menu: "View", Title: "Hide/Show Conversations", Key: "\\", Msg: menuToggle{}},
	)

	want := []mvu.Message{menuToggle{}, menuNew{}, menuSettings{}}
	order := []int{2, 0, 1}
	for i, idx := range order {
		chooseMenuTag(m.tags[idx])
		select {
		case got := <-m.msgs:
			if got != want[i] {
				t.Fatalf("choosing %q posted %T, want %T", m.items[idx].Title, got, want[i])
			}
		default:
			t.Fatalf("choosing %q posted nothing", m.items[idx].Title)
		}
	}
}

// The stream is the application's half of the seam, so it is exercised the
// way an application uses it rather than by reading the channel behind it.
func TestMessagesDeliverTheChoiceToASubscriber(t *testing.T) {
	m := NewMenuBar(testWindow(t),
		MenuItem{Menu: "File", Title: "New Chat", Key: "n", Msg: menuNew{}},
	)

	got := make(chan mvu.Message, 1)
	sub := m.Messages().Subscribe(rx.GoroutineContext(), func(msg mvu.Message, err error, done bool) {
		if !done {
			got <- msg
		}
	})
	defer sub.Unsubscribe()

	chooseMenuTag(m.tags[0])
	select {
	case msg := <-got:
		if _, ok := msg.(menuNew); !ok {
			t.Fatalf("Messages() delivered %T, want desktop.menuNew", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Messages() delivered nothing for a chosen item")
	}
}

// A tag from a build that no longer exists resolves nowhere. It must be
// silence rather than a panic: the callback arrives from AppKit, where a
// panic crosses back into Objective-C.
func TestAnUnknownTagPostsNothing(t *testing.T) {
	m := NewMenuBar(testWindow(t),
		MenuItem{Menu: "File", Title: "New Chat", Msg: menuNew{}},
	)
	chooseMenuTag(-1)
	select {
	case got := <-m.msgs:
		t.Fatalf("an unknown tag posted %T, want nothing", got)
	default:
	}
}

// Two bars declared in one process keep their own items: the tags are handed
// out once and never reused, so a choice can only reach the bar that declared
// it. This is what makes superseding a bar safe rather than crossed.
func TestBarsDoNotShareTheirItems(t *testing.T) {
	first := NewMenuBar(testWindow(t), MenuItem{Title: "First", Msg: menuNew{}})
	second := NewMenuBar(testWindow(t), MenuItem{Title: "Second", Msg: menuSettings{}})

	if first.tags[0] == second.tags[0] {
		t.Fatalf("both bars were handed tag %d; tags must never be reused", first.tags[0])
	}
	chooseMenuTag(second.tags[0])
	select {
	case got := <-first.msgs:
		t.Fatalf("the first bar received %T from the second bar's item", got)
	default:
	}
	select {
	case got := <-second.msgs:
		if _, ok := got.(menuSettings); !ok {
			t.Fatalf("the second bar received %T, want desktop.menuSettings", got)
		}
	default:
		t.Fatal("the second bar received nothing from its own item")
	}
}

// The callback runs on the platform's UI thread, so the post must return
// however full the buffer is — a blocked menu click freezes the menu itself.
// The overflow policy is drop-and-say-so, which this pins by asking for far
// more choices than the buffer holds and requiring the call to return at all.
func TestPostingNeverBlocksOnAFullBuffer(t *testing.T) {
	m := NewMenuBar(testWindow(t), MenuItem{Title: "New Chat", Msg: menuNew{}})

	done := make(chan struct{})
	go func() {
		for i := 0; i < menuBuffer*4; i++ {
			chooseMenuTag(m.tags[0])
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("choosing an item blocked once the buffer filled")
	}
	if len(m.msgs) != menuBuffer {
		t.Fatalf("the buffer holds %d messages, want the full %d", len(m.msgs), menuBuffer)
	}
}

// A malformed declaration fails where it is written. Both rules exist because
// the alternative is an item that silently never appears in the bar, which is
// the hardest kind of native bug to see.
func TestAMalformedDeclarationPanics(t *testing.T) {
	for _, tc := range []struct {
		name string
		item MenuItem
	}{
		{"no title", MenuItem{Menu: "File", Key: "n", Msg: menuNew{}}},
		{"multi-character key", MenuItem{Title: "New Chat", Key: "cmd-n", Msg: menuNew{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("declaring the item did not panic")
				}
			}()
			NewMenuBar(testWindow(t), tc.item)
		})
	}
}

// The empty menu title is the application's own menu, and it is a named
// constant precisely so an application does not write the emptiness out.
func TestApplicationMenuIsTheUntitledMenu(t *testing.T) {
	if ApplicationMenu != "" {
		t.Fatalf("ApplicationMenu = %q, want the empty title", ApplicationMenu)
	}
}
