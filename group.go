package vibrant

import (
	"gioui.org/layout"

	"github.com/reactivego/x"
)

// Group combines multiple layout.Widget streams into a single stream of layout.Widget.
func Group(layers ...x.Observable[layout.Widget]) x.Observable[layout.Widget] {
	return x.Map(x.Combine(layers...), func(widgets []layout.Widget) layout.Widget {
		if len(widgets) == 1 {
			return widgets[0]
		}
		return func(gtx layout.Context) layout.Dimensions {
			for _, widget := range widgets {
				widget(gtx)
			}
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}
	})
}
