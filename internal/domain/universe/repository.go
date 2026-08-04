package universe

// Repository defines persistence operations for an Aggregate.
// Implementations live in the infrastructure layer.
type Repository interface {
	Load() (*Aggregate, error)
	Save(u *Aggregate) error
}
