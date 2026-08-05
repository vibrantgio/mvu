package main

import (
	"image/color"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/x/richtext"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/spectrum/tokens"
)

func main() {
	go RichText()
	app.Main()
}

func RichText() {
	window := mvu.NewWindow(app.Title("MVU - Rich Text"))
	window.Render(rx.Of(Content())).Wait()
	os.Exit(0)
}

func Content() layout.Widget {
	black := color.NRGBA{A: 255}
	green := color.NRGBA{G: 170, A: 255}
	blue := color.NRGBA{B: 170, A: 255}
	red := color.NRGBA{R: 170, A: 255}

	// The theme's Typography supplies the shaper and the typeface: the spans
	// name Roboto by face and keep their own per-span sizes.
	typ := tokens.DefaultTypography
	roboto := font.Font{Typeface: font.Typeface(typ.BodyLarge.Typeface)}

	// define the text that you want to present. This can be persisted
	// across frames, recomputed every frame, or modified in any way between
	// frames.
	spans := []richtext.SpanStyle{
		{
			Content: "Hello ",
			Color:   black,
			Size:    unit.Sp(24),
			Font:    roboto,
		},
		{
			Content: "in ",
			Color:   green,
			Size:    unit.Sp(36),
			Font:    roboto,
		},
		{
			Content: "rich ",
			Color:   blue,
			Size:    unit.Sp(30),
			Font:    roboto,
		},
		{
			Content: "text\n",
			Color:   red,
			Size:    unit.Sp(40),
			Font:    roboto,
		},
		{
			Content:     "Interact with me!",
			Color:       black,
			Size:        unit.Sp(40),
			Font:        roboto,
			Interactive: true,
		},
	}
	state := richtext.InteractiveText{}
	index := 0
	shaper := typ.Shaper()

	return func(gtx layout.Context) layout.Dimensions {

		// change the spans to reflect interaction
		swatch := []color.NRGBA{black, green, blue, red}
		spans[4].Color = swatch[index%len(swatch)]

		// process any interactions with the text since the last frame.
		for {
			span, event, ok := state.Update(gtx)
			if !ok {
				break
			}
			content, _ := span.Content()
			switch event.Type {
			case richtext.Click:
				log.Println(event.ClickData.Kind)
				if event.ClickData.Kind == gesture.KindClick {
					index++
					gtx.Execute(op.InvalidateCmd{})
				}
			case richtext.Hover:
				log.Println("Hovered: " + content)
			case richtext.LongPress:
				log.Println("Long-pressed: " + content)
			}
		}

		return richtext.Text(&state, shaper, spans...).Layout(gtx)
	}
}
