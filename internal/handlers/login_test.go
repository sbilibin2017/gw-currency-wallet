package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sbilibin2017/gw-currency-wallet/internal/services"
	"github.com/stretchr/testify/assert"
)

func TestLoginHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type requestBody struct {
		username string
		password string
	}

	tests := []struct {
		name         string
		reqBody      requestBody
		mockSetup    func(m *MockLoginer)
		expectedCode int
		expectedBody map[string]string
		rawBody      bool // if true, pass raw body to simulate invalid JSON
	}{
		{
			name: "success",
			reqBody: requestBody{
				username: "john",
				password: "secret",
			},
			mockSetup: func(m *MockLoginer) {
				m.EXPECT().
					Login(gomock.Any(), "john", "secret").
					Return("JWT_TOKEN", nil)
			},
			expectedCode: http.StatusOK,
			expectedBody: map[string]string{"token": "JWT_TOKEN"},
		},
		{
			name: "invalid credentials",
			reqBody: requestBody{
				username: "alice",
				password: "wrongpass",
			},
			mockSetup: func(m *MockLoginer) {
				m.EXPECT().
					Login(gomock.Any(), "alice", "wrongpass").
					Return("", services.ErrInvalidCredentials)
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: map[string]string{"error": "Invalid username or password"},
		},
		{
			name: "user does not exist",
			reqBody: requestBody{
				username: "bob",
				password: "secret",
			},
			mockSetup: func(m *MockLoginer) {
				m.EXPECT().
					Login(gomock.Any(), "bob", "secret").
					Return("", services.ErrUserDoesNotExist)
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: map[string]string{"error": "Invalid username or password"},
		},
		{
			name: "internal server error",
			reqBody: requestBody{
				username: "eve",
				password: "pass",
			},
			mockSetup: func(m *MockLoginer) {
				m.EXPECT().
					Login(gomock.Any(), "eve", "pass").
					Return("", errors.New("database failure"))
			},
			expectedCode: http.StatusInternalServerError,
			expectedBody: map[string]string{"error": "Internal server error"},
		},
		{
			name:         "invalid json",
			rawBody:      true,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid request body"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockLoginer(ctrl)
			if tt.mockSetup != nil {
				tt.mockSetup(mockSvc)
			}

			handler := NewLoginHandler(mockSvc)

			var req *http.Request
			if tt.rawBody {
				req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString("{invalid json}"))
			} else {
				bodyBytes, _ := json.Marshal(LoginRequest{
					Username: tt.reqBody.username,
					Password: tt.reqBody.password,
				})
				req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(bodyBytes))
			}

			rr := httptest.NewRecorder()
			handler(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)

			var resp map[string]string
			err := json.Unmarshal(rr.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody, resp)
		})
	}
}
