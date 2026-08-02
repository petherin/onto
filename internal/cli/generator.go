package cli

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/petherin/onto/internal/reality"
)

// autoGenerateNearby creates a single nearby node and bidirectional edges
// from the given id. Returns true when a node was created.
func autoGenerateNearby(a *App, id string) bool {
	rand.Seed(time.Now().UnixNano())
	for i := 1; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-%d", id, i)
		if _, ok := a.universe.GetLocation(candidate); !ok {
			coord := a.currentCoordinate
			coord.Location = fmt.Sprintf("Nearby %d", i)
			loc := reality.Location{ID: candidate, Name: fmt.Sprintf("Nearby %d", i), Description: "Auto-generated nearby location", Coordinate: coord}
			a.universe.AddLocation(loc)
			edge := reality.Edge{From: id, To: candidate, Mode: reality.Walk, Distance: 0.5 + rand.Float64(), Cost: 1, Description: "Auto-generated path"}
			a.universe.AddEdge(edge)
			reverse := reality.Edge{From: candidate, To: id, Mode: reality.Walk, Distance: edge.Distance, Cost: edge.Cost, Description: "Auto-generated return path"}
			a.universe.AddEdge(reverse)
			return true
		}
	}
	return false
}
