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
//
// Trap carries the trap archetype for a trap node — a generated trap or the
// hand-placed well (NoTrap for ordinary places). Like the sink/dead-end
// distinction, a trap is structural — carried here on the location, never
// inferred from its ID or name. omitempty keeps ordinary nodes and hand-seeded
// locations.json unchanged.
type LocationEntity struct {
	ID          string
	Name        string
	Description string
	Coordinate  CoordinateVO
	Generated   bool     `json:"Generated,omitempty"`
	Trap        TrapType `json:"Trap,omitempty"`
}
