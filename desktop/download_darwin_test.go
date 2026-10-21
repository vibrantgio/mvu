//go:build darwin

package desktop_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/reactivego/rx"

	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/mvu/desktop"
)

// The iCloud Drive folder every Mac signed into iCloud has. A machine
// without it cannot answer anything about downloads, and the test says so
// rather than passing on nothing.
const cloudDocs = "Library/Mobile Documents/com~apple~CloudDocs"

// uploadWait is how long the test waits for the file it wrote to reach the
// cloud. Eviction refuses until the upload has happened — measured on this
// machine, a file is evictable a few seconds after it is written — and the
// test retries rather than sleeping a fixed guess.
const uploadWait = 2 * time.Minute

// downloadWait bounds the download itself. A small file measured 450 ms
// from the ask to being current, so a minute is a failure rather than a
// slow machine.
const downloadWait = 1 * time.Minute

// runDownload subscribes the command the way the loop does and answers the
// one message it emits.
func runDownload(t *testing.T, path string) desktop.Downloaded {
	t.Helper()
	msgs := make(chan mvu.Message, 1)
	sub := desktop.Download(path).Subscribe(rx.GoroutineContext(), func(next mvu.Message, err error, done bool) {
		if done || next == nil {
			return
		}
		select {
		case msgs <- next:
		default:
			t.Error("the download emitted more than one message")
		}
	})
	defer sub.Unsubscribe()
	select {
	case msg := <-msgs:
		got, ok := msg.(desktop.Downloaded)
		if !ok {
			t.Fatalf("the download emitted a %T, want desktop.Downloaded", msg)
		}
		return got
	case <-time.After(downloadWait):
		t.Fatalf("no answer from the download of %s within %s", path, downloadWait)
		return desktop.Downloaded{}
	}
}

// TestADownloadFetchesAnEvictedFile is the whole bridge against the real
// service: a file written into iCloud Drive, evicted so the machine holds
// no contents for it, asked for, and read back. Every piece of it has to be
// here — the folder, brctl, and a file the service accepts and then lets go
// of — and the test says which one is missing rather than passing without
// it.
func TestADownloadFetchesAnEvictedFile(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory on this machine: %v", err)
	}
	root := filepath.Join(home, cloudDocs)
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		t.Skip("this machine has no iCloud Drive folder; a download has nothing to fetch from")
	}
	brctl, err := exec.LookPath("brctl")
	if err != nil {
		t.Skip("brctl is not on this machine, so nothing here can evict a file to download again")
	}

	dir, err := os.MkdirTemp(root, "vgio-download-test-")
	if err != nil {
		t.Skipf("cannot write into iCloud Drive: %v", err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "note.md")
	want := "# A note the cloud holds\n"
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatalf("writing the note: %v", err)
	}

	// Eviction refuses until the service has the file, so the ask is
	// repeated until it is taken or the wait runs out.
	var out []byte
	deadline := time.Now().Add(uploadWait)
	for {
		out, err = exec.Command(brctl, "evict", path).CombinedOutput()
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Skipf("the file never became evictable within %s, so it never reached the cloud: %s", uploadWait, out)
		}
		time.Sleep(time.Second)
	}
	t.Logf("evicted: %s", out)

	start := time.Now()
	got := runDownload(t, path)
	t.Logf("the download answered in %s", time.Since(start).Round(time.Millisecond))

	if got.Path != path {
		t.Errorf("the message carries %s, want the path the download was asked for %s", got.Path, path)
	}
	if got.Err != "" {
		t.Fatalf("the download of an evicted file failed: %s", got.Err)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the download reported the file was here and it is not: %v", err)
	}
	if string(src) != want {
		t.Errorf("the file reads %q, want the contents it was written with %q", src, want)
	}
}

// TestADownloadOfAFileOutsideTheCloudIsRefused holds the other answer the
// platform gives: a path no file provider knows is refused where it is
// asked for, so a caller is told at once rather than watching a file
// nothing is going to write.
func TestADownloadOfAFileOutsideTheCloudIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "local.md")
	if err := os.WriteFile(path, []byte("on this disk and nowhere else\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := runDownload(t, path)
	if got.Err == "" {
		t.Error("a file outside the cloud answered a download with no error")
	}
	if got.Path != path {
		t.Errorf("the refusal carries %s, want %s", got.Path, path)
	}
	t.Logf("the platform refused: %s", got.Err)
}
