package universe

// UniverseRepository defines persistence operations for a UniverseAggregate.
// Implementations live in the infrastructure layer.
type UniverseRepository interface {
	Load() (*UniverseAggregate, error)
	Save(u *UniverseAggregate) error
}
