//go:build !darwin

package desktop

// Away from macOS the menu declaration is inert. There is no per-application
// menu bar to amend — Linux and Windows put their menus inside the window,
// which is the application's own layout rather than this package's — so the
// items are remembered, nothing native is touched, and no message is ever
// posted. The chords an application declares in the window itself are what
// carry the actions here.

func (m *MenuBar) install() {}

// show has no bar to change. [MenuBar.Set] keeps the item's state in Go.
func (m *MenuBar) show(tag int, title string, disabled bool) {}

// showItems has no bar to change. [MenuBar.SetItems] keeps the items in Go.
func (m *MenuBar) showItems(parent int, items []MenuItem, tags []int) {}
