package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
)

// DepositTokener defines only the methods needed by this handler.
type DepositTokener interface {
	GetTokenFromRequest(ctx context.Context, r *http.Request) (string, error)
	GetClaims(ctx context.Context, tokenString string) (*jwt.Claims, error)
}

// DepositWriter defines the interface that the service must implement.
type DepositWriter interface {
	Deposit(ctx context.Context, userID uuid.UUID, amount float64, currency string) (usd, rub, eur float64, err error)
}

// CurrencyBalanceAfterDeposit represents balances for different currencies
// swagger:model CurrencyDeposit
type CurrencyBalanceAfterDeposit struct {
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

// DepositRequest represents the JSON body for depositing funds
// swagger:model DepositRequest
type DepositRequest struct {
	// Amount to deposit
	// required: true
	// default: 100.0
	Amount float64 `json:"amount"`

	// Currency
	// required: true
	// default: USD
	Currency string `json:"currency"`
}

// DepositResponse represents a successful deposit response
// swagger:model DepositResponse
type DepositResponse struct {
	// Success message
	// default: Account topped up successfully
	Message string `json:"message"`

	// New balance of the user
	NewBalance CurrencyBalanceAfterDeposit `json:"new_balance"`
}

// DepositErrorResponse represents an error response for deposit
// swagger:model DepositErrorResponse
type DepositErrorResponse struct {
	// Error message
	// default: Invalid amount or currency
	Error string `json:"error"`
}

// NewDepositHandler returns an HTTP handler for depositing funds into user wallet.
// @Summary Deposit funds
// @Description Add funds to user wallet. Validates amount and currency. Updates user balance in the database.
// @Tags wallet
// @Accept json
// @Produce json
// @Param request body handlers.DepositRequest true "Deposit Request"
// @Success 200 {object} handlers.DepositResponse "Account topped up successfully"
// @Failure 400 {object} handlers.DepositErrorResponse "Invalid amount or currency"
// @Failure 401 {object} handlers.DepositErrorResponse "Unauthorized"
// @Router /wallet/deposit [post]
// @Security BearerAuth
func NewDepositHandler(
	svc DepositWriter,
	tokenGetter DepositTokener,
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
			logger.Log.Errorw("failed to get token from request", "error", err)
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(DepositErrorResponse{Error: "Unauthorized"})
			return
		}

		claims, err := tokenGetter.GetClaims(ctx, tokenStr)
		if err != nil {
			logger.Log.Errorw("failed to get claims from token", "error", err)
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(DepositErrorResponse{Error: "Unauthorized"})
			return
		}

		var req DepositRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Log.Errorw("failed to decode deposit request", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DepositErrorResponse{Error: "Invalid request body"})
			return
		}

		if req.Amount <= 0 {
			logger.Log.Warnw("invalid deposit amount", "amount", req.Amount)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DepositErrorResponse{Error: "Invalid amount or currency"})
			return
		}
		if _, ok := validCurrencies[req.Currency]; !ok {
			logger.Log.Warnw("invalid deposit currency", "currency", req.Currency)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DepositErrorResponse{Error: "Invalid amount or currency"})
			return
		}

		usd, rub, eur, err := svc.Deposit(ctx, claims.UserID, req.Amount, req.Currency)
		if err != nil {
			logger.Log.Errorw("failed to deposit funds", "userID", claims.UserID, "amount", req.Amount, "currency", req.Currency, "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(DepositErrorResponse{Error: "Internal server error"})
			return
		}

		newBalance := CurrencyBalanceAfterDeposit{
			USD: usd,
			RUB: rub,
			EUR: eur,
		}

		resp := DepositResponse{
			Message:    "Account topped up successfully",
			NewBalance: newBalance,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}
