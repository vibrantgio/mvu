package main

import (
	"os"

	"golang.org/x/exp/shiny/materialdesign/colornames"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/backdrop"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/textdraw"
)

func main() {
	go Hello()
	app.Main()
}

func Hello() {
	window := mvu.NewWindow(app.Title("MVU - Hello"))
	backdrop := rx.Of(backdrop.Widget(colornames.Grey600))
	typ := tokens.DefaultTypography
	content := rx.Of(textdraw.Text(typ.Shaper(), textStyle(typ.DisplayLarge), 0.5, 0.5, colornames.LightBlue100, "Hello, World!"))
	window.Render(backdrop, content).Wait()
	os.Exit(0)
}

// textStyle converts one Typography role to a single-line textdraw style.
func textStyle(ts tokens.TextStyle) textdraw.TextStyle {
	f := font.Font{Typeface: font.Typeface(ts.Typeface)}
	if ts.Weight != 0 {
		f.Weight = tokens.FontWeight(ts.Weight)
	}
	return textdraw.TextStyle{Font: f, Alignment: textdraw.Start, Size: unit.Sp(ts.Size), MaxLines: 1, Truncator: "…"}
}
