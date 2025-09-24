package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

// Balancer defines the interface that the service must implement.
type Balancer interface {
	GetBalance(ctx context.Context, username string) (*models.CurrencyBalance, error)
}

// NewGetBalanceHandler returns an HTTP handler for fetching user balances.
// @Summary Get user balance
// @Description Returns balances for all supported currencies
// @Tags wallet
// @Produce json
// @Success 200 {object} models.BalanceResponse "User balance"
// @Failure 401 {object} models.BalanceErrorResponse "Unauthorized"
// @Router /balance [get]
// @Security BearerAuth
func NewGetBalanceHandler(svc Balancer, tokenGetter func(ctx context.Context) (string, bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, ok := tokenGetter(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(models.BalanceErrorResponse{
				Error: "unauthorized",
			})
			return
		}

		balance, err := svc.GetBalance(r.Context(), username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(models.BalanceErrorResponse{
				Error: err.Error(),
			})
			return
		}

		resp := models.BalanceResponse{
			Balance: balance,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
