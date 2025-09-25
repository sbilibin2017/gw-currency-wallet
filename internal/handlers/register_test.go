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

func TestRegisterHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type requestBody struct {
		username string
		password string
		email    string
	}

	tests := []struct {
		name         string
		reqBody      requestBody
		mockSetup    func(m *MockRegisterer)
		expectedCode int
		expectedBody map[string]string
		rawBody      bool // if true, pass raw body (to simulate invalid JSON)
	}{
		{
			name: "success",
			reqBody: requestBody{
				username: "john",
				password: "secret",
				email:    "john@example.com",
			},
			mockSetup: func(m *MockRegisterer) {
				m.EXPECT().
					Register(gomock.Any(), "john", "secret", "john@example.com").
					Return(nil)
			},
			expectedCode: 201,
			expectedBody: map[string]string{"message": "User registered successfully"},
		},
		{
			name: "user already exists",
			reqBody: requestBody{
				username: "alice",
				password: "pass",
				email:    "alice@example.com",
			},
			mockSetup: func(m *MockRegisterer) {
				m.EXPECT().
					Register(gomock.Any(), "alice", "pass", "alice@example.com").
					Return(services.ErrUserAlreadyExists)
			},
			expectedCode: 400,
			expectedBody: map[string]string{"error": "Username or email already exists"},
		},
		{
			name: "internal server error",
			reqBody: requestBody{
				username: "bob",
				password: "pass",
				email:    "bob@example.com",
			},
			mockSetup: func(m *MockRegisterer) {
				m.EXPECT().
					Register(gomock.Any(), "bob", "pass", "bob@example.com").
					Return(errors.New("database failure"))
			},
			expectedCode: 500,
			expectedBody: map[string]string{"error": "Internal server error"},
		},
		{
			name:         "invalid json",
			rawBody:      true,
			expectedCode: 400,
			expectedBody: map[string]string{"error": "Username or email already exists"}, // matches handler
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockRegisterer(ctrl)
			if tt.mockSetup != nil {
				tt.mockSetup(mockSvc)
			}

			handler := NewRegisterHandler(mockSvc)

			var req *http.Request
			if tt.rawBody {
				req = httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString("{invalid json}"))
			} else {
				bodyBytes, _ := json.Marshal(RegisterRequest{
					Username: tt.reqBody.username,
					Password: tt.reqBody.password,
					Email:    tt.reqBody.email,
				})
				req = httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(bodyBytes))
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
