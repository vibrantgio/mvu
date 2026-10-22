//go:build !darwin

package desktop

// Away from macOS there is no setting to read: the overlay bar that shows
// while the content scrolls is what the other desktop platforms draw, and it
// is what the library draws by default.
func scrollBars() ScrollBarSetting { return ScrollBarsWhenScrolling }
