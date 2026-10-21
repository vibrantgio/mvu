package desktop

import (
	"time"

	"github.com/vibrantgio/mvu"
)

// downloadPoll is how often the watch re-reads a download's state. The
// platform publishes no notification this package can wait on, so the
// state is read at an interval. A small file evicted from iCloud Drive and
// asked for again was current 450 ms later (macOS 26, this package's own
// darwin test), so a tenth of a second resolves the wait closely enough
// and costs one resource-value read per interval.
const downloadPoll = 100 * time.Millisecond

// downloadDeadline bounds the watch so the command always answers. A
// download the platform never finishes — a machine off the network, an
// item the service has given up on — would otherwise leave a window
// saying it is fetching a file for the rest of the run.
const downloadDeadline = 5 * time.Minute

// Download asks the platform to fetch the file at path — an absolute
// path — out of the cloud, and answers exactly one [Downloaded] when the
// file is at that path or the download failed. Whether the item is in the
// cloud at all is the platform's to decide, not this package's: the ask is
// made and a path the platform keeps no cloud item for is refused by the
// platform, which is what the message then carries.
//
//	case OpenNote:
//		return model, desktop.Download(full)
//	case desktop.Downloaded:
//		if m.Err != "" { ... }
//
// The command does its waiting on the goroutine the loop subscribes it on,
// so nothing of this reaches the window's own thread. Away from macOS it
// answers at once that nothing is in the cloud.
func Download(path string) mvu.Command {
	return mvu.Do(func() (mvu.Message, error) {
		return download(path, downloadPoll, downloadDeadline), nil
	})
}
