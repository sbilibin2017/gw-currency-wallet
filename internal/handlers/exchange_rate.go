package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

// ExchangeRatesTokener defines only the methods needed by this handler.
type ExchangeRatesTokener interface {
	GetTokenFromRequest(ctx context.Context, r *http.Request) (string, error)
	GetClaims(ctx context.Context, tokenString string) (*jwt.Claims, error)
}

// ExchangeRatesReader defines the interface for fetching exchange rates.
type ExchangeRatesReader interface {
	GetExchangeRates(ctx context.Context) (map[string]float32, error)
}

// ExchangeRates represents exchange rates for supported currencies
// swagger:model ExchangeRates
type ExchangeRates struct {
	// USD exchange rate
	// default: 1.0
	USD float32 `json:"USD"`

	// RUB exchange rate
	// default: 90.0
	RUB float32 `json:"RUB"`

	// EUR exchange rate
	// default: 0.85
	EUR float32 `json:"EUR"`
}

// ExchangeRatesResponse represents a successful response with exchange rates
// swagger:model ExchangeRatesResponse
type ExchangeRatesResponse struct {
	// Exchange rates
	Rates ExchangeRates `json:"rates"`
}

// ExchangeRatesErrorResponse represents an error response when fetching exchange rates
// swagger:model ExchangeRatesErrorResponse
type ExchangeRatesErrorResponse struct {
	// Error message
	// default: Failed to retrieve exchange rates
	Error string `json:"error"`
}

// NewGetExchangeRatesHandler returns an HTTP handler for fetching currency exchange rates.
// @Summary Get exchange rates
// @Description Fetches current exchange rates for all supported currencies
// @Tags exchange
// @Produce json
// @Success 200 {object} ExchangeRatesResponse "Exchange rates"
// @Failure 500 {object} ExchangeRatesErrorResponse "Failed to retrieve exchange rates"
// @Failure 401 {object} ExchangeRatesErrorResponse "Unauthorized"
// @Router /exchange/rates [get]
// @Security BearerAuth
func NewGetExchangeRatesHandler(
	reader ExchangeRatesReader,
	tokenGetter ExchangeRatesTokener,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		tokenStr, err := tokenGetter.GetTokenFromRequest(ctx, r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ExchangeRatesErrorResponse{Error: "Unauthorized"})
			return
		}

		_, err = tokenGetter.GetClaims(ctx, tokenStr)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ExchangeRatesErrorResponse{Error: "Unauthorized"})
			return
		}

		ratesMap, err := reader.GetExchangeRates(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ExchangeRatesErrorResponse{Error: "Failed to retrieve exchange rates"})
			return
		}

		getRate := func(currency string) float32 {
			if val, ok := ratesMap[currency]; ok {
				return val
			}
			return 0.0
		}

		resp := ExchangeRatesResponse{
			Rates: ExchangeRates{
				USD: getRate(models.USD),
				RUB: getRate(models.RUB),
				EUR: getRate(models.EUR),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}
