package router

import (
	"errors"
	"sort"
	"strings"

	"github.com/JoelJosy/api-gateway/config"
)

// router handles pushing reqs upstream
type Router struct {
	routes []config.Route
}

// initializes a Router and sorts routes by path length descending
func NewRouter(routes []config.Route) *Router {
	sortedRoutes := make([]config.Route, len(routes))
	copy(sortedRoutes, routes)

	sort.Slice(sortedRoutes, func(i, j int) bool {
		return len(sortedRoutes[i].Path) > len(sortedRoutes[j].Path)
	})
	return &Router{sortedRoutes}
}

// finds the first upstream URL that matches the incoming path prefix
func (r *Router) Match(path string) (string, string, error) {
	for _, route := range r.routes {
		if strings.HasPrefix(path, route.Path) {
			return route.Upstream, route.Path, nil
		}
	}
	return "", "", errors.New("no route matched")
}