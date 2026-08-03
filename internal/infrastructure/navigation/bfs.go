// Package navigation provides the concrete BFSPathfinder, which implements the
// domain/navigation.PathfinderService interface using breadth-first search.
// Keeping the implementation here ensures the domain layer has no dependency on
// any specific graph algorithm.
package navigation

import (
	domainnavigation "github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
)

// BFSPathfinder implements domain/navigation.PathfinderService using breadth-first search.
type BFSPathfinder struct{}

// NewBFSPathfinder returns a ready-to-use BFSPathfinder.
func NewBFSPathfinder() *BFSPathfinder { return &BFSPathfinder{} }

// FindRoute delegates to the domain-layer BFS implementation.
func (p *BFSPathfinder) FindRoute(u *universe.UniverseAggregate, from, to string) ([]universe.EdgeVO, bool) {
	return domainnavigation.FindRoute(u, from, to)
}
