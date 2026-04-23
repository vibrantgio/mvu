package main

import (
	"os"

	"golang.org/x/exp/shiny/materialdesign/colornames"

	"gioui.org/app"
	"gioui.org/f32"

	"github.com/reactivego/gio"
	"github.com/reactivego/mvu"
	"github.com/reactivego/rx"
)

func main() {
	go Gradient()
	app.Main()
}

func Gradient() {
	window := mvu.NewWindow(app.Title("MVU - Gradient"))
	gradient := rx.Of(gio.LinearGradient(f32.Pt(0, 0), colornames.DeepPurple800, f32.Pt(1, 1), colornames.DeepPurple300))
	window.Render(gradient).Wait()
	os.Exit(0)
}
