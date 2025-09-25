package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	reflect "reflect"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/handlers"
	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
	"github.com/sbilibin2017/gw-currency-wallet/internal/services"
	"github.com/stretchr/testify/require"
)

func TestExchangeHandler(t *testing.T) {
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name      string
		request   handlers.ExchangeRequest
		mockSetup func(tokener *handlers.MockExchangeRateForCurrencyTokener, exchanger *handlers.MockExchanger)
		wantCode  int
		wantBody  map[string]interface{}
	}{
		{
			name:    "success",
			request: handlers.ExchangeRequest{FromCurrency: "USD", ToCurrency: "EUR", Amount: 100},
			mockSetup: func(tokener *handlers.MockExchangeRateForCurrencyTokener, exchanger *handlers.MockExchanger) {
				tokener.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				tokener.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
				exchanger.EXPECT().Exchange(gomock.Any(), userID, "USD", "EUR", 100.0).
					Return(float32(85), map[string]float64{
						models.USD: 1000, models.RUB: 90000, models.EUR: 185,
					}, nil)
			},
			wantCode: http.StatusOK,
			wantBody: map[string]interface{}{
				"message":          "Exchange successful",
				"exchanged_amount": float64(85),
				"new_balance": map[string]interface{}{
					"USD": float64(1000),
					"RUB": float64(90000),
					"EUR": float64(185),
				},
			},
		},
		{
			name:    "bad_request_invalid_amount",
			request: handlers.ExchangeRequest{FromCurrency: "USD", ToCurrency: "EUR", Amount: -10},
			mockSetup: func(tokener *handlers.MockExchangeRateForCurrencyTokener, exchanger *handlers.MockExchanger) {
				// Even though amount is invalid, the handler still calls token methods
				tokener.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				tokener.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
			},
			wantCode: http.StatusBadRequest,
			wantBody: map[string]interface{}{"error": "Insufficient funds or invalid currencies"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh controller & mocks for each subtest
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockTokener := handlers.NewMockExchangeRateForCurrencyTokener(ctrl)
			mockExchanger := handlers.NewMockExchanger(ctrl)

			tt.mockSetup(mockTokener, mockExchanger)

			handler := handlers.NewExchangeHandler(mockTokener, mockExchanger)

			bodyBytes, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/exchange", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			handler(w, req)

			res := w.Result()
			defer res.Body.Close()

			require.Equal(t, tt.wantCode, res.StatusCode)

			var body map[string]interface{}
			err := json.NewDecoder(res.Body).Decode(&body)
			require.NoError(t, err)
			require.Equal(t, tt.wantBody, body)
		})
	}
}

func TestExchangeHandler_Errors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uuid.New()

	tokener := handlers.NewMockExchangeRateForCurrencyTokener(ctrl)
	exchanger := handlers.NewMockExchanger(ctrl)

	handler := handlers.NewExchangeHandler(tokener, exchanger)

	tests := []struct {
		name        string
		mockSetup   func()
		requestBody string
		wantCode    int
		wantBody    map[string]interface{}
	}{
		{
			name: "insufficient_funds",
			mockSetup: func() {
				tokener.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				tokener.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
				exchanger.EXPECT().
					Exchange(gomock.Any(), userID, "USD", "EUR", 100.0).
					Return(float32(0), nil, services.ErrInsufficientFunds)
			},
			requestBody: `{"from_currency":"USD","to_currency":"EUR","amount":100}`,
			wantCode:    http.StatusBadRequest,
			wantBody:    map[string]interface{}{"error": "Insufficient funds or invalid currencies"},
		},
		{
			name: "internal_server_error",
			mockSetup: func() {
				tokener.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				tokener.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
				exchanger.EXPECT().
					Exchange(gomock.Any(), userID, "USD", "EUR", 100.0).
					Return(float32(0), nil, errors.New("db error"))
			},
			requestBody: `{"from_currency":"USD","to_currency":"EUR","amount":100}`,
			wantCode:    http.StatusInternalServerError,
			wantBody:    map[string]interface{}{"error": "Internal server error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			req := httptest.NewRequest(http.MethodPost, "/exchange", strings.NewReader(tt.requestBody))
			w := httptest.NewRecorder()

			handler(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantCode {
				t.Errorf("expected status %d, got %d", tt.wantCode, resp.StatusCode)
			}

			var body map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if !reflect.DeepEqual(body, tt.wantBody) {
				t.Errorf("expected body %v, got %v", tt.wantBody, body)
			}
		})
	}
}
