package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

// NewExchangeHandler handles currency exchange
// @Summary Exchange currencies
// @Description Exchange funds from one currency to another
// @Tags exchange
// @Accept json
// @Produce json
// @Param request body models.ExchangeRequest true "Exchange Request"
// @Success 200 {object} models.ExchangeResponse
// @Failure 400 {object} models.ExchangeErrorResponse
// @Router /exchange [post]
// @Security Bearer
func NewExchangeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.ExchangeRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(models.ExchangeResponse{
			Message:         "Exchange successful",
			ExchangedAmount: 85.0,
			NewBalance: models.CurrencyBalance{
				USD: 0.0,
				RUB: 0.0,
				EUR: 85.0,
			},
		})
	}
}

// RegisterExchangeHandler registers routes for currency exchange
func RegisterExchangeHandler(r chi.Router, h http.HandlerFunc) {
	r.Post("/exchange", h)
}
