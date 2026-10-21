//go:build !darwin

package desktop

import "time"

// Away from macOS this package knows of no cloud a file could be waiting
// in, so the download answers at once rather than leaving the caller
// watching a file nothing is going to write. The message is the same one
// the darwin half delivers, so an application reads one answer everywhere
// and shows the file it has or says why it has none.
func download(path string, _, _ time.Duration) Downloaded {
	return Downloaded{Path: path, Err: "nothing is in the cloud on this platform"}
}
