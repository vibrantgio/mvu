package desktop

import "image"

// FileURLs is the payload kind for files dragged in from the Finder or
// another application, named by its MIME type the way Gio's transfer package
// types its payloads. It covers both shapes a source offers files in: a URL
// naming a file that exists, and a promise the source fulfils when the drop
// asks it to. It is the one kind a [DropTarget] accepts today;
// further kinds — images, text, custom types — are an additive extension
// that will deliver message types of their own, so a file path list is one
// registered payload shape rather than the only concept the API has.
const FileURLs = "text/uri-list"

// The three messages a drag session can produce, resolved to zones. There is
// deliberately no rejection message: a drag that carries no acceptable
// payload is refused at the window edge, where the OS itself animates the
// refusal and no drop ever happens, and a drop outside every zone yields
// silence because silence is the contract — deliberately ignored, not lost.

// FilesEntered reports that a drag carrying files moved into the zone's
// rectangle — by entering the window inside it, or by crossing into it
// during the hover. Highlight on it. A drag crossing directly from one zone
// into another delivers the old zone's [FilesExited] first.
type FilesEntered struct {
	Zone int
}

// FilesExited reports that the drag left the zone — for dead space or
// another zone, by leaving the window, or by dropping. A drop ends the
// hover, so highlight state is correct from the Entered/Exited pair alone;
// that holds for a drop whose promised files are still being written too,
// which arrives here when the drop is taken and as [FilesDropped] later.
// Clear the highlight on it.
type FilesExited struct {
	Zone int
}

// FilesDropped delivers a completed drop: the zone that received it, the
// dropped paths in the order the source provided them, and the drop point in
// Gio pixels (window coordinates, upper-left origin). A folder arrives as
// its own path, not its contents. Drops outside every zone produce no
// message at all.
//
// A source that keeps its files in a library of its own drags a promise rather
// than a URL, and the file exists only once the drop asks for it. Such a drop
// arrives as this same message, once the last promised file has been written:
// the promised files are written under the system's temporary directory, in a
// folder named for this package, with a fresh subfolder per drop. The package
// deletes nothing it writes there, so the application copies what it means to
// keep and treats the dropped path as a file it may lose. A promise the source
// fails to fulfil costs its own file only: the message still arrives, carrying
// the files that did.
type FilesDropped struct {
	Zone  int
	Paths []string
	Pos   image.Point
}
