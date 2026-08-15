package desktop

import (
	"image"
	"reflect"
	"testing"

	"github.com/vibrantgio/mvu"
)

// The tracker over a fixed two-zone geometry: zone 0 at x∈[0,100), zone 1 at
// x∈[200,300), dead space between and elsewhere. y is ignored by this fake
// resolver; resolver geometry has its own tests in zones_test.go.
func fakeHit(p image.Point) int {
	switch {
	case p.X >= 0 && p.X < 100:
		return 0
	case p.X >= 200 && p.X < 300:
		return 1
	default:
		return -1
	}
}

func TestHoverTracker(t *testing.T) {
	at := func(x int) image.Point { return image.Pt(x, 50) }
	tests := []struct {
		name string
		evs  []dragEvent
		want []mvu.Message
	}{
		{
			"enter zone, hover inside, exit window: enter then exit, no drop",
			[]dragEvent{
				{kind: dragEnter, pos: at(10)},
				{kind: dragMove, pos: at(50)},
				{kind: dragMove, pos: at(90)},
				{kind: dragExit},
			},
			[]mvu.Message{FilesEntered{Zone: 0}, FilesExited{Zone: 0}},
		},
		{
			"cross from zone 0 through dead space into zone 1",
			[]dragEvent{
				{kind: dragEnter, pos: at(50)},
				{kind: dragMove, pos: at(150)},
				{kind: dragMove, pos: at(250)},
			},
			[]mvu.Message{
				FilesEntered{Zone: 0},
				FilesExited{Zone: 0},
				FilesEntered{Zone: 1},
			},
		},
		{
			"direct zone-to-zone crossing exits before entering",
			[]dragEvent{
				{kind: dragEnter, pos: at(99)},
				{kind: dragMove, pos: at(200)},
			},
			[]mvu.Message{
				FilesEntered{Zone: 0},
				FilesExited{Zone: 0},
				FilesEntered{Zone: 1},
			},
		},
		{
			"drop on a zone: exit precedes delivery, drop carries the zone",
			[]dragEvent{
				{kind: dragEnter, pos: at(250)},
				{kind: dragDrop, pos: at(250), paths: []string{"/tmp/a"}},
			},
			[]mvu.Message{
				FilesEntered{Zone: 1},
				FilesExited{Zone: 1},
				FilesDropped{Zone: 1, Paths: []string{"/tmp/a"}, Pos: at(250)},
			},
		},
		{
			"drop in dead space yields nothing — silence by design",
			[]dragEvent{
				{kind: dragEnter, pos: at(150)},
				{kind: dragDrop, pos: at(150), paths: []string{"/tmp/a"}},
			},
			nil,
		},
		{
			"hover entirely in dead space: silence",
			[]dragEvent{
				{kind: dragEnter, pos: at(150)},
				{kind: dragMove, pos: at(160)},
				{kind: dragExit},
			},
			nil,
		},
		{
			"exit without any enter is harmless",
			[]dragEvent{{kind: dragExit}},
			nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := newHoverTracker(fakeHit)
			var got []mvu.Message
			for _, ev := range tc.evs {
				got = append(got, tr.step(ev)...)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("messages = %#v, want %#v", got, tc.want)
			}
		})
	}
}
