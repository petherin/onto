package universe

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
