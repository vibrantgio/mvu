package main

import (
	"os"

	"golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"gioui.org/app"

	"github.com/reactivego/gio"
	raster "github.com/vibrantgio/ivg/raster/gio"
	"github.com/reactivego/mvu"
	"github.com/reactivego/rx"
)

func main() {
	go Icon()
	app.Main()
}

func Icon() {
	window := mvu.NewWindow(app.Title("MVU - Icon"))
	backdrop := rx.Of(gio.Backdrop(colornames.Grey600))
	icon := rx.Of(rx.Must(raster.Widget(icons.ActionAlarm, 48, 48, raster.WithColors(colornames.Orange400))))
	window.Render(backdrop, icon).Wait()
	os.Exit(0)
}
