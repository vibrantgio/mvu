//go:build darwin

package desktop_test

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/mvu/desktop"
	"github.com/vibrantgio/mvu/desktop/internal/dropsource"
)

// How a drag is driven here, and how far a driven drag reaches.
//
// A drag cannot be synthesised: the platform's drag sessions start from a
// pointer press inside a source application, and synthetic pointer events do
// not produce one. What CAN be built is what a drag presents to a drop target
// — the pasteboard it carries and the dragging info it hands the view — but
// every piece of that is Objective-C, and Go does not support cgo in test
// files ("use of cgo in test ... not supported"). The stand-in source
// therefore lives in an internal package, mvu/desktop/internal/dropsource,
// which nothing outside this module can see; this test opens a real treated
// window, records a zone covering it, and drives real drags over its real
// view.
//
// A promise carries one step further than a stand-in can go. A promise written
// to the drag pasteboard in this very process is read back as a promise
// receiver, and the receiver asks its provider for the file name and then
// never asks for the file: the writing belongs to a live drag session, so the
// bytes of a promised file arrive only from a real drag out of a real
// application. What the window therefore pins here is everything up to that
// hand-off — the view accepts a promise drag, answers its hover with the same
// messages a URL drag gets, accepts its drop, and holds the drop back while
// the promised file is still owed. The counting and assembly that turn the
// received files into the one message are Go's, and drop_darwin_test.go drives
// them directly.
//
// The child process is the same reason the close scenario has one: a window
// needs app.Main on the process's main goroutine and app.Main never returns.
const dropChildEnv = "VGIO_DESKTOP_DROP_CHILD"

// What the drop child's exit statuses mean to the parent.
const (
	dropChildPassed   = 0 // every case landed
	dropChildNoWindow = 2 // no window ever drew: no window server to open one
	dropChildFailed   = 3 // a case failed; the child says which
)

// The scenario's durations. Opening a window is the slow half, as it is for
// the close scenario. A message from a driven drag is one non-blocking channel
// hand-off behind the call, so a second is already generous.
const (
	dropChildFirstFrame = 20 * time.Second
	dropChildFrames     = 150 * time.Millisecond
	dropChildMessage    = 2 * time.Second
	dropChildSilence    = 500 * time.Millisecond
)

// The drag location, in window coordinates (AppKit points, lower-left origin)
// well inside a window of the size below, and so well inside the zone that
// covers it.
const (
	dropChildWindowWidth  = 360
	dropChildWindowHeight = 240
	dropChildAtX          = 120
	dropChildAtY          = 120
)

// The zone the window records, the only one, covering all of it.
const dropChildZone = 0

// dropChild opens a treated window with one zone covering it, drives three
// drags over its view — a file that exists, a file the source only promises,
// and both at once — and reports through its exit status.
func dropChild() {
	w := mvu.NewWindow(append(desktop.FullSizeContent(),
		app.Title("desktop drop test"),
		app.Size(unit.Dp(dropChildWindowWidth), unit.Dp(dropChildWindowHeight)),
	)...)
	desktop.ShowWindowButtons(w)

	zones := &desktop.ZoneGroup{}
	drops := desktop.NewDropTarget(w, zones)

	msgs := make(chan mvu.Message, 8)
	drops.Messages().Subscribe(rx.GoroutineContext(), func(m mvu.Message, err error, done bool) {
		if !done {
			msgs <- m
		}
	})

	var zone layout.Widget = func(gtx layout.Context) layout.Dimensions {
		zones.Update(gtx)
		zones.Record(dropChildZone, image.Rectangle{Max: gtx.Constraints.Max})
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}

	drawn := make(chan struct{})
	var once sync.Once
	w.OnConfigure(func() { once.Do(func() { close(drawn) }) })

	ended := make(chan error, 1)
	go func() { ended <- w.Render(rx.Of(zone)).Wait() }()

	select {
	case <-drawn:
	case err := <-ended:
		fmt.Printf("the run ended before the window drew: %v\n", err)
		os.Exit(dropChildNoWindow)
	case <-time.After(dropChildFirstFrame):
		fmt.Println("no first frame")
		os.Exit(dropChildNoWindow)
	}

	// A zone is resolved against the last COMPLETED frame, so the zone
	// recorded during the first frame is live only once a second one has run.
	for range 3 {
		w.Window().Invalidate()
		time.Sleep(dropChildFrames)
	}

	existing := dropChildFile()
	promise := &dropsource.Promise{
		Name:  "promised.txt",
		Bytes: []byte("the bytes the promise would write"),
	}

	// A file that exists arrives as it did before promises were accepted at
	// all: the hover enters the zone, and the drop delivers the path.
	dropChildBegin("a file that exists", w, existing, nil)
	dropChildHover("a file that exists", w, msgs)
	dropChildDrop("a file that exists", w)
	dropChildWants("a file that exists", msgs, desktop.FilesExited{Zone: dropChildZone})
	dropChildDropped("a file that exists", msgs, []string{existing})

	// A promise drag is accepted where it used to be refused at the window
	// edge: the hover answers with the same pair of messages, and the drop is
	// taken and ends the hover at once even though its file is still owed. The
	// file itself is beyond a stand-in, so nothing waits for it.
	const promised = "a file the source promises"
	dropChildBegin(promised, w, "", promise)
	dropChildHover(promised, w, msgs)
	dropChildExit(promised, w)
	dropChildWants(promised, msgs, desktop.FilesExited{Zone: dropChildZone})
	dropChildHover(promised, w, msgs)
	dropChildDrop(promised, w)
	dropChildWants(promised, msgs, desktop.FilesExited{Zone: dropChildZone})

	// A drag carrying both is one drop that waits: the plain URL is held back
	// with the promised file rather than delivered ahead of it.
	dropChildBegin("both at once", w, existing, promise)
	dropChildHover("both at once", w, msgs)
	dropChildDrop("both at once", w)
	dropChildWants("both at once", msgs, desktop.FilesExited{Zone: dropChildZone})
	dropChildStaysSilent("both at once", msgs)

	fmt.Println("every drag was accepted and answered on the window's stream")
	os.Exit(dropChildPassed)
}

// dropChildBegin puts the drag's files on the drag pasteboard.
func dropChildBegin(what string, w *mvu.Window, url string, promise *dropsource.Promise) {
	var err error
	w.Window().Run(func() { err = dropsource.Begin(url, promise) })
	if err != nil {
		fmt.Printf("%s: the drag could not be built: %v\n", what, err)
		os.Exit(dropChildFailed)
	}
}

// dropChildHover drives draggingEntered: and waits for the entered message,
// which is the half that used to be refused for a promise drag.
func dropChildHover(what string, w *mvu.Window, msgs chan mvu.Message) {
	var answered bool
	var err error
	w.Window().Run(func() { answered, err = dropsource.Enter(dropChildAtX, dropChildAtY) })
	if err != nil {
		fmt.Printf("%s: the hover could not be driven: %v\n", what, err)
		os.Exit(dropChildFailed)
	}
	if !answered {
		fmt.Printf("%s: the view refused the drag at the window edge\n", what)
		os.Exit(dropChildFailed)
	}
	dropChildWants(what, msgs, desktop.FilesEntered{Zone: dropChildZone})
}

// dropChildExit drives draggingExited:.
func dropChildExit(what string, w *mvu.Window) {
	var err error
	w.Window().Run(func() { err = dropsource.Exit() })
	if err != nil {
		fmt.Printf("%s: the exit could not be driven: %v\n", what, err)
		os.Exit(dropChildFailed)
	}
}

// dropChildDrop drives performDragOperation: and requires the view to accept
// the drop.
func dropChildDrop(what string, w *mvu.Window) {
	var accepted bool
	var err error
	w.Window().Run(func() { accepted, err = dropsource.Drop(dropChildAtX, dropChildAtY) })
	if err != nil {
		fmt.Printf("%s: the drop could not be driven: %v\n", what, err)
		os.Exit(dropChildFailed)
	}
	if !accepted {
		fmt.Printf("%s: the view refused the drop\n", what)
		os.Exit(dropChildFailed)
	}
}

// dropChildWants takes the next message and requires it to be the one given.
func dropChildWants(what string, msgs chan mvu.Message, want mvu.Message) {
	select {
	case got := <-msgs:
		if got != want {
			fmt.Printf("%s: got %#v, want %#v\n", what, got, want)
			os.Exit(dropChildFailed)
		}
	case <-time.After(dropChildMessage):
		fmt.Printf("%s: no %T arrived\n", what, want)
		os.Exit(dropChildFailed)
	}
}

// dropChildDropped takes the next message and requires it to be the drop,
// carrying the given paths in the given order, inside the window's zone.
func dropChildDropped(what string, msgs chan mvu.Message, want []string) {
	var got desktop.FilesDropped
	select {
	case m := <-msgs:
		dropped, ok := m.(desktop.FilesDropped)
		if !ok {
			fmt.Printf("%s: got %T, want desktop.FilesDropped\n", what, m)
			os.Exit(dropChildFailed)
		}
		got = dropped
	case <-time.After(dropChildMessage):
		fmt.Printf("%s: no drop arrived\n", what)
		os.Exit(dropChildFailed)
	}
	if got.Zone != dropChildZone {
		fmt.Printf("%s: the drop landed in zone %d, want %d\n", what, got.Zone, dropChildZone)
		os.Exit(dropChildFailed)
	}
	if len(got.Paths) != len(want) {
		fmt.Printf("%s: got %d path(s) %v, want %d\n", what, len(got.Paths), got.Paths, len(want))
		os.Exit(dropChildFailed)
	}
	for i := range want {
		if got.Paths[i] != want[i] {
			fmt.Printf("%s: path %d is %q, want %q\n", what, i, got.Paths[i], want[i])
			os.Exit(dropChildFailed)
		}
	}
}

// dropChildStaysSilent requires that nothing follows: the drop is still waiting
// for a promised file, so neither it nor the plain URL it also carried has
// been delivered.
func dropChildStaysSilent(what string, msgs chan mvu.Message) {
	select {
	case m := <-msgs:
		if _, ok := m.(desktop.FilesDropped); ok {
			fmt.Printf("%s: the drop landed with a promised file still owed: %#v\n", what, m)
			os.Exit(dropChildFailed)
		}
	case <-time.After(dropChildSilence):
	}
}

// dropChildFile writes the file the plain-URL cases drag and returns its path.
func dropChildFile() string {
	dir, err := os.MkdirTemp("", "mvu-desktop-drop-test")
	if err != nil {
		fmt.Printf("no temporary directory: %v\n", err)
		os.Exit(dropChildFailed)
	}
	path := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(path, []byte("the bytes of a file that already exists"), 0o600); err != nil {
		fmt.Printf("the file to drag could not be written: %v\n", err)
		os.Exit(dropChildFailed)
	}
	return path
}

// A drag carrying a file promise is accepted and answered on a real window —
// its hover delivers the entered and exited pair, its drop is taken, and the
// drop waits for the promised file rather than landing without it — while a
// drag carrying a plain file URL still lands as it did, with its path.
func TestAPromiseDragIsAcceptedOnAWindow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(),
		dropChildFirstFrame+10*dropChildMessage+30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), dropChildEnv+"=1")
	out, err := cmd.CombinedOutput()
	t.Logf("child: %s", out)
	if err == nil {
		return
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("running the child failed: %v", err)
	}
	if exit.ExitCode() == dropChildNoWindow {
		t.Skip("no window drew: nothing here can be verified without a window server")
	}
	t.Fatalf("the child exited %d, want %d", exit.ExitCode(), dropChildPassed)
}
