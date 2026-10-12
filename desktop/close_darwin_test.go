//go:build darwin

package desktop_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"gioui.org/app"
	"gioui.org/unit"

	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/mvu/desktop"
)

// Why this scenario runs in a child process. A window needs app.Main on the
// process's main goroutine, and app.Main never returns — so a test binary
// that opens one has given its main goroutine away for good, and every other
// test in the package would run with an NSApplication behind it, which is the
// very condition the rest of them are written to document the absence of. The
// child is this same binary re-executed with closeChildEnv set: there
// TestMain hands the main goroutine to app.Main and the scenario runs as a
// plain goroutine, reporting through the exit status.
const closeChildEnv = "VGIO_DESKTOP_CLOSE_CHILD"

// What the child's exit statuses mean to the parent.
const (
	closeChildEnded    = 0 // the run ended with the destroy event
	closeChildNoWindow = 2 // no window ever drew: no window server to open one
	closeChildStuck    = 3 // the window closed and the run stayed in its loop
)

// The scenario's three durations.
//
// Opening a window is the slow half — a cold process, a GPU surface, the
// first frame — where the close is one notification and one main-queue turn,
// so a close that has not landed within a second of its own bound has not
// landed at all.
//
// The settle is the one that carries a measurement. A window closed within
// the first ~100 ms of its first frame is deallocated by its own close on
// macOS 26, which raises Gio's destroy event all by itself; from ~200 ms on
// it is not, and the run then stays in its event loop with nothing on screen
// unless the close drops the window's hold on the view. The scenario waits
// ten times that boundary before asking, so it exercises the case an
// application is actually in.
const (
	closeChildFirstFrame = 20 * time.Second
	closeChildSettle     = 1 * time.Second
	closeChildDestroy    = 5 * time.Second
)

// TestMain routes the package's window scenarios. A scenario runs in a child
// process this binary re-executes with its own variable set: there the main
// goroutine goes to app.Main and the scenario runs as a plain goroutine,
// reporting through the exit status. Without a variable set, this is the
// ordinary test run.
func TestMain(m *testing.M) {
	switch {
	case os.Getenv(closeChildEnv) != "":
		go closeChild()
	case os.Getenv(dropChildEnv) != "":
		go dropChild()
	default:
		os.Exit(m.Run())
	}
	app.Main()
}

// closeChild opens a treated window, asks the package to close it, and
// reports through its exit status whether the run ended.
func closeChild() {
	w := mvu.NewWindow(append(desktop.FullSizeContent(),
		app.Title("desktop close test"),
		app.Size(unit.Dp(360), unit.Dp(200)),
	)...)
	desktop.ShowWindowButtons(w)

	// The configuration notification fires once the window has delivered its
	// first frame, which is the first moment there is a native window for a
	// close to find.
	drawn := make(chan struct{})
	var once sync.Once
	w.OnConfigure(func() { once.Do(func() { close(drawn) }) })

	ended := make(chan error, 1)
	go func() { ended <- w.Render().Wait() }()

	select {
	case <-drawn:
	case err := <-ended:
		fmt.Printf("the run ended before the window drew: %v\n", err)
		os.Exit(closeChildNoWindow)
	case <-time.After(closeChildFirstFrame):
		fmt.Println("no first frame")
		os.Exit(closeChildNoWindow)
	}
	time.Sleep(closeChildSettle)

	desktop.Close(w)

	select {
	case err := <-ended:
		if err != nil {
			fmt.Printf("the run ended with an error: %v\n", err)
			os.Exit(closeChildStuck)
		}
		fmt.Println("the run ended with the destroy event")
		os.Exit(closeChildEnded)
	case <-time.After(closeChildDestroy):
		fmt.Println("the window closed and the run is still in its event loop")
		os.Exit(closeChildStuck)
	}
}

// The close the application asks for ends the run, on a real window. mvu's
// Render returns when and only when the window emits app.DestroyEvent, so a
// Render that has returned without an error is that event having arrived —
// and the event is the whole of it: AppKit closes the window either way, and
// before this the run went on without one.
func TestTheCloseEndsTheRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(),
		closeChildFirstFrame+closeChildSettle+closeChildDestroy+15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), closeChildEnv+"=1")
	out, err := cmd.CombinedOutput()
	t.Logf("child: %s", out)
	if err == nil {
		return
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("running the child failed: %v", err)
	}
	if exit.ExitCode() == closeChildNoWindow {
		t.Skip("no window drew: nothing here can be verified without a window server")
	}
	t.Fatalf("the child exited %d, want %d — desktop.Close did not end the run",
		exit.ExitCode(), closeChildEnded)
}
