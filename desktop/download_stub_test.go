//go:build !darwin

package desktop

import (
	"testing"
	"time"
)

// Away from macOS a download answers that nothing is in the cloud, at
// once: there is no platform here to ask, so a caller that showed a
// waiting state gets the message that takes it down on the very next turn
// of the loop rather than after a wait that could never end.
func TestADownloadOffMacOSSaysNothingIsInTheCloud(t *testing.T) {
	answered := make(chan Downloaded, 1)
	go func() { answered <- download("/some/file.md", time.Second, time.Minute) }()

	select {
	case got := <-answered:
		if got.Err == "" {
			t.Error("the stub answered a download with no error; no file here is in a cloud")
		}
		if got.Path != "/some/file.md" {
			t.Errorf("the answer carries %s, want the path it was asked for", got.Path)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the stub blocked; it must answer without waiting on anything")
	}
}
