package middleware

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"

	"github.com/JoelJosy/api-gateway/config"
	"github.com/golang-jwt/jwt/v5"
)

// payload / data expected in jwt
type GatewayClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims // embedded, includes sub
}

var claimsKey = contextKey{}

// AuthMiddleware takes your gateway configurations as dependencies
func AuthMiddleware(cfg *config.Config, verifyKey *rsa.PublicKey) Middleware {
	// return middleware
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var matchedRoute *config.Route
			
			// Find the correct route using prefix matching
			for i := range cfg.Routes {
				if strings.HasPrefix(r.URL.Path, cfg.Routes[i].Path) {
					matchedRoute = &cfg.Routes[i]
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
			// 401 unauthorized
			if (authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ")) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return 
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			// Provide pointer to empty struct where decoded claims will be stored
			// Callback provides key used to verify signature
			// JWT library: parses header/payload, gets key via callback,
			// verifies signature, then decodes payload into struct
			token, err := jwt.ParseWithClaims(tokenString, &GatewayClaims{}, func(token *jwt.Token) (any, error) {
				// Verify the signing method is RS256
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return verifyKey, nil
			})

			// Example: 
			// token := &jwt.Token{
			//     Claims: &GatewayClaims{
			//         Role: "admin",
			// 		   sub: "user123",
			//		   etc...
			//     },
			//     Valid: true
			// }

			// reject if parsing failed, signature invalid, or token expired/invalid
			if err != nil || !token.Valid {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return 
			}
			
			// convert type from interface{} to *GatewayClaims and save in claims var
			claims, ok := token.Claims.(*GatewayClaims)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			// inject user id from payload into context
			ctx := context.WithValue(r.Context(), claimsKey, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}