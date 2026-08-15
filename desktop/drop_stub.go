//go:build !darwin

package desktop

import "gioui.org/app"

// Away from macOS the drop machinery compiles but does nothing: a
// [DropTarget] constructs, its message stream simply never emits, and no
// native code is touched. The zone registry, the messages and the hover
// tracking are platform-neutral and identical everywhere, so an application
// written against them gains native drops platform by platform as support
// arrives, without changing shape.

func (d *DropTarget) handleViewEvent(app.ViewEvent) {}

func (d *DropTarget) release() {}
