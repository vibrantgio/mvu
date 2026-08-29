//go:build !darwin

package desktop

// Away from macOS the menu declaration is inert. There is no per-application
// menu bar to amend — Linux and Windows put their menus inside the window,
// which is the application's own layout rather than this package's — so the
// items are remembered, nothing native is touched, and no message is ever
// posted. The chords an application declares beside its menu, in the window
// itself, are what carry the actions here; that is why [MenuItem.Key]
// documents declaring both as the right thing to do.

func (m *MenuBar) install() {}
