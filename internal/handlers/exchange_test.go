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
	"github.com/sbilibin2017/gw-currency-wallet/internal/services"
	"github.com/stretchr/testify/assert"
)

func TestExchangeHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTokener := NewMockExchangeRateForCurrencyTokener(ctrl)
	mockExchanger := NewMockExchanger(ctrl)

	userID := uuid.New()

	handler := NewExchangeHandler(mockTokener, mockExchanger)

	// Allow token calls for all subtests
	mockTokener.EXPECT().
		GetTokenFromRequest(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return("valid-token", nil)
	mockTokener.EXPECT().
		GetClaims(gomock.Any(), "valid-token").
		AnyTimes().
		Return(&jwt.Claims{UserID: userID}, nil)

	tests := []struct {
		name           string
		reqBody        interface{}
		mockExchange   func()
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "success",
			reqBody: ExchangeRequest{
				FromCurrency: "USD",
				ToCurrency:   "EUR",
				Amount:       100,
			},
			mockExchange: func() {
				mockExchanger.EXPECT().
					Exchange(gomock.Any(), userID, "USD", "EUR", 100.0).
					Return(float32(85.0), 200.0, 5000.0, 50.0, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: ExchangeResponse{
				Message:         "Exchange successful",
				ExchangedAmount: 85.0,
				NewBalance: ExchangedBalance{
					USD: 200.0,
					RUB: 5000.0,
					EUR: 50.0,
				},
			},
		},
		{
			name:           "bad_request_invalid_amount",
			reqBody:        ExchangeRequest{FromCurrency: "USD", ToCurrency: "EUR", Amount: -10},
			mockExchange:   nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   ExchangeErrorResponse{Error: "Insufficient funds or invalid currencies"},
		},
		{
			name:           "bad_request_invalid_json",
			reqBody:        `invalid-json`,
			mockExchange:   nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   ExchangeErrorResponse{Error: "Insufficient funds or invalid currencies"},
		},
		{
			name: "insufficient_funds",
			reqBody: ExchangeRequest{
				FromCurrency: "USD",
				ToCurrency:   "EUR",
				Amount:       100,
			},
			mockExchange: func() {
				mockExchanger.EXPECT().
					Exchange(gomock.Any(), userID, "USD", "EUR", 100.0).
					Return(float32(0), 100.0, 5000.0, 50.0, services.ErrInsufficientFunds)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   ExchangeErrorResponse{Error: "Insufficient funds or invalid currencies"},
		},
		{
			name: "internal_server_error",
			reqBody: ExchangeRequest{
				FromCurrency: "USD",
				ToCurrency:   "EUR",
				Amount:       100,
			},
			mockExchange: func() {
				mockExchanger.EXPECT().
					Exchange(gomock.Any(), userID, "USD", "EUR", 100.0).
					Return(float32(0), 100.0, 5000.0, 50.0, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ExchangeErrorResponse{Error: "Internal server error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockExchange != nil {
				tt.mockExchange()
			}

			var bodyBytes []byte
			switch v := tt.reqBody.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/exchange", bytes.NewReader(bodyBytes))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Result().StatusCode)

			respBody := rec.Body.Bytes()
			switch expected := tt.expectedBody.(type) {
			case ExchangeResponse:
				var got ExchangeResponse
				err := json.Unmarshal(respBody, &got)
				assert.NoError(t, err)
				assert.Equal(t, expected, got)
			case ExchangeErrorResponse:
				var got ExchangeErrorResponse
				err := json.Unmarshal(respBody, &got)
				assert.NoError(t, err)
				assert.Equal(t, expected, got)
			}
		})
	}
}
