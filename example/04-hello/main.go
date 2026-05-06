package main

import (
	"os"

	"golang.org/x/exp/shiny/materialdesign/colornames"

	"gioui.org/app"
	"gioui.org/text"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/backdrop"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/style"
	"github.com/vibrantgio/textdraw"
)

func main() {
	go Hello()
	app.Main()
}

func Hello() {
	window := mvu.NewWindow(app.Title("MVU - Hello"))
	backdrop := rx.Of(backdrop.Widget(colornames.Grey600))
	shaper := text.NewShaper(text.WithCollection(style.FontFaces()))
	content := rx.Of(textdraw.Text(shaper, style.H1, 0.5, 0.5, colornames.LightBlue100, "Hello, World!"))
	window.Render(backdrop, content).Wait()
	os.Exit(0)
}
