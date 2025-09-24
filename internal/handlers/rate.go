package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
	pb "github.com/sbilibin2017/proto-exchange/exchange"
)

// NewGetRatesHandlerWithClient returns an HTTP handler that fetches exchange rates via gRPC.
// exchangeClient: a connected pb.ExchangeServiceClient
func NewGetRatesHandlerWithClient(exchangeClient pb.ExchangeServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx := r.Context()

		// Call gRPC GetExchangeRates
		resp, err := exchangeClient.GetExchangeRates(ctx, &pb.Empty{})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(models.RatesErrorResponse{
				Error: err.Error(),
			})
			return
		}

		// Convert gRPC map to internal models.Rates
		rates := models.Rates{
			USD: float64(resp.Rates["USD"]),
			RUB: float64(resp.Rates["RUB"]),
			EUR: float64(resp.Rates["EUR"]),
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(models.RatesResponse{
			Rates: rates,
		})
	}
}
