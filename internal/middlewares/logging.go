package middlewares

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// LoggingMiddleware returns a middleware that logs requests and responses using the provided SugaredLogger.
// It also generates a unique request ID for each HTTP request.
func LoggingMiddleware(log *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate a new UUID for this request
			reqID := uuid.New().String()

			start := time.Now()

			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Add request ID to context and headers for downstream handlers
			r = r.WithContext(
				context.WithValue(r.Context(), "requestID", reqID),
			)
			w.Header().Set("X-Request-ID", reqID)

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			log.Infow("request",
				"request_id", reqID,
				"method", r.Method,
				"uri", r.RequestURI,
				"duration", duration,
			)

			log.Infow("response",
				"request_id", reqID,
				"status", rw.statusCode,
				"response_size", strconv.Itoa(rw.size)+"B",
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}
