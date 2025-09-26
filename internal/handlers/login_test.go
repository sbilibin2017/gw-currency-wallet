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

	mockSvc := NewMockLoginer(ctrl)

	tests := []struct {
		name         string
		inputBody    interface{}
		mockSetup    func()
		expectedCode int
		expectedBody interface{}
	}{
		{
			name: "success",
			inputBody: LoginRequest{
				Username: "john",
				Password: "pass123",
			},
			mockSetup: func() {
				mockSvc.EXPECT().
					Login(gomock.Any(), "john", "pass123").
					Return("JWT_TOKEN", nil)
			},
			expectedCode: http.StatusOK,
			expectedBody: &LoginResponse{
				Token: "JWT_TOKEN",
			},
		},
		{
			name:         "invalid JSON",
			inputBody:    "{invalid json}",
			mockSetup:    func() {},
			expectedCode: http.StatusBadRequest,
			expectedBody: &LoginErrorResponse{
				Error: "invalid request body",
			},
		},
		{
			name: "user does not exist / wrong credentials",
			inputBody: LoginRequest{
				Username: "wronguser",
				Password: "wrongpass",
			},
			mockSetup: func() {
				mockSvc.EXPECT().
					Login(gomock.Any(), "wronguser", "wrongpass").
					Return("", services.ErrUserDoesNotExist)
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: &LoginErrorResponse{
				Error: "Invalid username or password",
			},
		},
		{
			name: "internal error",
			inputBody: LoginRequest{
				Username: "john",
				Password: "pass123",
			},
			mockSetup: func() {
				mockSvc.EXPECT().
					Login(gomock.Any(), "john", "pass123").
					Return("", errors.New("database error"))
			},
			expectedCode: http.StatusInternalServerError,
			expectedBody: &LoginErrorResponse{
				Error: "Internal server error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			var bodyBytes []byte
			switch v := tt.inputBody.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			handler := NewLoginHandler(mockSvc)
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if w.Code != http.StatusNoContent {
				var respBody interface{}
				switch tt.expectedCode {
				case http.StatusOK:
					respBody = &LoginResponse{}
				default:
					respBody = &LoginErrorResponse{}
				}
				err := json.Unmarshal(w.Body.Bytes(), respBody)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, respBody)
			}
		})
	}
}
