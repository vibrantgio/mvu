package main

import (
	"os"

	"golang.org/x/exp/shiny/materialdesign/colornames"

	"gioui.org/app"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/backdrop"
	"github.com/vibrantgio/mvu"
)

func main() {
	go Backdrop()
	app.Main()
}

func Backdrop() {
	window := mvu.NewWindow(app.Title("MVU - Backdrop"))
	backdrop := rx.Of(backdrop.Widget(colornames.Grey600))
	window.Render(backdrop).Wait()
	os.Exit(0)
}
