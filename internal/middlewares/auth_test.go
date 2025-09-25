package middlewares

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name             string
		mockSetup        func(m *MockTokener)
		expectedStatus   int
		expectNextCalled bool
	}{
		{
			name: "NoToken",
			mockSetup: func(m *MockTokener) {
				m.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return("", errors.New("no token"))
			},
			expectedStatus:   http.StatusUnauthorized,
			expectNextCalled: false,
		},
		{
			name: "InvalidToken",
			mockSetup: func(m *MockTokener) {
				m.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return("sometoken", nil)
				m.EXPECT().Validate(gomock.Any(), "sometoken").
					Return(errors.New("invalid token"))
			},
			expectedStatus:   http.StatusUnauthorized,
			expectNextCalled: false,
		},
		{
			name: "ValidToken",
			mockSetup: func(m *MockTokener) {
				m.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return("validtoken", nil)
				m.EXPECT().Validate(gomock.Any(), "validtoken").
					Return(nil)
			},
			expectedStatus:   http.StatusOK,
			expectNextCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTokener := NewMockTokener(ctrl)
			tt.mockSetup(mockTokener)

			// Wrap a next handler to check if it was called
			nextCalled := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := AuthMiddleware(mockTokener)(nextHandler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectNextCalled, nextCalled)
		})
	}
}
