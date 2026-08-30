package universe

// LocationEntity is a named place within the universe. ID is the canonical
// lowercase-hyphenated key used throughout the graph; Name is the
// human-readable display label.
//
// Generated marks a node the game auto-created when expanding a dead end (see
// NewNearbyCluster) rather than a hand-seeded place. It is the real signal the
// location validator uses to skip the ID/coordinate consistency check for such
// nodes, whose IDs use plain index numbering rather than reality-branch
// suffixes. omitempty keeps hand-seeded locations.json unchanged.
type LocationEntity struct {
	ID          string
	Name        string
	Description string
	Coordinate  CoordinateVO
	Generated   bool `json:"Generated,omitempty"`
}
