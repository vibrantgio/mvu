package desktop

// ScrollBarSetting is the platform's "Show scroll bars" setting: when a
// scrollable region shows its bar.
type ScrollBarSetting int

const (
	// ScrollBarsAutomatic leaves it to the input device: a mouse gets a bar
	// that stands, a trackpad one that shows while scrolling. It is the
	// platform's own default.
	ScrollBarsAutomatic ScrollBarSetting = iota
	// ScrollBarsWhenScrolling shows the bar while the content scrolls and
	// under the pointer, and fades it otherwise.
	ScrollBarsWhenScrolling
	// ScrollBarsAlways keeps the bar for as long as the content overflows.
	ScrollBarsAlways
)

// String names the setting as the platform does.
func (s ScrollBarSetting) String() string {
	switch s {
	case ScrollBarsWhenScrolling:
		return "WhenScrolling"
	case ScrollBarsAlways:
		return "Always"
	default:
		return "Automatic"
	}
}

// ScrollBars reports when the user has asked for scroll bars to show.
//
// A scrollable region asks this once and spends the answer on its bar:
// Always is a bar that never fades, and the other two are the fading overlay
// bar the library draws by default. Automatic is told from WhenScrolling by
// the input device rather than by the setting, which no Gio window can see,
// so both are drawn the same way and the two are kept apart here only
// because the platform keeps them apart.
//
// Away from macOS there is nothing to read and the answer is
// [ScrollBarsWhenScrolling], the overlay bar every desktop platform draws
// for a trackpad.
func ScrollBars() ScrollBarSetting {
	return scrollBars()
}
