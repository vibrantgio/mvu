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
	go IconList()
	app.Main()
}

func IconList() {
	window := mvu.NewWindow(app.Title("MVU - Icon List"))
	backdrop := rx.Of(gio.Backdrop(colornames.Grey600))

	icons := rx.Map(rx.Timer[int](0, time.Second), func(i int) []byte {
		return [...][]byte{icons.ActionEuroSymbol, icons.AVArtTrack, icons.ActionAlarm}[i%3]
	})

	widgets := rx.Defer(func() rx.Observable[[]layout.Widget] {
		widgets := []layout.Widget{}
		return rx.Map(icons, func(data []byte) []layout.Widget {
			icon := rx.Must(raster.Widget(data, 240, 240, raster.WithColors(colornames.Orange400)))
			widgets = append(widgets, icon)
			return widgets
		})
	})

	content := rx.Defer(func() rx.Observable[layout.Widget] {
		list := layout.List{
			Axis:      layout.Vertical,
			Alignment: layout.Start,
		}
		return rx.Map(widgets, func(widgets []layout.Widget) layout.Widget {
			return func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return list.Layout(gtx, len(widgets), func(gtx layout.Context, i int) layout.Dimensions {
						return widgets[i](gtx)
					})
				})
			}
		})
	})

	window.Render(backdrop, content).Wait()
	os.Exit(0)
}
