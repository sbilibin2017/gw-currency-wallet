package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/handlers"
	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
	"github.com/sbilibin2017/gw-currency-wallet/internal/services"
)

func TestWithdrawHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockToken := handlers.NewMockWithdrawTokener(ctrl)
	mockSvc := handlers.NewMockWalletWithdrawWriter(ctrl)

	handler := handlers.NewWithdrawHandler(mockSvc, mockToken)

	userID := uuid.New()

	tests := []struct {
		name       string
		body       interface{}
		mockSetup  func()
		wantCode   int
		wantErrMsg string
	}{
		{
			name: "success withdraw USD",
			body: handlers.WithdrawRequest{
				Amount:   50,
				Currency: "USD",
			},
			mockSetup: func() {
				mockToken.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				mockToken.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
				mockSvc.EXPECT().SaveWithdraw(gomock.Any(), userID, 50.0, "USD").Return(map[string]float64{
					"USD": 150, "RUB": 5000, "EUR": 50,
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
			wantErrMsg: "invalid request body",
		},
		{
			name: "unauthorized no token",
			body: handlers.WithdrawRequest{
				Amount:   50,
				Currency: "USD",
			},
			mockSetup: func() {
				mockToken.EXPECT().
					GetTokenFromRequest(gomock.Any(), gomock.Any()).
					Return("", errors.New("unauthorized")) // just a generic error
			},
			wantCode:   http.StatusUnauthorized,
			wantErrMsg: "Unauthorized",
		},
		{
			name: "invalid amount",
			body: handlers.WithdrawRequest{
				Amount:   -10,
				Currency: "USD",
			},
			mockSetup: func() {
				mockToken.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				mockToken.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
			},
			wantCode:   http.StatusBadRequest,
			wantErrMsg: "Insufficient funds or invalid amount",
		},
		{
			name: "invalid currency",
			body: handlers.WithdrawRequest{
				Amount:   50,
				Currency: "BTC",
			},
			mockSetup: func() {
				mockToken.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				mockToken.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
			},
			wantCode:   http.StatusBadRequest,
			wantErrMsg: "Insufficient funds or invalid amount",
		},
		{
			name: "insufficient funds",
			body: handlers.WithdrawRequest{
				Amount:   1000,
				Currency: "USD",
			},
			mockSetup: func() {
				mockToken.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				mockToken.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
				mockSvc.EXPECT().SaveWithdraw(gomock.Any(), userID, 1000.0, "USD").Return(nil, services.ErrInsufficientFunds)
			},
			wantCode:   http.StatusBadRequest,
			wantErrMsg: "Insufficient funds or invalid amount",
		},
		{
			name: "internal server error from SaveWithdraw",
			body: handlers.WithdrawRequest{
				Amount:   50,
				Currency: "USD",
			},
			mockSetup: func() {
				mockToken.EXPECT().GetTokenFromRequest(gomock.Any(), gomock.Any()).Return("token", nil)
				mockToken.EXPECT().GetClaims(gomock.Any(), "token").Return(&jwt.Claims{UserID: userID}, nil)
				mockSvc.EXPECT().SaveWithdraw(gomock.Any(), userID, 50.0, "USD").Return(nil, http.ErrBodyNotAllowed)
			},
			wantCode:   http.StatusInternalServerError,
			wantErrMsg: "Internal server error",
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

			req := httptest.NewRequest(http.MethodPost, "/wallet/withdraw", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			handler(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.wantCode {
				t.Errorf("expected status %d, got %d", tt.wantCode, resp.StatusCode)
			}

			if tt.wantErrMsg != "" {
				var respBody handlers.WithdrawErrorResponse
				json.NewDecoder(resp.Body).Decode(&respBody)
				if respBody.Error != tt.wantErrMsg {
					t.Errorf("expected error message %q, got %q", tt.wantErrMsg, respBody.Error)
				}
			}
		})
	}
}
