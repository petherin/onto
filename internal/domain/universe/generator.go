package universe

// LocationGeneratorService is a domain service interface that creates new
// outgoing locations when a dead end is reached.
// Implementations live in the infrastructure and interface layers.
type LocationGeneratorService interface {
	Handle(u *Aggregate, id string, coord CoordinateVO) bool
}
