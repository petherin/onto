package universe

// Repository defines persistence operations for a Universe.
// Implementations live in the infrastructure layer.
type Repository interface {
	Load() (*Universe, error)
	Save(u *Universe) error
}
