//go:build !darwin

package desktop

import (
	"testing"

	"gioui.org/app"

	"github.com/vibrantgio/mvu"
)

// Away from macOS the whole drop seam must be inert: constructing a target,
// feeding it a view event and releasing it all do nothing and touch nothing
// native. The platform-neutral pipeline above it is covered by the tests
// that run everywhere.
func TestDropSeamInertOffMacOS(t *testing.T) {
	w := mvu.NewWindow(app.Title("drop stub test"))
	d := NewDropTarget(w, &ZoneGroup{})
	d.handleViewEvent(nil)
	d.release()
	d.close()
}
