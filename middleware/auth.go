package middleware

import (
	"net/http"
	"strings"

	"github.com/JoelJosy/api-gateway/config"
)

// AuthMiddleware takes your gateway configurations as dependencies
func AuthMiddleware(cfg *config.Config) Middleware {
	// return middleware
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var matchedRoute *config.Route
			
			// Find the correct route using prefix matching
			for _, route := range cfg.Routes {
				if strings.HasPrefix(r.URL.Path, route.Path) {
					matchedRoute = &route
					break
				}
			}
			
			// if no match, let proxy handle 404 downstream
			if matchedRoute == nil {
                next.ServeHTTP(w, r)
                return
            }
			
			// Public endpoint, let it through
			if !matchedRoute.Auth_Required {
                next.ServeHTTP(w, r) 
                return
            }

			// Auth required
			authHeader := r.Header.Get("Authorization")
			if (len(authHeader) == 0 || strings.HasPrefix(authHeader, "Bearer ")) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return 
			}
			

			next.ServeHTTP(w, r)
		})
	}
}