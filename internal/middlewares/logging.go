package middlewares

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	"go.uber.org/zap"
)

// LoggingMiddleware logs requests and responses, generating a unique request ID for each request.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.New().String()
		start := time.Now()

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		w.Header().Set("X-Request-ID", reqID)

		// Call the next handler
		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// Log request
		logger.Log.Desugar().Info("request",
			zap.String("request_id", reqID),
			zap.String("method", r.Method),
			zap.String("uri", r.RequestURI),
			zap.Int64("duration_ms", duration.Milliseconds()),
		)

		// Log response
		logger.Log.Desugar().Info("response",
			zap.String("request_id", reqID),
			zap.Int("status", rw.statusCode),
			zap.Int("response_size_bytes", rw.size),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code and size
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
