package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
	"github.com/stretchr/testify/assert"
)

func TestDepositHandler(t *testing.T) {
	userID := uuid.New()
	validToken := "valid-token"

	tests := []struct {
		name               string
		requestBody        any
		setupMocks         func(mockWriter *MockDepositWriter, mockTokener *MockDepositTokener)
		expectedStatusCode int
		expectedKey        string
	}{
		{
			name: "successful deposit",
			requestBody: DepositRequest{
				Amount:   100.0,
				Currency: "USD",
			},
			setupMocks: func(mockWriter *MockDepositWriter, mockTokener *MockDepositTokener) {
				mockTokener.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return(validToken, nil)
				mockTokener.EXPECT().GetClaims(gomock.Any(), validToken).Return(&jwt.Claims{UserID: userID}, nil)
				mockWriter.EXPECT().Deposit(gomock.Any(), userID, 100.0, "USD").Return(200.0, 5000.0, 50.0, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedKey:        "message",
		},
		{
			name:        "invalid request body",
			requestBody: "invalid-json",
			setupMocks: func(mockWriter *MockDepositWriter, mockTokener *MockDepositTokener) {
				mockTokener.EXPECT().
					GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return(validToken, nil)
				mockTokener.EXPECT().
					GetClaims(gomock.Any(), validToken).
					Return(&jwt.Claims{UserID: userID}, nil) // <- add this line
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedKey:        "error",
		},
		{
			name: "unauthorized missing token",
			requestBody: DepositRequest{
				Amount:   100.0,
				Currency: "USD",
			},
			setupMocks: func(mockWriter *MockDepositWriter, mockTokener *MockDepositTokener) {
				mockTokener.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("", http.ErrNoCookie)
			},
			expectedStatusCode: http.StatusUnauthorized,
			expectedKey:        "error",
		},
		{
			name: "unauthorized invalid token",
			requestBody: DepositRequest{
				Amount:   100.0,
				Currency: "USD",
			},
			setupMocks: func(mockWriter *MockDepositWriter, mockTokener *MockDepositTokener) {
				mockTokener.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return(validToken, nil)
				mockTokener.EXPECT().GetClaims(gomock.Any(), validToken).Return(nil, http.ErrNoCookie)
			},
			expectedStatusCode: http.StatusUnauthorized,
			expectedKey:        "error",
		},
		{
			name: "invalid amount",
			requestBody: DepositRequest{
				Amount:   -10.0,
				Currency: "USD",
			},
			setupMocks: func(mockWriter *MockDepositWriter, mockTokener *MockDepositTokener) {
				mockTokener.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return(validToken, nil)
				mockTokener.EXPECT().GetClaims(gomock.Any(), validToken).Return(&jwt.Claims{UserID: userID}, nil)
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedKey:        "error",
		},
		{
			name: "invalid currency",
			requestBody: DepositRequest{
				Amount:   100.0,
				Currency: "BTC",
			},
			setupMocks: func(mockWriter *MockDepositWriter, mockTokener *MockDepositTokener) {
				mockTokener.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return(validToken, nil)
				mockTokener.EXPECT().GetClaims(gomock.Any(), validToken).Return(&jwt.Claims{UserID: userID}, nil)
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedKey:        "error",
		},
		{
			name: "internal server error from writer",
			requestBody: DepositRequest{
				Amount:   100.0,
				Currency: "USD",
			},
			setupMocks: func(mockWriter *MockDepositWriter, mockTokener *MockDepositTokener) {
				mockTokener.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return(validToken, nil)
				mockTokener.EXPECT().GetClaims(gomock.Any(), validToken).Return(&jwt.Claims{UserID: userID}, nil)
				mockWriter.EXPECT().Deposit(gomock.Any(), userID, 100.0, "USD").Return(0.0, 0.0, 0.0, assert.AnError)
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedKey:        "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockTokener := NewMockDepositTokener(ctrl)
			mockWriter := NewMockDepositWriter(ctrl)

			tt.setupMocks(mockWriter, mockTokener)

			var bodyBytes []byte
			switch v := tt.requestBody.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/wallet/deposit", bytes.NewReader(bodyBytes))
			rr := httptest.NewRecorder()

			handler := NewDepositHandler(mockWriter, mockTokener)
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			var resp map[string]interface{}
			err := json.NewDecoder(rr.Body).Decode(&resp)
			assert.NoError(t, err)

			_, ok := resp[tt.expectedKey]
			assert.True(t, ok, "response should contain key %s", tt.expectedKey)
		})
	}
}
