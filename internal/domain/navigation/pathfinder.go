// Package navigation defines the PathfinderService interface and the pure graph
// functions (FindRoute, PathDistance, PathCost) used to plan routes through a
// universe.Aggregate. Concrete algorithm implementations live in
// internal/infrastructure/navigation so that the domain stays free of
// technical dependencies.
package navigation

import (
	"errors"

	"github.com/petherin/onto/internal/domain/universe"
)

// Sentinel errors returned by routing operations. Callers may use errors.Is
// to distinguish between an unknown destination and an unreachable one.
var (
	ErrUnknownDestination = errors.New("unknown destination")
	ErrNoRoute            = errors.New("no route")
)

// PathfinderService is a domain service interface that finds a route between
// two locations in a universe. Implementations are free to use any graph algorithm.
type PathfinderService interface {
	FindRoute(u *universe.Aggregate, from, to string) ([]universe.EdgeVO, bool)
}

// FindRoute runs BFS from the `from` location to the `to` location across the universe graph.
func FindRoute(u *universe.Aggregate, from, to string) ([]universe.EdgeVO, bool) {
	if from == to {
		return nil, true
	}

	visited := map[string]bool{from: true}
	parents := map[string]string{}
	parentEdges := map[string]universe.EdgeVO{}
	queue := []string{from}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range u.EdgesFrom(current) {
			if visited[edge.To] {
				continue
			}
			visited[edge.To] = true
			parents[edge.To] = current
			parentEdges[edge.To] = edge

			if edge.To == to {
				return reconstructRoute(parents, parentEdges, from, to), true
			}
			queue = append(queue, edge.To)
		}
	}

	return nil, false
}

// PathDistance sums the distance of each EdgeVO in a path.
func PathDistance(path []universe.EdgeVO) float64 {
	total := 0.0
	for _, e := range path {
		total += e.Distance
	}
	return total
}

// PathCost sums the cost of each EdgeVO in a path.
func PathCost(path []universe.EdgeVO) float64 {
	total := 0.0
	for _, e := range path {
		total += e.Cost
	}
	return total
}

func reconstructRoute(parents map[string]string, parentEdges map[string]universe.EdgeVO, from, to string) []universe.EdgeVO {
	var result []universe.EdgeVO
	current := to
	for current != from {
		edge, ok := parentEdges[current]
		if !ok {
			return nil
		}
		result = append(result, edge)
		current = parents[current]
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}
