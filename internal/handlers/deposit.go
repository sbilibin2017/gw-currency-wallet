package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

// Depositor defines the interface that the service must implement.
type Depositor interface {
	Deposit(ctx context.Context, username string, amount float64, currency string) (*models.CurrencyBalance, error)
}

// NewDepositHandler returns an HTTP handler for depositing funds into user wallet.
// @Summary Deposit funds
// @Description Add funds to user wallet. Validates amount and currency. Updates user balance in the database.
// @Tags wallet
// @Accept json
// @Produce json
// @Param request body models.DepositRequest true "Deposit Request"
// @Success 200 {object} models.DepositResponse "Account topped up successfully"
// @Failure 400 {object} models.DepositErrorResponse "Invalid amount or currency"
// @Failure 401 {object} models.DepositErrorResponse "Unauthorized"
// @Router /wallet/deposit [post]
// @Security BearerAuth
func NewDepositHandler(svc Depositor, tokenGetter func(ctx context.Context) (string, bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, ok := tokenGetter(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(models.DepositErrorResponse{
				Error: "unauthorized",
			})
			return
		}

		var req models.DepositRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(models.DepositErrorResponse{
				Error: "invalid request body",
			})
			return
		}

		newBalance, err := svc.Deposit(r.Context(), username, req.Amount, req.Currency)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(models.DepositErrorResponse{
				Error: err.Error(),
			})
			return
		}

		resp := models.DepositResponse{
			Message:    "Account topped up successfully",
			NewBalance: *newBalance,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
