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

func TestWithdrawHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTokener := NewMockWithdrawTokener(ctrl)
	mockWriter := NewMockWalletWithdrawWriter(ctrl)

	userID := uuid.New()

	handler := NewWithdrawHandler(mockWriter, mockTokener)

	// Allow token extraction for all subtests
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
		mockWithdraw   func()
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "success",
			reqBody: WithdrawRequest{
				Amount:   50,
				Currency: "USD",
			},
			mockWithdraw: func() {
				mockWriter.EXPECT().
					Withdraw(gomock.Any(), userID, 50.0, "USD").
					Return(200.0, 5000.0, 50.0, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: WithdrawResponse{
				Message: "Withdrawal successful",
				NewBalance: CurrencyBalanceAfterWithdraw{
					USD: 200.0,
					RUB: 5000.0,
					EUR: 50.0,
				},
			},
		},
		{
			name:           "bad_request_invalid_amount",
			reqBody:        WithdrawRequest{Amount: -10, Currency: "USD"},
			mockWithdraw:   nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   WithdrawErrorResponse{Error: "Insufficient funds or invalid amount"},
		},
		{
			name:           "bad_request_invalid_json",
			reqBody:        `invalid-json`,
			mockWithdraw:   nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   WithdrawErrorResponse{Error: "invalid request body"},
		},
		{
			name: "insufficient_funds",
			reqBody: WithdrawRequest{
				Amount:   100,
				Currency: "USD",
			},
			mockWithdraw: func() {
				mockWriter.EXPECT().
					Withdraw(gomock.Any(), userID, 100.0, "USD").
					Return(100.0, 5000.0, 50.0, services.ErrInsufficientFunds)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   WithdrawErrorResponse{Error: "Insufficient funds or invalid amount"},
		},
		{
			name: "invalid_currency",
			reqBody: WithdrawRequest{
				Amount:   50,
				Currency: "ABC",
			},
			mockWithdraw:   nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   WithdrawErrorResponse{Error: "Insufficient funds or invalid amount"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockWithdraw != nil {
				tt.mockWithdraw()
			}

			var bodyBytes []byte
			switch v := tt.reqBody.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/wallet/withdraw", bytes.NewReader(bodyBytes))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Result().StatusCode)

			respBody := rec.Body.Bytes()
			switch expected := tt.expectedBody.(type) {
			case WithdrawResponse:
				var got WithdrawResponse
				err := json.Unmarshal(respBody, &got)
				assert.NoError(t, err)
				assert.Equal(t, expected, got)
			case WithdrawErrorResponse:
				var got WithdrawErrorResponse
				err := json.Unmarshal(respBody, &got)
				assert.NoError(t, err)
				assert.Equal(t, expected, got)
			}
		})
	}
}
