package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

// NewWithdrawHandler handles withdrawing funds from user wallet
// @Summary Withdraw funds
// @Description Withdraw funds from user wallet
// @Tags wallet
// @Accept json
// @Produce json
// @Param request body models.WithdrawRequest true "Withdraw Request"
// @Success 200 {object} models.WithdrawResponse
// @Failure 400 {object} models.WithdrawErrorResponse
// @Router /wallet/withdraw [post]
// @Security Bearer
func NewWithdrawHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.WithdrawRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(models.WithdrawResponse{
			Message: "Withdrawal successful",
			NewBalance: models.CurrencyBalance{
				USD: 150.0,
				RUB: 4800.0,
				EUR: 50.0,
			},
		})
	}
}

// RegisterWithdrawHandler registers routes for withdrawing funds
func RegisterWithdrawHandler(r chi.Router, h http.HandlerFunc) {
	r.Post("/wallet/withdraw", h)
}
