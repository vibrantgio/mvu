package main

import (
	"os"

	"golang.org/x/exp/shiny/materialdesign/colornames"

	"gioui.org/app"

	"github.com/reactivego/gio"
	"github.com/reactivego/mvu"
	"github.com/reactivego/rx"
)

func main() {
	go Backdrop()
	app.Main()
}

func Backdrop() {
	window := mvu.NewWindow(app.Title("MVU - Backdrop"))
	backdrop := rx.Of(gio.Backdrop(colornames.Grey600))
	window.Render(backdrop).Wait()
	os.Exit(0)
}
