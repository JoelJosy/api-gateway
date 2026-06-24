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
	UserID string `json:"user_id"`
    Email  string `json:"email"`
	jwt.RegisteredClaims
}

var claimsKey = contextKey{}

// AuthMiddleware takes your gateway configurations as dependencies
func AuthMiddleware(cfg *config.Config, verifyKey *rsa.PublicKey) Middleware {
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
			// 401 unauthorized
			if (len(authHeader) == 0 || strings.HasPrefix(authHeader, "Bearer ")) {
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
			//         UserID: "123",
			//         Role: "admin",
			//     },
			//     Valid: true
			// }

			// reject if parsing failed, signature invalid, or token expired/invalid
			if err != nil || !token.Valid {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return 
			}
			
			// convert type from interface{} to *GatewayClaims and save in claims var
			if claims, ok := token.Claims.(*GatewayClaims); ok {
				// inject user id from payload into
				ctx := context.WithValue(r.Context(), claimsKey, claims.UserID)
				newRequest := r.WithContext(ctx)

				next.ServeHTTP(w, newRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}