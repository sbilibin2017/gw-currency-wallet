package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
	"github.com/sbilibin2017/gw-currency-wallet/internal/services"
)

// ExchangeRateForCurrencyTokener is responsible for extracting and validating JWT tokens
// for the currency exchange handler.
type ExchangeRateForCurrencyTokener interface {
	GetTokenFromRequest(ctx context.Context, r *http.Request) (string, error)
	GetClaims(ctx context.Context, tokenString string) (*jwt.Claims, error)
}

// Exchanger
type Exchanger interface {
	Exchange(
		ctx context.Context,
		userID uuid.UUID,
		fromCurrency, toCurrency string,
		amount float64,
	) (exchangedAmount float32, newBalance map[string]float64, err error)
}

// ExchangeRequest represents the JSON body for currency exchange
// swagger:model ExchangeRequest
type ExchangeRequest struct {
	// Source currency
	// required: true
	// default: USD
	FromCurrency string `json:"from_currency"`

	// Target currency
	// required: true
	// default: EUR
	ToCurrency string `json:"to_currency"`

	// Amount to exchange
	// required: true
	// default: 100.0
	Amount float64 `json:"amount"`
}

// ExchangedBalance represents balances for different currencies
// swagger:model ExchangedBalance
type ExchangedBalance struct {
	// Balance in USD
	// default: 100.0
	USD float64 `json:"USD"`

	// Balance in RUB
	// default: 5000.0
	RUB float64 `json:"RUB"`

	// Balance in EUR
	// default: 50.0
	EUR float64 `json:"EUR"`
}

// ExchangeResponse represents a successful currency exchange response
// swagger:model ExchangeResponse
type ExchangeResponse struct {
	// Success message
	// default: Exchange successful
	Message string `json:"message"`

	// Amount received after exchange
	// default: 85.0
	ExchangedAmount float64 `json:"exchanged_amount"`

	// New balance after exchange
	NewBalance ExchangedBalance `json:"new_balance"`
}

// ExchangeErrorResponse represents an error response for currency exchange
// swagger:model ExchangeErrorResponse
type ExchangeErrorResponse struct {
	// Error message
	// default: Insufficient funds or invalid currencies
	Error string `json:"error"`
}

// NewExchangeHandler handles currency exchange requests.
// @Summary Exchange currency
// @Description Exchange funds from one currency to another. Checks user balance and updates it accordingly.
// @Tags exchange
// @Accept json
// @Produce json
// @Param request body handlers.ExchangeRequest true "Exchange Request"
// @Success 200 {object} handlers.ExchangeResponse "Exchange successful"
// @Failure 400 {object} handlers.ExchangeErrorResponse "Insufficient funds or invalid currencies"
// @Failure 401 {object} handlers.ExchangeErrorResponse "Unauthorized"
// @Router /exchange [post]
// @Security BearerAuth
func NewExchangeHandler(
	tokener ExchangeRateForCurrencyTokener,
	exchanger Exchanger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		tokenStr, err := tokener.GetTokenFromRequest(ctx, r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ExchangeErrorResponse{Error: "unauthorized"})
			return
		}

		claims, err := tokener.GetClaims(ctx, tokenStr)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ExchangeErrorResponse{Error: "unauthorized"})
			return
		}
		userID := claims.UserID

		var req ExchangeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ExchangeErrorResponse{Error: "Insufficient funds or invalid currencies"})
			return
		}

		exchangedAmount, balancesMap, err := exchanger.Exchange(ctx, userID, req.FromCurrency, req.ToCurrency, req.Amount)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrInsufficientFunds):
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(ExchangeErrorResponse{Error: "Insufficient funds or invalid currencies"})
			default:
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ExchangeErrorResponse{Error: "Internal server error"})
			}
			return
		}

		newBalance := ExchangedBalance{
			USD: balancesMap[models.USD],
			RUB: balancesMap[models.RUB],
			EUR: balancesMap[models.EUR],
		}

		resp := ExchangeResponse{
			Message:         "Exchange successful",
			ExchangedAmount: float64(exchangedAmount),
			NewBalance:      newBalance,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}
