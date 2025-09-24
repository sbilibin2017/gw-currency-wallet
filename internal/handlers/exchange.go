package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

// Exchanger defines the interface that the service must implement.
type Exchanger interface {
	Exchange(ctx context.Context, username, fromCurrency, toCurrency string, amount float64) (exchangedAmount float64, newBalance *models.BalanceResponse, err error)
}

// NewExchangeHandler returns an HTTP handler for performing currency exchange.
// @Summary Exchange currency
// @Description Exchange funds from one currency to another. Checks user balance and updates it accordingly.
// @Tags exchange
// @Accept json
// @Produce json
// @Param request body models.ExchangeRequest true "Exchange Request"
// @Success 200 {object} models.ExchangeResponse "Exchange successful"
// @Failure 400 {object} models.ExchangeErrorResponse "Insufficient funds or invalid currencies"
// @Failure 401 {object} models.ExchangeErrorResponse "Unauthorized"
// @Router /exchange [post]
// @Security BearerAuth
func NewExchangeHandler(svc Exchanger, tokenGetter func(ctx context.Context) (string, bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, ok := tokenGetter(r.Context())
		if !ok || username == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(models.ExchangeErrorResponse{
				Error: "unauthorized",
			})
			return
		}

		var req models.ExchangeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(models.ExchangeErrorResponse{
				Error: "invalid request body",
			})
			return
		}

		exchangedAmount, newBalance, err := svc.Exchange(r.Context(), username, req.FromCurrency, req.ToCurrency, req.Amount)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(models.ExchangeErrorResponse{
				Error: err.Error(),
			})
			return
		}

		resp := models.ExchangeResponse{
			Message:         "Exchange successful",
			ExchangedAmount: exchangedAmount,
			NewBalance:      *newBalance.Balance,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
