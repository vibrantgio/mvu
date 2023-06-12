package vibrant

import (
	"gioui.org/app"
	"gioui.org/io/pointer"
	"gioui.org/op"
	"gioui.org/op/clip"
	"github.com/reactivego/gio"
)

type MsgOp struct{ Msg any }

func (op MsgOp) Add(ops *op.Ops) {
	defer clip.Rect{}.Push(ops).Pop()
	pointer.InputOp{Tag: op}.Add(ops)
}

var _MsgOps = Map[*app.Window, chan MsgOp]{}

func MsgOps(window *app.Window, size int) <-chan MsgOp {
	if ch, ok := _MsgOps.Load(window); ok {
		return ch
	}
	ch, _ := _MsgOps.LoadOrStore(window, make(chan MsgOp, size))
	return ch
}

func init() {
	gio.AddFrameEventHook(func(window *app.Window, op any) bool {
		msgop, matches := op.(MsgOp)
		if matches {
			if ch, ok := _MsgOps.Load(window); ok {
				ch <- msgop
			}
		}
		return matches
	})
}
