package navigation

type Node struct {
	ID   string
	Name string
}

type Graph struct {
	Nodes map[string]Node
	Edges map[string][]string
}

func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]Node),
		Edges: make(map[string][]string),
	}
}

func (g *Graph) AddNode(node Node) {
	g.Nodes[node.ID] = node
}

func (g *Graph) AddEdge(from, to string) {
	g.Edges[from] = append(g.Edges[from], to)
}
