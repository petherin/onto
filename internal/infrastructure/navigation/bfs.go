package navigation

import (
	domainnavigation "github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
)

// BFSPathfinder implements domain/navigation.Pathfinder using breadth-first search.
type BFSPathfinder struct{}

func NewBFSPathfinder() *BFSPathfinder { return &BFSPathfinder{} }

func (p *BFSPathfinder) FindRoute(u *universe.Universe, from, to string) ([]universe.Edge, bool) {
	return domainnavigation.FindRoute(u, from, to)
}
