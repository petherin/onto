package universe

// LocationGenerator creates new outgoing locations when a dead end is reached.
// Implementations live in the infrastructure and interface layers.
type LocationGenerator interface {
	Handle(u *Universe, id string, coord Coordinate) bool
}
