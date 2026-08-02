package navigation

type Path struct {
	Nodes []string
}

func (g *Graph) ShortestPath(from, to string) ([]string, bool) {
	if from == to {
		return []string{from}, true
	}

	visited := make(map[string]bool)
	queue := []string{from}
	parents := make(map[string]string)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		for _, neighbor := range g.Edges[current] {
			if visited[neighbor] {
				continue
			}
			parents[neighbor] = current
			if neighbor == to {
				return buildPath(parents, from, to), true
			}
			queue = append(queue, neighbor)
		}
	}

	return nil, false
}

func buildPath(parents map[string]string, from, to string) []string {
	path := []string{to}
	for current := to; current != from; current = parents[current] {
		if parent, ok := parents[current]; ok {
			path = append(path, parent)
		} else {
			break
		}
	}

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}
