package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

// NewGetBalanceHandler handles fetching user balance
// @Summary Get user balance
// @Description Returns balances for all supported currencies
// @Tags wallet
// @Produce json
// @Success 200 {object} models.BalanceResponse
// @Router /balance [get]
// @Security Bearer
func NewGetBalanceHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := models.BalanceResponse{
			Balance: models.CurrencyBalance{
				USD: 100.0,
				RUB: 5000.0,
				EUR: 50.0,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// RegisterGetBalanceHandler registers routes for fetching user balance
func RegisterGetBalanceHandler(r chi.Router, h http.HandlerFunc) {
	r.Get("/balance", h)
}
