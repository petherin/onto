package universe

// LocationEntity is a named place within the universe. ID is the canonical
// lowercase-hyphenated key used throughout the graph; Name is the
// human-readable display label.
type LocationEntity struct {
	ID          string
	Name        string
	Description string
	Coordinate  CoordinateVO
}
