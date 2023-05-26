package vibrant

import (
	"sync"

	"gioui.org/app"
)

// Window handles the events of a single gioui app window.
type Window struct {
	*app.Window

	sync.Mutex
	handlers []Handler
}

func NewWindow(options ...app.Option) *Window {
	return &Window{
		Window: app.NewWindow(options...),
	}
}
