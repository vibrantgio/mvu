package vibrant

import (
	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
)

// BezierCircle returns a clip.PathSpec for a circle constructed from Cubic Bezier segments.
// Note that originally the bezier circle const was 0.55228475 equal to 4*(sqrt(2)-1)/3
// A better bezier circle const to use is 0.551915024494 according to the following article:
// http://spencermortensen.com/articles/bezier-circle/
func Circle(p f32.Point, r float32, ops *op.Ops) clip.PathSpec {
	const c = 0.551915024494
	east := f32.Point{X: r, Y: 0}
	sw := f32.Point{X: -r, Y: r}
	nw := f32.Point{X: -r, Y: -r}
	ne := f32.Point{X: r, Y: -r}
	se := f32.Point{X: r, Y: r}
	west := f32.Point{X: -r, Y: 0}
	path := &clip.Path{}
	path.Begin(ops)
	path.Move(p)
	path.Move(east)
	path.Cube(f32.Point{X: 0, Y: r * c}, f32.Point{X: (c - 1) * r, Y: r}, sw)
	path.Cube(f32.Point{X: -(r * c), Y: 0}, f32.Point{X: -r, Y: (c - 1) * r}, nw)
	path.Cube(f32.Point{X: 0, Y: -(r * c)}, f32.Point{X: (1 - c) * r, Y: -r}, ne)
	path.Cube(f32.Point{X: r * c, Y: 0}, f32.Point{X: r, Y: (1 - c) * r}, se)
	path.Close()
	path.Move(west)
	return path.End()
}
