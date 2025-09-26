package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
	"github.com/stretchr/testify/assert"
)

func TestGetBalanceHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTokenGetter := NewMockBalanceTokener(ctrl)
	mockBalancer := NewMockBalancer(ctrl)

	userID := uuid.New()
	token := "valid-token"

	tests := []struct {
		name                string
		setupMocks          func()
		expectedStatus      int
		expectedResponseKey string // "balance" or "error"
	}{
		{
			name: "successful balance fetch",
			setupMocks: func() {
				mockTokenGetter.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return(token, nil)
				mockTokenGetter.EXPECT().GetClaims(gomock.Any(), token).
					Return(&jwt.Claims{UserID: userID}, nil)
				mockBalancer.EXPECT().GetUserBalance(gomock.Any(), userID).
					Return(100.0, 5000.0, 50.0, nil)
			},
			expectedStatus:      http.StatusOK,
			expectedResponseKey: "balance",
		},
		{
			name: "unauthorized missing token",
			setupMocks: func() {
				mockTokenGetter.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return("", errors.New("no token"))
			},
			expectedStatus:      http.StatusUnauthorized,
			expectedResponseKey: "error",
		},
		{
			name: "unauthorized invalid token",
			setupMocks: func() {
				mockTokenGetter.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return(token, nil)
				mockTokenGetter.EXPECT().GetClaims(gomock.Any(), token).
					Return(nil, errors.New("invalid token"))
			},
			expectedStatus:      http.StatusUnauthorized,
			expectedResponseKey: "error",
		},
		{
			name: "internal server error from balancer",
			setupMocks: func() {
				mockTokenGetter.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return(token, nil)
				mockTokenGetter.EXPECT().GetClaims(gomock.Any(), token).
					Return(&jwt.Claims{UserID: userID}, nil)
				mockBalancer.EXPECT().GetUserBalance(gomock.Any(), userID).
					Return(0.0, 0.0, 0.0, errors.New("db error"))
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedResponseKey: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()
			handler := NewGetBalanceHandler(mockBalancer, mockTokenGetter)

			req := httptest.NewRequest(http.MethodGet, "/balance", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			var body map[string]interface{}
			err := json.NewDecoder(rr.Body).Decode(&body)
			assert.NoError(t, err)

			_, ok := body[tt.expectedResponseKey]
			assert.True(t, ok, "response should contain key %s", tt.expectedResponseKey)
		})
	}
}
