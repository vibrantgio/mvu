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
		{"items with no submenu", MenuItem{Title: "Move", Items: []MenuItem{{Title: "Up", Msg: menuNew{}}}}},
		{"submenu with a key", MenuItem{Title: "Move", Key: "m", Submenu: true}},
		{"submenu with a message", MenuItem{Title: "Move", Submenu: true, Msg: menuNew{}}},
		{"submenu in a submenu", MenuItem{Title: "Move", Submenu: true, Items: []MenuItem{{Title: "To", Submenu: true}}}},
		{"menu in a submenu", MenuItem{Title: "Move", Submenu: true, Items: []MenuItem{{Menu: "Card", Title: "Up", Msg: menuNew{}}}}},
		{"untitled item in a submenu", MenuItem{Title: "Move", Submenu: true, Items: []MenuItem{{Msg: menuNew{}}}}},
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

// Set changes the item's title and disabled state, item by item, and a
// declared Disabled item starts disabled. The bar keeps the state of the
// items that Set did not name.
func TestSetChangesTheItemsTitleAndState(t *testing.T) {
	m := NewMenuBar(testWindow(t),
		MenuItem{Menu: "Edit", Title: "Undo", Disabled: true, Msg: menuNew{}},
		MenuItem{Menu: "Edit", Title: "Redo", Disabled: true, Msg: menuSettings{}},
	)
	if !m.items[0].Disabled || !m.items[1].Disabled {
		t.Fatal("the declared Disabled items do not start disabled")
	}
	m.Set(0, "Undo Delete Card", false)
	if got := m.items[0]; got.Title != "Undo Delete Card" || got.Disabled {
		t.Fatalf("item 0 is %q disabled=%v, want \"Undo Delete Card\" enabled", got.Title, got.Disabled)
	}
	if got := m.items[1]; got.Title != "Redo" || !got.Disabled {
		t.Fatalf("item 1 is %q disabled=%v, want \"Redo\" disabled as declared", got.Title, got.Disabled)
	}
	m.Set(0, "Undo", true)
	if got := m.items[0]; got.Title != "Undo" || !got.Disabled {
		t.Fatalf("item 0 is %q disabled=%v, want \"Undo\" disabled", got.Title, got.Disabled)
	}
	// The item's message stays its own after Set.
	chooseMenuTag(m.tags[0])
	select {
	case got := <-m.msgs:
		if _, ok := got.(menuNew); !ok {
			t.Fatalf("item 0 posted %T after Set, want desktop.menuNew", got)
		}
	default:
		t.Fatal("item 0 posted nothing after Set")
	}
}

// An empty title is a bug in Set as it is in the declaration.
func TestSetWithAnEmptyTitlePanics(t *testing.T) {
	m := NewMenuBar(testWindow(t), MenuItem{Title: "Undo", Msg: menuNew{}})
	defer func() {
		if recover() == nil {
			t.Fatal("Set with an empty title did not panic")
		}
	}()
	m.Set(0, "", false)
}

// menuColumn is the message of one item of a submenu that SetItems fills.
type menuColumn struct{ Column int }

// columnItems is a submenu's items, one per title, each posting its index.
// The item at disabled is disabled.
func columnItems(disabled int, titles ...string) []MenuItem {
	items := make([]MenuItem, len(titles))
	for i, title := range titles {
		items[i] = MenuItem{Title: title, Disabled: i == disabled, Msg: menuColumn{i}}
	}
	return items
}

// chosen is the message that choosing the item with tag posts, or nil.
func chosen(m *MenuBar, tag int) any {
	chooseMenuTag(tag)
	select {
	case got := <-m.msgs:
		return got
	default:
		return nil
	}
}

// An item of a submenu posts its own message, and the submenu item posts
// nothing. SetItems replaces the items and their messages. A change of
// state alone keeps the items' tags, and a change of the item count gives
// new tags and retires the old ones.
func TestSetItemsReplacesASubmenusItems(t *testing.T) {
	m := NewMenuBar(testWindow(t),
		MenuItem{Menu: "Card", Title: "Move Item", Submenu: true, Items: []MenuItem{
			{Title: "Up", Key: UpArrow, Control: true, Msg: menuNew{}},
			{Title: "Down", Key: DownArrow, Control: true, Msg: menuSettings{}},
		}},
		MenuItem{Menu: "Card", Title: "Move to Column", Submenu: true},
	)
	if got := chosen(m, m.subTags[0][1]); got != (menuSettings{}) {
		t.Fatalf("choosing Down posted %v, want desktop.menuSettings", got)
	}
	if got := chosen(m, m.tags[0]); got != nil {
		t.Fatalf("choosing the submenu item posted %T", got)
	}
	if len(m.subTags[1]) != 0 {
		t.Fatalf("the empty submenu holds %d tags", len(m.subTags[1]))
	}

	m.SetItems(1, columnItems(0, "To Do", "Done")...)
	first := append([]int(nil), m.subTags[1]...)
	if got := chosen(m, first[1]); got != (menuColumn{1}) {
		t.Fatalf("choosing Done posted %v, want column 1", got)
	}

	m.SetItems(1, columnItems(1, "To Do", "Done")...)
	if m.subTags[1][0] != first[0] || m.subTags[1][1] != first[1] {
		t.Fatal("a change of state alone gave the items new tags")
	}
	if got := m.items[1].Items[1]; !got.Disabled {
		t.Fatal("SetItems did not disable Done")
	}

	m.SetItems(1, columnItems(2, "To Do", "In Progress", "Done")...)
	if len(m.subTags[1]) != 3 || m.subTags[1][0] == first[0] {
		t.Fatalf("a new item count kept the tags %v", m.subTags[1])
	}
	if got := chosen(m, first[1]); got != nil {
		t.Fatalf("a retired tag posted %v", got)
	}
	if got := chosen(m, m.subTags[1][2]); got != (menuColumn{2}) {
		t.Fatalf("choosing Done posted %v, want column 2", got)
	}
}

// SetItems takes only a submenu item, and only well-formed items.
func TestSetItemsPanicsOnAMalformedCall(t *testing.T) {
	for _, tc := range []struct {
		name  string
		i     int
		items []MenuItem
	}{
		{"no submenu", 0, nil},
		{"untitled item", 1, []MenuItem{{Msg: menuNew{}}}},
		{"submenu in a submenu", 1, []MenuItem{{Title: "To", Submenu: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMenuBar(testWindow(t),
				MenuItem{Menu: "Edit", Title: "Undo", Msg: menuNew{}},
				MenuItem{Menu: "Card", Title: "Move to Column", Submenu: true},
			)
			defer func() {
				if recover() == nil {
					t.Fatal("SetItems did not panic")
				}
			}()
			m.SetItems(tc.i, tc.items...)
		})
	}
}
