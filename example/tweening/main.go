package main

import (
	"image/color"
	"math"
	"os"
	"time"

	"golang.org/x/exp/shiny/materialdesign/colornames"

	"gioui.org/app"
	"gioui.org/layout"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/backdrop"
	"github.com/vibrantgio/mvu"

	"github.com/fogleman/ease"
)

func main() {
	go Tweening()
	app.Main()
}

func Tweening() {
	window := mvu.NewWindow(app.Title("MVU - Tweening"))

	colors := []color.RGBA{
		colornames.Grey900,
		colornames.Grey600,
		colornames.Grey300,
		colornames.Grey100,
		colornames.Grey300,
		colornames.Grey600,
		colornames.Grey900,
	}
	stops := rx.Map(rx.Timer[int](0, time.Second), func(next int) []color.RGBA {
		i := next % (len(colors) - 1)
		return colors[i : i+2]
	})
	tweened := TweenColors(stops, 60, 900*time.Millisecond, ease.InOutQuad)
	backdrop := rx.Map(tweened, func(fill color.RGBA) layout.Widget {
		return backdrop.Widget(fill)
	})

	window.Render(backdrop).Wait()
	os.Exit(0)
}

func TweenColors(colors rx.Observable[[]color.RGBA], fps int64, duration time.Duration, ease ease.Function) rx.Observable[color.RGBA] {
	return rx.SwitchMap(colors, func(pair []color.RGBA) rx.Observable[color.RGBA] {
		// fmt.Printf("tweening %#v\n", pair)

		rgbaf := func(r, g, b, a uint32) (R, G, B, A float64) {
			R = float64(r) / float64(a)
			G = float64(g) / float64(a)
			B = float64(b) / float64(a)
			A = float64(a) / 0xFFFF
			return
		}
		// linearTosRGB transforms color value from linear to sRGB.
		linearTosRGB := func(c float64) float64 {
			// Formula from EXT_sRGB.
			switch {
			case c <= 0:
				return 0
			case 0 < c && c < 0.0031308:
				return 12.92 * c
			case 0.0031308 <= c && c < 1:
				return 1.055*math.Pow(c, 0.41666) - 0.055
			}
			return 1
		}
		// sRGBToLinear transforms color value from sRGB to linear.
		sRGBToLinear := func(c float64) float64 {
			// Formula from EXT_sRGB.
			if c <= 0.04045 {
				return c / 12.92
			} else {
				return math.Pow((c+0.055)/1.055, 2.4)
			}
		}
		r0, g0, b0, _ := rgbaf(pair[0].RGBA())
		r1, g1, b1, _ := rgbaf(pair[1].RGBA())
		return rx.Map(rx.Map(Duration(fps, duration), ease), func(t float64) color.RGBA {
			// translucent color 1 over translucent color 0
			// a = (1 - a1)*a0 + a1
			// r = ((1 - a1)*a0*r0 + a1*r1) / a
			// g = ((1 - a1)*a0*g0 + a1*g1) / a
			// b = ((1 - a1)*a0*b0 + a1*b1) / a

			// Blend between opaque colors
			r := linearTosRGB(sRGBToLinear(r0)*(1.0-t) + sRGBToLinear(r1)*t)
			g := linearTosRGB(sRGBToLinear(g0)*(1.0-t) + sRGBToLinear(g1)*t)
			b := linearTosRGB(sRGBToLinear(b0)*(1.0-t) + sRGBToLinear(b1)*t)
			a := 1.0
			return color.RGBAModel.Convert(color.NRGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), uint8(a * 255)}).(color.RGBA)
		})
	})
}

func Duration(fps int64, duration time.Duration) rx.Observable[float64] {
	return Defer(func(scheduler rx.Scheduler) rx.Observable[float64] {
		start := scheduler.Now()
		return rx.Map(rx.Ticker(0, time.Second/time.Duration(fps)), func(now time.Time) float64 {
			return float64(now.Sub(start)) / float64(duration)
		}).TakeWhile(func(t float64) bool {
			return t <= 1
		})
	})
}

func Defer[T any](factory func(rx.Scheduler) rx.Observable[T]) rx.Observable[T] {
	return func(observe rx.Observer[T], scheduler rx.Scheduler, subscriber rx.Subscriber) {
		factory(scheduler)(observe, scheduler, subscriber)
	}
}
