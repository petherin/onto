package reality

type Universe struct {
	Locations map[string]Location
	Edges     map[string][]Edge
}

func NewUniverse() *Universe {
	return &Universe{
		Locations: make(map[string]Location),
		Edges:     make(map[string][]Edge),
	}
}

func (u *Universe) AddLocation(location Location) {
	u.Locations[location.ID] = location
}

func (u *Universe) AddEdge(edge Edge) {
	u.Edges[edge.From] = append(u.Edges[edge.From], edge)
}

func (u *Universe) GetLocation(id string) (Location, bool) {
	location, ok := u.Locations[id]
	return location, ok
}

func (u *Universe) FindRoute(from, to string) ([]Edge, bool) {
	if from == to {
		return nil, true
	}

	visited := map[string]bool{from: true}
	parents := map[string]string{}
	parentEdges := map[string]Edge{}
	queue := []string{from}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range u.Edges[current] {
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

func reconstructRoute(parents map[string]string, parentEdges map[string]Edge, from, to string) []Edge {
	result := []Edge{}
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

func (u *Universe) Travel(from, to string) (TravelPlan, bool) {
	path, ok := u.FindRoute(from, to)
	if !ok {
		return TravelPlan{}, false
	}

	route := []string{from}
	totalDistance := 0.0
	for _, edge := range path {
		route = append(route, edge.To)
		totalDistance += edge.Distance
	}

	return TravelPlan{Route: route, Distance: totalDistance, Estimated: true}, true
}
