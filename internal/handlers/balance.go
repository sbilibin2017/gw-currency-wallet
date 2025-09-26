package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
)

// BalanceTokener defines only the methods needed by this handler.
type BalanceTokener interface {
	GetTokenFromRequest(ctx context.Context, r *http.Request) (string, error)
	GetClaims(ctx context.Context, tokenString string) (*jwt.Claims, error)
}

// Balancer defines the interface that the service must implement.
type Balancer interface {
	GetUserBalance(
		ctx context.Context,
		userID uuid.UUID,
	) (usd, rub, eur float64, err error)
}

// CurrencyBalance represents balances for different currencies
// swagger:model CurrencyBalance
type CurrencyBalance struct {
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

// BalanceResponse represents a successful response with user balances
// swagger:model BalanceResponse
type BalanceResponse struct {
	// User balances
	Balance *CurrencyBalance `json:"balance"`
}

// BalanceErrorResponse represents an error response when fetching balance
// swagger:model BalanceErrorResponse
type BalanceErrorResponse struct {
	// Error message
	// default: Unauthorized
	Error string `json:"error"`
}

// NewGetBalanceHandler returns an HTTP handler for fetching user balances.
// @Summary Get user balance
// @Description Returns balances for all supported currencies
// @Tags wallet
// @Produce json
// @Success 200 {object} handlers.BalanceResponse "User balance"
// @Failure 401 {object} handlers.BalanceErrorResponse "Unauthorized"
// @Failure 500 {object} handlers.BalanceErrorResponse "Internal server error"
// @Router /balance [get]
// @Security BearerAuth
func NewGetBalanceHandler(
	balancer Balancer,
	tokenGetter BalanceTokener,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		tokenStr, err := tokenGetter.GetTokenFromRequest(ctx, r)
		if err != nil {
			logger.Log.Error("unauthorized balance request: missing or invalid token")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(BalanceErrorResponse{
				Error: "Unauthorized",
			})
			return
		}

		claims, err := tokenGetter.GetClaims(ctx, tokenStr)
		if err != nil {
			logger.Log.Errorw("failed to parse token claims", "error", err)
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(BalanceErrorResponse{
				Error: "Unauthorized",
			})
			return
		}

		usd, rub, eur, err := balancer.GetUserBalance(ctx, claims.UserID)
		if err != nil {
			logger.Log.Errorw("failed to get balance", "userID", claims.UserID, "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(BalanceErrorResponse{
				Error: "Internal server error",
			})
			return
		}

		resp := BalanceResponse{
			Balance: &CurrencyBalance{
				USD: usd,
				RUB: rub,
				EUR: eur,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}
