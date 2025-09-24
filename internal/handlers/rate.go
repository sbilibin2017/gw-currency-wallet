package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

// NewGetRatesHandler handles fetching current exchange rates
// @Summary Get exchange rates
// @Description Returns current exchange rates for supported currencies
// @Tags exchange
// @Produce json
// @Success 200 {object} models.RatesResponse
// @Failure 500 {object} models.RatesErrorResponse
// @Router /exchange/rates [get]
// @Security Bearer
func NewGetRatesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(models.RatesResponse{
			Rates: models.Rates{
				USD: 1.0,
				RUB: 90.0,
				EUR: 0.85,
			},
		})
	}
}

// RegisterGetRatesHandler registers routes for fetching exchange rates
func RegisterGetRatesHandler(r chi.Router, h http.HandlerFunc) {
	r.Get("/exchange/rates", h)
}
