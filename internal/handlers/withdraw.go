package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

// Withdrawer defines the interface that the service must implement.
type Withdrawer interface {
	Withdraw(ctx context.Context, username string, amount float64, currency string) (*models.CurrencyBalance, error)
}

// NewWithdrawHandler returns an HTTP handler for withdrawing funds from user wallet.
// @Summary Withdraw funds
// @Description Withdraw funds from user wallet. Validates amount and currency. Checks for sufficient funds.
// @Tags wallet
// @Accept json
// @Produce json
// @Param request body models.WithdrawRequest true "Withdraw Request"
// @Success 200 {object} models.WithdrawResponse "Withdrawal successful"
// @Failure 400 {object} models.WithdrawErrorResponse "Insufficient funds or invalid amount"
// @Failure 401 {object} models.WithdrawErrorResponse "Unauthorized"
// @Router /wallet/withdraw [post]
// @Security BearerAuth
func NewWithdrawHandler(svc Withdrawer, tokenGetter func(ctx context.Context) (string, bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, ok := tokenGetter(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(models.WithdrawErrorResponse{
				Error: "unauthorized",
			})
			return
		}

		var req models.WithdrawRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(models.WithdrawErrorResponse{
				Error: "invalid request body",
			})
			return
		}

		newBalance, err := svc.Withdraw(r.Context(), username, req.Amount, req.Currency)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(models.WithdrawErrorResponse{
				Error: err.Error(),
			})
			return
		}

		resp := models.WithdrawResponse{
			Message:    "Withdrawal successful",
			NewBalance: *newBalance,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
