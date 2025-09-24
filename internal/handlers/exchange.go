package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
	pb "github.com/sbilibin2017/proto-exchange/exchange"
)

// NewExchangeHandlerWithClient returns an HTTP handler that performs currency exchange using gRPC.
// exchangeClient: a connected pb.ExchangeServiceClient
func NewExchangeHandlerWithClient(exchangeClient pb.ExchangeServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.ExchangeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(models.ExchangeErrorResponse{
				Error: "invalid request body",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")

		ctx := r.Context()
		// Call gRPC GetExchangeRateForCurrency
		grpcResp, err := exchangeClient.GetExchangeRateForCurrency(ctx, &pb.CurrencyRequest{
			FromCurrency: req.FromCurrency,
			ToCurrency:   req.ToCurrency,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(models.ExchangeErrorResponse{
				Error: err.Error(),
			})
			return
		}

		// Calculate exchanged amount
		exchangedAmount := req.Amount * float64(grpcResp.Rate)

		// TODO: Update user balance in DB / Redis as needed
		// For now, we return a dummy new balance
		newBalance := models.CurrencyBalance{
			USD: 0.0,
			RUB: 0.0,
			EUR: 0.0,
		}
		switch req.ToCurrency {
		case "USD":
			newBalance.USD = exchangedAmount
		case "RUB":
			newBalance.RUB = exchangedAmount
		case "EUR":
			newBalance.EUR = exchangedAmount
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(models.ExchangeResponse{
			Message:         "Exchange successful",
			ExchangedAmount: exchangedAmount,
			NewBalance:      newBalance,
		})
	}
}
