package main

import (
	"image"
	"image/color"
	"os"
	"time"

	"golang.org/x/exp/shiny/materialdesign/colornames"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/backdrop"
	"github.com/vibrantgio/mvu"
)

func main() {
	go Rectangles()
	app.Main()
}

func Rectangles() {
	window := mvu.NewWindow(app.Title("MVU - Rectangles"))

	backdrop := rx.Of(backdrop.Widget(colornames.Grey600))

	fixed := rx.Of(image.Rect(100, 100, 200, 200))
	yellow := RRect(fixed, 5, colornames.Yellow700)

	moving := rx.Map(rx.Interval[int](time.Second).StartWith(-1.0), func(i int) image.Rectangle {
		return image.Rect(100+50*i, 300, 200+50*i, 400)
	})
	purple := RRect(moving, 5, colornames.DeepPurple700)

	window.Render(backdrop, yellow, purple).Wait()
	os.Exit(0)
}

func RRect(rects rx.Observable[image.Rectangle], radius int, fill color.Color) rx.Observable[layout.Widget] {
	return rx.Map(rects, func(r image.Rectangle) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			fill := color.NRGBAModel.Convert(fill).(color.NRGBA)
			shape := clip.UniformRRect(r, radius).Op(gtx.Ops)
			paint.FillShape(gtx.Ops, fill, shape)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}
	})
}
