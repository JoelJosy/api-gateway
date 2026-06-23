package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

type contextKey struct {}
var requestIdKey = contextKey{}

// generate random string id
func NewID() string {
	b := make([]byte, 8) // 8 bytes = 16 hex chars
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
	
}


type responseRecorder struct {
	http.ResponseWriter // Embeds the original interface to get its methods
	statusCode int
}

// overwrite WriteHeader method to catch status code
func (rw *responseRecorder) WriteHeader(code int) {
	rw.statusCode = code          
	rw.ResponseWriter.WriteHeader(code) // Pass it through to the real client
}

func LoggerMiddleware(next http.Handler) http.Handler {
	// converts a function into a HandlerFunc type, which has ServeHTTP method (return Handler)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// create new context, as contexts cannot be modified
		// contexts contain data related to requests (cancellation signals, timeouts etc)
		id := NewID()
		ctx := context.WithValue(r.Context(), requestIdKey, id)
		
		// structured logging, messg, key:val, key:val
		slog.Info("incoming request", "path", r.URL.Path, "request_id", id)
		
		// shallow copy of request with new context
		newRequest := r.WithContext(ctx)

		// to intercept response status code
		rec := &responseRecorder{
    		ResponseWriter: w,
    		statusCode:     http.StatusOK, // Default to 200
		}

		// measure elapsed time 
		start := time.Now()
		next.ServeHTTP(rec, newRequest)
		duration := time.Since(start)

		slog.Info(
			"incoming request", "path", r.URL.Path, "request_id", id, "duration_ms", 
			duration.Milliseconds(), "status_code", rec.statusCode)
	})
}