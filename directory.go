package vibrant

import (
	"io/fs"
	"os"

	rx "github.com/reactivego/observable"
)

// Directory returns an observable of fs.DirEntry that will emit entries
// for all the files in the directory with the name passed in as an argument.
//
//	Example
//		datadir, _ := app.DataDir()
//		vibrant.Directory(path.Join(datadir, "nl.simpleapps", "AppViz")).Println()
//
func Directory(name string) rx.Observable[fs.DirEntry] {
	return rx.Defer(func() rx.Observable[fs.DirEntry] {
		entries, err := os.ReadDir(name)
		return rx.CreateRecursive(func() (fs.DirEntry, error, bool) {
			if len(entries) > 0 {
				next := entries[0]
				entries = entries[1:]
				return next, nil, false
			} else {
				var zero os.DirEntry
				return zero, err, true
			}
		})
	})
}
