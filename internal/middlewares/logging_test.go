package middlewares

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		handlerStatus  int
		handlerBody    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "OK response",
			handlerStatus:  http.StatusOK,
			handlerBody:    "hello",
			expectedStatus: http.StatusOK,
			expectedBody:   "hello",
		},
		{
			name:           "Internal server error",
			handlerStatus:  http.StatusInternalServerError,
			handlerBody:    "error",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Next handler returns predefined status and body
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.handlerStatus)
				_, _ = w.Write([]byte(tt.handlerBody))
			})

			handler := LoggingMiddleware(nextHandler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Assert body
			bodyBytes, _ := io.ReadAll(rr.Body)
			assert.Equal(t, tt.expectedBody, string(bodyBytes))

			// Assert X-Request-ID header exists and is non-empty
			reqID := rr.Header().Get("X-Request-ID")
			assert.NotEmpty(t, reqID)
			assert.True(t, strings.TrimSpace(reqID) != "")
		})
	}
}
