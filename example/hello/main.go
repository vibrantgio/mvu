package main

import (
	"os"

	"golang.org/x/exp/shiny/materialdesign/colornames"

	"gioui.org/app"
	"gioui.org/op"

	"github.com/reactivego/vibrant"
	"github.com/reactivego/vibrant/font/roboto"
	"github.com/reactivego/vibrant/theme"
	"github.com/reactivego/x"
)

func main() {
	go Hello()
	app.Main()
}

func Hello() {
	window := vibrant.NewWindow(app.Title("Vibrant - Hello"))

	drawing := x.Defer(func() x.Observable[op.CallOp] {
		shaper := roboto.Shaper()
		return x.Map(window.Frame(), func(frame vibrant.Frame) op.CallOp {
			h1 := theme.H1
			rect := frame.SafeRect()
			txt := "Hello, World!"
			fill := colornames.Amber200
			ops := new(op.Ops)
			macro := op.Record(ops)
			vibrant.Print(shaper, h1.Font, frame.Px(h1.Size), rect.Dx(), txt, rect, vibrant.Mid, vibrant.Mid, fill, ops)
			return macro.Stop()
		})
	})

	backdrop := vibrant.Backdrop(colornames.Grey600)

	window.Render(backdrop, x.Of(backdrop), drawing).Wait()
	os.Exit(0)
}
