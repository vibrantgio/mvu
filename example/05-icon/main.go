package main

import (
	"os"

	"golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"gioui.org/app"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/backdrop"
	ivg "github.com/vibrantgio/ivg/raster/gio"
	"github.com/vibrantgio/mvu"
)

func main() {
	go Icon()
	app.Main()
}

func Icon() {
	window := mvu.NewWindow(app.Title("MVU - Icon"))
	backdrop := rx.Of(backdrop.Widget(colornames.Grey600))
	icon := rx.Of(rx.Must(ivg.Widget(icons.ActionAlarm, 48, 48, ivg.WithColors(colornames.Orange400))))
	window.Render(backdrop, icon).Wait()
	os.Exit(0)
}
