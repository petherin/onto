package navigation

import "github.com/petherin/onto/internal/domain/universe"

// Pathfinder finds a route between two locations in a universe.
// Implementations are free to use any graph algorithm.
type Pathfinder interface {
	FindRoute(u *universe.Universe, from, to string) ([]universe.Edge, bool)
}

// FindRoute runs BFS from from to to across the universe graph.
func FindRoute(u *universe.Universe, from, to string) ([]universe.Edge, bool) {
	if from == to {
		return nil, true
	}

	visited := map[string]bool{from: true}
	parents := map[string]string{}
	parentEdges := map[string]universe.Edge{}
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

// PathDistance sums the distance of each edge in a path.
func PathDistance(path []universe.Edge) float64 {
	total := 0.0
	for _, e := range path {
		total += e.Distance
	}
	return total
}

// PathCost sums the cost of each edge in a path.
func PathCost(path []universe.Edge) float64 {
	total := 0.0
	for _, e := range path {
		total += e.Cost
	}
	return total
}

func reconstructRoute(parents map[string]string, parentEdges map[string]universe.Edge, from, to string) []universe.Edge {
	var result []universe.Edge
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
