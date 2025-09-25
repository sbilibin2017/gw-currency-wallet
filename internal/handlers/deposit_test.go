package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/handlers"
	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
)

func TestDepositHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockToken := handlers.NewMockDepositTokener(ctrl)
	mockSvc := handlers.NewMockWalletDepositWriter(ctrl)

	handler := handlers.NewDepositHandler(mockSvc, mockToken)

	userID := uuid.New()

	tests := []struct {
		name       string
		body       interface{}
		mockSetup  func()
		wantCode   int
		wantErrMsg string
	}{
		{
			name: "success deposit USD",
			body: handlers.DepositRequest{
				Amount:   100,
				Currency: "USD",
			},
			mockSetup: func() {
				mockToken.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				mockToken.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
				mockSvc.EXPECT().SaveDeposit(gomock.Any(), userID, 100.0, "USD").Return(map[string]float64{
					"USD": 200, "RUB": 5000, "EUR": 50,
				}, nil)
			},
			wantCode:   http.StatusOK,
			wantErrMsg: "",
		},
		{
			name: "invalid JSON",
			body: "{invalid-json}",
			mockSetup: func() {
				mockToken.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				mockToken.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
			},
			wantCode:   http.StatusBadRequest,
			wantErrMsg: "Invalid request body",
		},
		{
			name: "unauthorized no token",
			body: handlers.DepositRequest{
				Amount:   100,
				Currency: "USD",
			},
			mockSetup: func() {
				mockToken.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("", http.ErrNoCookie)
			},
			wantCode:   http.StatusUnauthorized,
			wantErrMsg: "Unauthorized",
		},
		{
			name: "invalid amount",
			body: handlers.DepositRequest{
				Amount:   -10,
				Currency: "USD",
			},
			mockSetup: func() {
				mockToken.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				mockToken.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
			},
			wantCode:   http.StatusBadRequest,
			wantErrMsg: "Invalid amount or currency",
		},
		{
			name: "invalid currency",
			body: handlers.DepositRequest{
				Amount:   100,
				Currency: "BTC",
			},
			mockSetup: func() {
				mockToken.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				mockToken.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
			},
			wantCode:   http.StatusBadRequest,
			wantErrMsg: "Invalid amount or currency",
		},
		{
			name: "internal server error from SaveDeposit",
			body: handlers.DepositRequest{
				Amount:   100,
				Currency: "USD",
			},
			mockSetup: func() {
				mockToken.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				mockToken.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
				mockSvc.EXPECT().SaveDeposit(gomock.Any(), userID, 100.0, "USD").Return(nil, http.ErrBodyNotAllowed)
			},
			wantCode:   http.StatusInternalServerError,
			wantErrMsg: "Intrenal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/wallet/deposit", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			handler(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.wantCode {
				t.Errorf("expected status %d, got %d", tt.wantCode, resp.StatusCode)
			}

			if tt.wantErrMsg != "" {
				var respBody handlers.DepositErrorResponse
				json.NewDecoder(resp.Body).Decode(&respBody)
				if respBody.Error != tt.wantErrMsg {
					t.Errorf("expected error message %q, got %q", tt.wantErrMsg, respBody.Error)
				}
			}
		})
	}
}
