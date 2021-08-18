package vibrant

import (
	"os"
	"path/filepath"

	_ "github.com/reactivego/rx/generic"

	"gioui.org/app"
)

func DataDir(vendor, appname string) (dir string, err error) {
	if dir, err = app.DataDir(); err == nil {
		dir = filepath.Join(dir, vendor, appname)
	}
	return
}

func EnforceDataDir(vendor, appname string) ObservableString {
	return CreateString(func(N NextString, E Error, C Complete, X Canceled) {
		if datadir, err := DataDir(vendor, appname); err != nil {
			E(err)
		} else {
			if err := os.MkdirAll(datadir, os.ModePerm); err != nil {
				E(err)
			} else {
				N(datadir)
				if !X() {
					C()
				}
			}
		}
	})
}

//jig:type DirEntry = fs.DirEntry

func ReadDir(name string) ObservableDirEntry {
	return DeferDirEntry(func() ObservableDirEntry {
		entries, err := os.ReadDir(name)
		return CreateRecursiveDirEntry(func(N NextDirEntry, E Error, C Complete) {
			if len(entries) > 0 {
				N(entries[0])
				entries[0] = nil
				entries = entries[1:]
			} else if err != nil {
				E(err)
			} else {
				C()
			}
		})
	})
}
