package main

import (
	"os"

	"golang.org/x/exp/shiny/materialdesign/colornames"

	"gioui.org/app"
	"gioui.org/text"

	"github.com/reactivego/gio"
	"github.com/reactivego/gio/style"
	"github.com/reactivego/mvu"
	"github.com/reactivego/rx"
)

func main() {
	go Hello()
	app.Main()
}

func Hello() {
	window := mvu.NewWindow(app.Title("MVU - Hello"))
	backdrop := rx.Of(gio.Backdrop(colornames.Grey600))
	shaper := text.NewShaper(style.FontFaces())
	content := rx.Of(gio.Text(shaper, style.H1, 0.5, 0.5, colornames.LightBlue100, "Hello, World!"))
	window.Render(backdrop, content).Wait()
	os.Exit(0)
}
