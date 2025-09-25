package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/handlers"
	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
	"github.com/stretchr/testify/require"
)

func TestGetExchangeRatesHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockToken := handlers.NewMockExchangeRatesTokener(ctrl)
	mockReader := handlers.NewMockExchangeRatesReader(ctrl)

	handler := handlers.NewGetExchangeRatesHandler(mockReader, mockToken)

	tests := []struct {
		name      string
		mockSetup func()
		wantCode  int
		wantBody  interface{}
	}{
		{
			name: "success",
			mockSetup: func() {
				mockToken.EXPECT().
					GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return("token", nil)
				mockToken.EXPECT().
					GetClaims(gomock.Any(), "token").
					Return(&jwt.Claims{UserID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)
				mockReader.EXPECT().
					GetExchangeRates(gomock.Any()).
					Return(map[string]float32{
						models.USD: 1.0,
						models.RUB: 90.0,
						models.EUR: 0.85,
					}, nil)
			},
			wantCode: http.StatusOK,
			wantBody: map[string]interface{}{
				"rates": map[string]interface{}{
					"USD": float64(1.0),
					"RUB": float64(90.0),
					"EUR": float64(0.85),
				},
			},
		},
		{
			name: "unauthorized_no_token",
			mockSetup: func() {
				mockToken.EXPECT().
					GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return("", context.Canceled) // simulate error getting token
			},
			wantCode: http.StatusUnauthorized,
			wantBody: map[string]interface{}{"error": "Unauthorized"},
		},
		{
			name: "unauthorized_invalid_token",
			mockSetup: func() {
				mockToken.EXPECT().
					GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return("token", nil)
				mockToken.EXPECT().
					GetClaims(gomock.Any(), "token").
					Return(nil, context.Canceled)
			},
			wantCode: http.StatusUnauthorized,
			wantBody: map[string]interface{}{"error": "Unauthorized"},
		},
		{
			name: "internal_error_on_reader",
			mockSetup: func() {
				mockToken.EXPECT().
					GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return("token", nil)
				mockToken.EXPECT().
					GetClaims(gomock.Any(), "token").
					Return(&jwt.Claims{UserID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)
				mockReader.EXPECT().
					GetExchangeRates(gomock.Any()).
					Return(nil, context.Canceled)
			},
			wantCode: http.StatusInternalServerError,
			wantBody: map[string]interface{}{"error": "Failed to retrieve exchange rates"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			req := httptest.NewRequest(http.MethodGet, "/exchange/rates", nil)
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
