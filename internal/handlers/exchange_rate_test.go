package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
)

func TestGetExchangeRatesHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uuid.New()
	validToken := "valid-token"

	tests := []struct {
		name               string
		setupMocks         func(*MockExchangeRatesReader, *MockExchangeRatesTokener)
		expectedStatusCode int
		expectedResponse   interface{}
	}{
		{
			name: "success",
			setupMocks: func(reader *MockExchangeRatesReader, tokener *MockExchangeRatesTokener) {
				tokener.EXPECT().
					GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return(validToken, nil)
				tokener.EXPECT().
					GetClaims(gomock.Any(), validToken).
					Return(&jwt.Claims{UserID: userID}, nil)
				reader.EXPECT().
					GetExchangeRates(gomock.Any()).
					Return(float32(1.0), float32(90.0), float32(0.85), nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedResponse: ExchangeRatesResponse{
				Rates: ExchangeRates{
					USD: 1.0,
					RUB: 90.0,
					EUR: 0.85,
				},
			},
		},
		{
			name: "unauthorized_token_error",
			setupMocks: func(reader *MockExchangeRatesReader, tokener *MockExchangeRatesTokener) {
				tokener.EXPECT().
					GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return("", errors.New("no token"))
			},
			expectedStatusCode: http.StatusUnauthorized,
			expectedResponse:   ExchangeRatesErrorResponse{Error: "Unauthorized"},
		},
		{
			name: "unauthorized_claims_error",
			setupMocks: func(reader *MockExchangeRatesReader, tokener *MockExchangeRatesTokener) {
				tokener.EXPECT().
					GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return(validToken, nil)
				tokener.EXPECT().
					GetClaims(gomock.Any(), validToken).
					Return(nil, errors.New("invalid claims"))
			},
			expectedStatusCode: http.StatusUnauthorized,
			expectedResponse:   ExchangeRatesErrorResponse{Error: "Unauthorized"},
		},
		{
			name: "internal_server_error",
			setupMocks: func(reader *MockExchangeRatesReader, tokener *MockExchangeRatesTokener) {
				tokener.EXPECT().
					GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return(validToken, nil)
				tokener.EXPECT().
					GetClaims(gomock.Any(), validToken).
					Return(&jwt.Claims{UserID: userID}, nil)
				reader.EXPECT().
					GetExchangeRates(gomock.Any()).
					Return(float32(0), float32(0), float32(0), errors.New("db error"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedResponse:   ExchangeRatesErrorResponse{Error: "Failed to retrieve exchange rates"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReader := NewMockExchangeRatesReader(ctrl)
			mockTokener := NewMockExchangeRatesTokener(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockReader, mockTokener)
			}

			handler := NewGetExchangeRatesHandler(mockReader, mockTokener)

			req := httptest.NewRequest(http.MethodGet, "/exchange/rates", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatusCode, rec.Code)

			if rec.Code == http.StatusOK {
				var got ExchangeRatesResponse
				err := json.NewDecoder(rec.Body).Decode(&got)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResponse, got)
			} else {
				var got ExchangeRatesErrorResponse
				err := json.NewDecoder(rec.Body).Decode(&got)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResponse, got)
			}
		})
	}
}
