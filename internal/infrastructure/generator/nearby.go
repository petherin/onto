// Package generator implements the universe.LocationGenerator interface.
// NearbyGenerator automatically creates a plausible neighbouring location and
// bidirectional walk edges when the traveller reaches a dead end, keeping the
// world explorable without manual data entry.
package generator

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/petherin/onto/internal/domain/universe"
)

// NearbyGenerator implements universe.LocationGenerator by auto-generating
// a neighbouring location with bidirectional walk edges.
type NearbyGenerator struct{}

// New returns a ready-to-use NearbyGenerator.
func New() *NearbyGenerator {
	return &NearbyGenerator{}
}

// Handle creates a new nearby location with bidirectional walk edges and adds
// it to the universe. Returns true if a new location was created.
func (g *NearbyGenerator) Handle(u *universe.Universe, id string, coord universe.Coordinate) bool {
	rand.Seed(time.Now().UnixNano())
	for i := 1; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-%d", id, i)
		if _, ok := u.GetLocation(candidate); !ok {
			c := coord
			c.Location = fmt.Sprintf("Nearby %d", i)
			loc := universe.Location{
				ID:          candidate,
				Name:        fmt.Sprintf("Nearby %d", i),
				Description: "Auto-generated nearby location",
				Coordinate:  c,
			}
			u.AddLocation(loc)
			edge := universe.Edge{From: id, To: candidate, Mode: universe.Walk, Distance: 0.5 + rand.Float64(), Cost: 1, Description: "Auto-generated path"}
			u.AddEdge(edge)
			reverse := universe.Edge{From: candidate, To: id, Mode: universe.Walk, Distance: edge.Distance, Cost: edge.Cost, Description: "Auto-generated return path"}
			u.AddEdge(reverse)
			return true
		}
	}
	return false
}
