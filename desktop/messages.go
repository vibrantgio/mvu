package desktop

import "image"

// FileURLs is the payload kind for file URLs dragged in from the Finder or
// another application, named by its MIME type the way Gio's transfer package
// types its payloads. It is the one kind a [DropTarget] registers today;
// further kinds — images, text, custom types — are an additive extension
// that will deliver message types of their own, so a file path list is one
// registered payload shape rather than the only concept the API has.
const FileURLs = "text/uri-list"

// The three messages a drag session can produce, resolved to zones. There is
// deliberately no rejection message: a drag that carries no acceptable
// payload is refused at the window edge, where the OS itself animates the
// refusal and no drop ever happens, and a drop outside every zone yields
// silence because silence is the contract — deliberately ignored, not lost.

// FilesEntered reports that a drag carrying file URLs moved into the zone's
// rectangle — by entering the window inside it, or by crossing into it
// during the hover. Highlight on it. A drag crossing directly from one zone
// into another delivers the old zone's [FilesExited] first.
type FilesEntered struct {
	Zone int
}

// FilesExited reports that the drag left the zone — for dead space or
// another zone, by leaving the window, or by dropping. A drop ends the
// hover, so highlight state is correct from the Entered/Exited pair alone.
// Clear the highlight on it.
type FilesExited struct {
	Zone int
}

// FilesDropped delivers a completed drop: the zone that received it, the
// dropped paths in the order the source provided them, and the drop point in
// Gio pixels (window coordinates, upper-left origin). A folder arrives as
// its own path, not its contents. Drops outside every zone produce no
// message at all.
type FilesDropped struct {
	Zone  int
	Paths []string
	Pos   image.Point
}
