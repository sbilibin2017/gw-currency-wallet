package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

// NewDepositHandler handles depositing funds to user wallet
// @Summary Deposit funds
// @Description Add funds to user wallet
// @Tags wallet
// @Accept json
// @Produce json
// @Param request body models.DepositRequest true "Deposit Request"
// @Success 200 {object} models.DepositResponse
// @Failure 400 {object} models.DepositErrorResponse
// @Router /wallet/deposit [post]
// @Security Bearer
func NewDepositHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.DepositRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(models.DepositResponse{
			Message: "Account topped up successfully",
			NewBalance: models.CurrencyBalance{
				USD: 200.0,
				RUB: 5000.0,
				EUR: 50.0,
			},
		})
	}
}
