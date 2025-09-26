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

	mockSvc := NewMockRegisterer(ctrl)

	tests := []struct {
		name         string
		inputBody    interface{}
		mockSetup    func()
		expectedCode int
		expectedBody interface{}
	}{
		{
			name: "success",
			inputBody: RegisterRequest{
				Username: "john",
				Password: "pass123",
				Email:    "john@example.com",
			},
			mockSetup: func() {
				mockSvc.EXPECT().
					Register(gomock.Any(), "john", "pass123", "john@example.com").
					Return(nil)
			},
			expectedCode: http.StatusCreated,
			expectedBody: &RegisterResponse{
				Message: "User registered successfully",
			},
		},
		{
			name: "user already exists",
			inputBody: RegisterRequest{
				Username: "existing",
				Password: "pass123",
				Email:    "existing@example.com",
			},
			mockSetup: func() {
				mockSvc.EXPECT().
					Register(gomock.Any(), "existing", "pass123", "existing@example.com").
					Return(services.ErrUserAlreadyExists)
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: &RegisterErrorResponse{
				Error: "Username or email already exists",
			},
		},
		{
			name:         "invalid JSON",
			inputBody:    "{invalid json}",
			mockSetup:    func() {},
			expectedCode: http.StatusBadRequest,
			expectedBody: &RegisterErrorResponse{
				Error: "Username or email already exists",
			},
		},
		{
			name: "internal error",
			inputBody: RegisterRequest{
				Username: "john",
				Password: "pass123",
				Email:    "john@example.com",
			},
			mockSetup: func() {
				mockSvc.EXPECT().
					Register(gomock.Any(), "john", "pass123", "john@example.com").
					Return(errors.New("database error"))
			},
			expectedCode: http.StatusInternalServerError,
			expectedBody: &RegisterErrorResponse{
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

			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			handler := NewRegisterHandler(mockSvc)
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if w.Code != http.StatusNoContent {
				var respBody interface{}
				switch tt.expectedCode {
				case http.StatusCreated:
					respBody = &RegisterResponse{}
				default:
					respBody = &RegisterErrorResponse{}
				}
				err := json.Unmarshal(w.Body.Bytes(), respBody)
				assert.NoError(t, err)
				// Now both are pointers
				assert.Equal(t, tt.expectedBody, respBody)
			}
		})
	}
}
