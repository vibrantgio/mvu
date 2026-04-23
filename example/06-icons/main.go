package main

import (
	"os"
	"time"

	"golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"gioui.org/app"
	"gioui.org/layout"

	"github.com/reactivego/gio"
	raster "github.com/vibrantgio/ivg/raster/gio"
	"github.com/reactivego/mvu"
	"github.com/reactivego/rx"
)

func main() {
	go Icons()
	app.Main()
}

func Icons() {
	window := mvu.NewWindow(app.Title("MVU - Icons"))
	backdrop := rx.Of(gio.Backdrop(colornames.Grey600))

	icons := rx.Map(rx.Timer[int](0, time.Second), func(i int) []byte {
		return [...][]byte{icons.ActionEuroSymbol, icons.AVArtTrack, icons.ActionAlarm}[i%3]
	})

	content := rx.Map(icons, func(data []byte) layout.Widget {
		icon := rx.Must(raster.Widget(data, 48, 48, raster.WithColors(colornames.Orange400)))
		return func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(24).Layout(gtx, icon)
		}
	})

	window.Render(backdrop, content).Wait()
	os.Exit(0)
}
