package main

import (
	"image"
	"log"
	"os"

	"golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/backdrop"
	raster "github.com/vibrantgio/ivg/raster/gio"
	"github.com/vibrantgio/mvu"
)

func main() {
	go Minimal()
	app.Main()
}

func Minimal() {
	window := mvu.NewWindow(app.Title("MVU - Icon Scroll"))
	backdrops := rx.Of(backdrop.Widget(colornames.Grey600))

	icon, err := raster.Widget(icons.ActionAlarm, 48, 48, raster.WithColors(colornames.Amber600))
	if err != nil {
		log.Fatal(err)
	}

	contents := rx.Defer(func() rx.Observable[layout.Widget] {
		var offset struct{ X, Y unit.Dp }
		widget := func(gtx layout.Context) layout.Dimensions {
			max := gtx.Constraints.Max

			cs := clip.Rect{Max: max}.Op().Push(gtx.Ops)
			event.Op(gtx.Ops, &offset)
			cs.Pop()
			for {
				e, ok := gtx.Source.Event(pointer.Filter{
					Target:  &offset,
					Kinds:   pointer.Scroll,
					ScrollX: pointer.ScrollRange{Min: -100, Max: 100},
					ScrollY: pointer.ScrollRange{Min: -100, Max: 100},
				})
				if !ok {
					break
				}
				if e, ok := e.(pointer.Event); ok {
					offset.X += gtx.Metric.PxToDp(int(e.Scroll.X))
					offset.Y += gtx.Metric.PxToDp(int(e.Scroll.Y))
				}
			}
			offset := image.Pt(gtx.Dp(offset.X), gtx.Dp(offset.Y))
			pos := gtx.Constraints.Max.Div(2).Sub(offset)

			ts := op.Offset(pos.Sub(image.Pt(gtx.Dp(50), gtx.Dp(50)))).Push(gtx.Ops)
			gtx.Constraints.Max = image.Pt(gtx.Dp(100), gtx.Dp(100))
			icon(gtx)
			ts.Pop()

			return layout.Dimensions{Size: max}
		}
		return rx.Of[layout.Widget](widget)
	})

	window.Render(backdrops, contents).Wait()
	os.Exit(0)
}
