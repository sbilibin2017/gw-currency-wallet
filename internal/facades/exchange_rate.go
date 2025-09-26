package facades

import (
	"context"

	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	pb "github.com/sbilibin2017/proto-exchange/exchange"
)

// ExchangeRatesGRPCFacade implements currency exchange readers using gRPC.
type ExchangeRatesGRPCFacade struct {
	client pb.ExchangeServiceClient
}

// NewExchangeRatesGRPCFacade creates a new facade with a gRPC client.
func NewExchangeRatesGRPCFacade(client pb.ExchangeServiceClient) *ExchangeRatesGRPCFacade {
	return &ExchangeRatesGRPCFacade{client: client}
}

// GetExchangeRates fetches all exchange rates and returns them as map[string]float32
func (f *ExchangeRatesGRPCFacade) GetExchangeRates(
	ctx context.Context,
) (map[string]float32, error) {
	resp, err := f.client.GetExchangeRates(ctx, &pb.Empty{})
	if err != nil {
		logger.Log.Errorw("failed to fetch exchange rates via gRPC", "error", err)
		return nil, err
	}

	rates := make(map[string]float32, len(resp.Rates))
	for currency, rate := range resp.Rates {
		rates[currency] = rate
	}

	return rates, nil
}

// GetExchangeRateForCurrency fetches exchange rate between two currencies
func (f *ExchangeRatesGRPCFacade) GetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string) (float32, error) {
	req := &pb.CurrencyRequest{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
	}

	resp, err := f.client.GetExchangeRateForCurrency(ctx, req)
	if err != nil {
		logger.Log.Errorw("failed to fetch exchange rate for currency via gRPC",
			"from", fromCurrency, "to", toCurrency, "error", err)
		return 0, err
	}

	return resp.Rate, nil
}
