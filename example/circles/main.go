package main

import (
	"math"
	"os"

	"golang.org/x/exp/shiny/materialdesign/colornames"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/backdrop"
	vcircle "github.com/vibrantgio/circle"
	"github.com/vibrantgio/mvu"
)

func main() {
	go Circles()
	app.Main()
}

func Circles() {
	window := mvu.NewWindow(app.Title("MVU - Circles"))

	backdrops := rx.Of(backdrop.Widget(colornames.Grey600))

	type circle struct {
		Center f32.Point
		Radius float32
	}

	drawing := rx.Defer(func() rx.Observable[layout.Widget] {
		circles := []circle(nil)
		return rx.Of[layout.Widget](func(gtx layout.Context) layout.Dimensions {
			size := gtx.Constraints.Max
			defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
			event.Op(gtx.Ops, &circles)
			for {
				e, ok := gtx.Source.Event(pointer.Filter{
					Target: &circles,
					Kinds:  pointer.Press | pointer.Drag | pointer.Release,
				})
				if !ok {
					break
				}
				if e, ok := e.(pointer.Event); ok {
					var delta f32.Point
					switch e.Kind {
					case pointer.Press:
						circles = append(circles, circle{Center: e.Position})
						fallthrough
					case pointer.Drag, pointer.Release:
						c := circles[len(circles)-1]
						delta = e.Position.Sub(c.Center)
						c.Radius = float32(math.Sqrt(float64(delta.X*delta.X + delta.Y*delta.Y)))
						circles[len(circles)-1] = c
					}
				}
			}
			for _, c := range circles {
				vcircle.FillCircle(gtx.Ops, c.Center, c.Radius, colornames.Yellow800)
			}
			return layout.Dimensions{Size: size}
		})
	})

	window.Render(backdrops, drawing).Wait()
	os.Exit(0)
}
