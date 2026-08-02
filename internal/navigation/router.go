package navigation

type Router struct {
	routes map[string]string
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]string)}
}

func (r *Router) AddRoute(name, target string) {
	r.routes[name] = target
}

func (r *Router) Resolve(name string) (string, bool) {
	target, ok := r.routes[name]
	return target, ok
}
