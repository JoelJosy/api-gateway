package middleware

import "net/http"

type Middleware func(http.Handler) http.Handler


// chain middlewares to run sequentially
func Chain(base http.Handler, middlewares ...Middleware) http.Handler {
	for i:= len(middlewares) - 1; i >= 0; i-- {
		base = middlewares[i](base)
	}
	return base
}