package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

// ExchangeRater defines the interface that the service must implement.
type ExchangeRater interface {
	GetExchangeRates(ctx context.Context) (*models.ExchangeRates, error)
}

// NewGetExchangeRatesHandler returns an HTTP handler for fetching currency exchange rates.
// @Summary Get exchange rates
// @Description Fetches current exchange rates for all supported currencies
// @Tags exchange
// @Produce json
// @Success 200 {object} models.ExchangeRatesResponse "Exchange rates"
// @Failure 500 {object} models.ExchangeRatesErrorResponse "Failed to retrieve exchange rates"
// @Failure 401 {object} models.ExchangeRatesErrorResponse "Unauthorized"
// @Router /exchange/rates [get]
// @Security BearerAuth
func NewGetExchangeRatesHandler(svc ExchangeRater, tokenGetter func(ctx context.Context) (string, bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, ok := tokenGetter(r.Context())
		if !ok || username == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(models.ExchangeRatesErrorResponse{
				Error: "unauthorized",
			})
			return
		}

		rates, err := svc.GetExchangeRates(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(models.ExchangeRatesErrorResponse{
				Error: "Failed to retrieve exchange rates",
			})
			return
		}

		resp := models.ExchangeRatesResponse{
			Rates: *rates,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
