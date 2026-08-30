package desktop

import (
	"sync"
	"unicode/utf8"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/mvu"
)

// ApplicationMenu is the [MenuItem.Menu] title that names the application's
// OWN menu — the first menu in the bar, the one this platform titles with the
// application's name and where it keeps Hide and Quit, and therefore where it
// keeps Settings. An item declared with this title amends that menu instead of
// adding a menu of its own; every other title names a menu beside it.
//
// It is the empty string deliberately: the menu is not titled by the
// application, so there is no title to state, and an item that says nothing
// about which menu it belongs to belongs to the application's.
const ApplicationMenu = ""

// menuBuffer is the capacity of the channel between the native menu callback
// and the [MenuBar.Messages] subscriber. Menu choices arrive at the speed of
// a hand, and an application that merges Messages into its loop drains them
// as fast as any other message; the buffer only covers the moment before the
// loop is running.
const menuBuffer = 16

// MenuItem declares one item of the application's menu bar: which menu it
// belongs to, what it says, the chord it answers to, and the message choosing
// it posts. It is a declaration and nothing more — the platform layer owns
// when and how the bar is amended.
type MenuItem struct {
	// Menu is the title of the menu the item sits in, created on the item
	// that first names it and shown in the bar as written: "File", "View".
	// [ApplicationMenu], the empty title, is the application's own menu.
	Menu string

	// Title is the item's own label, shown as written. It is required: an
	// item with no label is a bug, not a separator.
	Title string

	// Key is the item's key equivalent — one character, with this platform's
	// command modifier implied, so "n" is Cmd-N and "," is Cmd-comma. The
	// empty string asks for no chord at all.
	//
	// A menu's key equivalent is answered by the menu bar before the key
	// reaches the window, so an item and an in-window accelerator on the same
	// chord do not both fire: the menu wins and the window's never sees it.
	// Declaring both is therefore the right thing to do when the action must
	// survive a platform whose bar this package does not amend — they post
	// the same message, and exactly one of them posts it.
	Key string

	// Msg is the message posted when the item is chosen. The same value is
	// posted on every choice, so it is a value the application's update
	// function reads as an intent — [MenuBar] never mutates it.
	Msg mvu.Message
}

// MenuBar amends the application's menu bar with items that post messages.
// Declare it once, beside the window, and merge [MenuBar.Messages] into the
// application's message stream:
//
//	menu := desktop.NewMenuBar(w,
//		desktop.MenuItem{Menu: "File", Title: "New Note", Key: "n", Msg: NewNote{}},
//		desktop.MenuItem{Title: "Settings…", Key: ",", Msg: OpenSettings{}},
//	)
//	models, runner := mvu.Loop(rx.Merge(w.Messages(), menu.Messages()), Init, Update)
//
// The bar belongs to the application rather than to any one window — macOS
// draws one bar for the process, whatever it has open — so a MenuBar is a
// process-wide declaration that takes a window only to know when the
// application is far enough along to be amended. Declaring a second MenuBar
// supersedes the first; two bars would be two claims on one bar.
//
// Away from macOS the declaration is inert: the constructor succeeds, the
// items are remembered, no native code is touched and no message is ever
// posted. An application therefore declares its menu unconditionally and
// gains the bar platform by platform, exactly as it declares a drop target.
type MenuBar struct {
	w     *mvu.Window
	items []MenuItem
	tags  []int

	msgs chan mvu.Message
}

// menuAction is what a native menu callback resolves to: the bar that owns
// the item and the message choosing it posts.
type menuAction struct {
	bar *MenuBar
	msg mvu.Message
}

// menuActions maps each declared item's tag to what choosing it does. It must
// be package-level state: the menu callback re-enters Go from Objective-C
// carrying nothing but the chosen item's integer tag, so no instance, closure
// or argument can carry the owning bar across, and a tag-keyed registry behind
// a mutex is the only bridge. Tags are handed out here and never reused, so a
// stale callback for a superseded bar's item resolves to that bar and posts
// into a stream nobody reads rather than into another bar's.
var (
	menuMu      sync.Mutex
	menuActions = map[int]menuAction{}
	menuNextTag int
)

// NewMenuBar declares items on the application's menu bar and returns the bar
// delivering their messages. Every item must carry a Title, and a Key must be
// a single character or empty; either violation panics, so a malformed
// declaration fails where it is written rather than as an item that silently
// never appears.
//
// Call it once, right after constructing w and before the window starts
// rendering. The amendment itself happens later — after the window's first
// frame, when there is an application to amend — and is this package's
// business, not the caller's.
func NewMenuBar(w *mvu.Window, items ...MenuItem) *MenuBar {
	m := &MenuBar{
		w:     w,
		items: make([]MenuItem, len(items)),
		tags:  make([]int, len(items)),
		msgs:  make(chan mvu.Message, menuBuffer),
	}
	copy(m.items, items)

	menuMu.Lock()
	for i, it := range m.items {
		if it.Title == "" {
			menuMu.Unlock()
			panic("desktop: MenuItem needs a Title (an item with no label is a bug, not a separator)")
		}
		if it.Key != "" && utf8.RuneCountInString(it.Key) != 1 {
			menuMu.Unlock()
			panic("desktop: MenuItem Key " + it.Key + " must be a single character or empty")
		}
		menuNextTag++
		m.tags[i] = menuNextTag
		menuActions[menuNextTag] = menuAction{bar: m, msg: it.Msg}
	}
	menuMu.Unlock()

	m.install()
	return m
}

// Messages returns the bar's message stream: the [MenuItem.Msg] value of each
// item as it is chosen, ready to merge into the application's message stream
// beside the window's own. Subscribe it once — it is backed by a per-bar
// channel, and two subscriptions would compete for the same messages.
//
// The stream does not complete. The menu bar outlives every window the
// application opens and is torn down with the process, so there is no moment
// at which it ends; a caller that needs an end merges it with a stream that
// has one.
func (m *MenuBar) Messages() rx.Observable[mvu.Message] {
	return rx.Recv(m.msgs)
}

// post hands one chosen item's message to the subscriber without ever
// blocking the calling thread: the native callback runs on the platform's UI
// thread, where blocking freezes the menu mid-click.
//
// On a full buffer the choice is dropped and said so, rather than evicted
// oldest-first the way hover events are: every menu choice is a distinct
// intent a person expressed, none supersedes another, and losing the newest
// is no worse than losing the oldest. A full buffer means nothing is draining
// the stream, which is a wiring bug the log names.
func (m *MenuBar) post(msg mvu.Message) {
	select {
	case m.msgs <- msg:
	default:
		debugf("desktop: menu message %T dropped — nothing is draining MenuBar.Messages()", msg)
	}
}

// chooseMenuTag routes one native menu callback to the bar that declared the
// item. An unknown tag — an item from a build that no longer exists — resolves
// nowhere, silently. The lock is held across the delivery; post never blocks,
// so the UI thread is never held up.
func chooseMenuTag(tag int) {
	menuMu.Lock()
	defer menuMu.Unlock()
	if a, ok := menuActions[tag]; ok {
		a.bar.post(a.msg)
	}
}
