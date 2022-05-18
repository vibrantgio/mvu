package vibrant

import (
	"io/fs"
	"os"

	"github.com/reactivego/x"
)

// Directory returns an observable of fs.DirEntry that will emit entries
// for all the files in the directory with the name passed in as an argument.
//
//	Example
//		datadir, _ := app.DataDir()
//		vibrant.Directory(path.Join(datadir, "nl.simpleapps", "AppViz")).Println()
//
func Directory(name string) x.Observable[fs.DirEntry] {
	return x.Defer(func() x.Observable[fs.DirEntry] {
		entries, err := os.ReadDir(name)
		return x.Create(func(index int) (fs.DirEntry, error, bool) {
			if index < len(entries) {
				return entries[index], nil, false
			} else {
				var zero os.DirEntry
				return zero, err, true
			}
		})
	})
}
