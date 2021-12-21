package main

import (
	"os"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/op"
	"golang.org/x/exp/shiny/materialdesign/colornames"

	_ "github.com/reactivego/rx"
	_ "github.com/reactivego/vibrant/gio/generic"

	"github.com/reactivego/vibrant/gio"
	"github.com/reactivego/vibrant/text"

	roboto "github.com/reactivego/vibrant/roboto/italic"
)

func main() {
	go Hello()
	app.Main()
}

func Hello() {
	window := gio.NewWindow(app.Title("Vibrant - Hello"))

	shaper := roboto.Shaper()

	frames := ExtendGioObservableFrameEvent(window.FrameEvents())
	AliasGioObservableCallOp()

	loading := gio.BlankScreen(colornames.BlueGrey600)
	backdrop := gio.FromCallOp(loading)
	caption := frames.MapCallOp(func(fe gio.FrameEvent) CallOp {
		print := func(r f32.Rectangle, txt string, ax, ay float32) op.CallOp {
			ops := &op.Ops{}
			m := op.Record(ops)
			text.Print(shaper, txt, r, ax, ay, 1000, roboto.H1.Scale(fe.Metric), colornames.DeepOrangeA100, ops)
			return m.Stop()
		}
		return print(f32.Rect(0, 0, float32(fe.Size.X), float32(fe.Size.Y)), "Hello, World!", 0.5, 0.5)
	})

	window.Frame(loading, backdrop, caption).Wait()
	os.Exit(0)
}
