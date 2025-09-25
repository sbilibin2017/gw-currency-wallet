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
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestGetBalanceHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uuid.New()

	tests := []struct {
		name         string
		mockSetup    func(tk *MockBalanceTokener, wr *MockWalletReader)
		expectedCode int
		expectedBody map[string]interface{}
	}{
		{
			name: "success",
			mockSetup: func(tk *MockBalanceTokener, wr *MockWalletReader) {
				tk.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("valid_token", nil)
				tk.EXPECT().GetClaims(gomock.Any(), "valid_token").Return(&jwt.Claims{
					UserID: userID,
				}, nil)
				wr.EXPECT().GetByUserID(gomock.Any(), userID).Return(map[string]float64{
					models.USD: 100,
					models.RUB: 5000,
					models.EUR: 50,
				}, nil)
			},
			expectedCode: http.StatusOK,
			expectedBody: map[string]interface{}{
				"balance": map[string]interface{}{
					"USD": float64(100),
					"RUB": float64(5000),
					"EUR": float64(50),
				},
			},
		},
		{
			name: "unauthorized missing token",
			mockSetup: func(tk *MockBalanceTokener, wr *MockWalletReader) {
				tk.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("", errors.New("no token"))
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{"error": "Unauthorized"},
		},
		{
			name: "unauthorized invalid token",
			mockSetup: func(tk *MockBalanceTokener, wr *MockWalletReader) {
				tk.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("bad_token", nil)
				tk.EXPECT().GetClaims(gomock.Any(), "bad_token").Return(nil, errors.New("invalid claims"))
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{"error": "Unauthorized"},
		},
		{
			name: "internal server error",
			mockSetup: func(tk *MockBalanceTokener, wr *MockWalletReader) {
				tk.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("valid_token", nil)
				tk.EXPECT().GetClaims(gomock.Any(), "valid_token").Return(&jwt.Claims{UserID: userID}, nil)
				wr.EXPECT().GetByUserID(gomock.Any(), userID).Return(nil, errors.New("db error"))
			},
			expectedCode: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{"error": "Internal server error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := NewMockBalanceTokener(ctrl)
			wr := NewMockWalletReader(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(tk, wr)
			}

			handler := NewGetBalanceHandler(wr, tk)
			req := httptest.NewRequest(http.MethodGet, "/balance", nil)
			rr := httptest.NewRecorder()

			handler(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(rr.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody, resp)
		})
	}
}
