package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
	"github.com/sbilibin2017/gw-currency-wallet/internal/services"
)

// WithdrawTokener defines only the methods needed by this handler.
type WithdrawTokener interface {
	GetTokenFromRequest(ctx context.Context, r *http.Request) (string, error)
	GetClaims(ctx context.Context, tokenString string) (*jwt.Claims, error)
}

// WalletWithdrawWriter defines the interface that the service must implement.
type WalletWithdrawWriter interface {
	Withdraw(ctx context.Context, userID uuid.UUID, amount float64, currency string) (usd, rub, eur float64, err error)
}

// CurrencyBalanceAfterWithdraw represents balances for different currencies
// swagger:model CurrencyBalanceAfterWithdraw
type CurrencyBalanceAfterWithdraw struct {
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

// WithdrawRequest represents the JSON body for withdrawing funds
// swagger:model WithdrawRequest
type WithdrawRequest struct {
	// Amount to withdraw
	// required: true
	// default: 50.0
	Amount float64 `json:"amount"`

	// Currency
	// required: true
	// default: USD
	Currency string `json:"currency"`
}

// WithdrawResponse represents a successful withdrawal response
// swagger:model WithdrawResponse
type WithdrawResponse struct {
	// Success message
	// default: Withdrawal successful
	Message string `json:"message"`

	// New balance of the user
	NewBalance CurrencyBalanceAfterWithdraw `json:"new_balance"`
}

// WithdrawErrorResponse represents an error response for withdrawal
// swagger:model WithdrawErrorResponse
type WithdrawErrorResponse struct {
	// Error message
	// default: Insufficient funds or invalid amount
	Error string `json:"error"`
}

// NewWithdrawHandler returns an HTTP handler for withdrawing funds from user wallet.
// @Summary Withdraw funds
// @Description Withdraw funds from user wallet. Validates amount and currency. Checks for sufficient funds.
// @Tags wallet
// @Accept json
// @Produce json
// @Param request body handlers.WithdrawRequest true "Withdraw Request"
// @Success 200 {object} handlers.WithdrawResponse "Withdrawal successful"
// @Failure 400 {object} handlers.WithdrawErrorResponse "Insufficient funds or invalid amount"
// @Failure 401 {object} handlers.WithdrawErrorResponse "Unauthorized"
// @Router /wallet/withdraw [post]
// @Security BearerAuth
func NewWithdrawHandler(
	svc WalletWithdrawWriter,
	tokenGetter WithdrawTokener,
) http.HandlerFunc {
	validCurrencies := map[string]struct{}{
		"USD": {},
		"RUB": {},
		"EUR": {},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		tokenStr, err := tokenGetter.GetTokenFromRequest(ctx, r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(WithdrawErrorResponse{Error: "Unauthorized"})
			return
		}

		claims, err := tokenGetter.GetClaims(ctx, tokenStr)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(WithdrawErrorResponse{Error: "Unauthorized"})
			return
		}

		var req WithdrawRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(WithdrawErrorResponse{Error: "invalid request body"})
			return
		}

		if req.Amount <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(WithdrawErrorResponse{Error: "Insufficient funds or invalid amount"})
			return
		}
		if _, ok := validCurrencies[req.Currency]; !ok {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(WithdrawErrorResponse{Error: "Insufficient funds or invalid amount"})
			return
		}

		usd, rub, eur, err := svc.Withdraw(ctx, claims.UserID, req.Amount, req.Currency)
		if err != nil {
			switch err {
			case services.ErrInsufficientFunds:
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(WithdrawErrorResponse{Error: "Insufficient funds or invalid amount"})
			default:
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(WithdrawErrorResponse{Error: "Internal server error"})
			}
			return
		}

		newBalance := CurrencyBalanceAfterWithdraw{
			USD: usd,
			RUB: rub,
			EUR: eur,
		}

		resp := WithdrawResponse{
			Message:    "Withdrawal successful",
			NewBalance: newBalance,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}
