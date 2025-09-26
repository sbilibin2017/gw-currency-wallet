package facades

import (
	"context"
	"errors"
	"testing"

	pb "github.com/sbilibin2017/proto-exchange/exchange"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// --- Fake gRPC client ---
type fakeExchangeClient struct {
	rates           map[string]float32
	rateForCurrency float32
	err             error
}

func (f *fakeExchangeClient) GetExchangeRates(ctx context.Context, _ *pb.Empty, opts ...grpc.CallOption) (*pb.ExchangeRatesResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &pb.ExchangeRatesResponse{Rates: f.rates}, nil
}

func (f *fakeExchangeClient) GetExchangeRateForCurrency(ctx context.Context, req *pb.CurrencyRequest, opts ...grpc.CallOption) (*pb.ExchangeRateResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &pb.ExchangeRateResponse{FromCurrency: req.FromCurrency, ToCurrency: req.ToCurrency, Rate: f.rateForCurrency}, nil
}

// --- Tests ---
func TestGetExchangeRates(t *testing.T) {
	client := &fakeExchangeClient{
		rates: map[string]float32{
			"USD": 1.0,
			"EUR": 0.9,
		},
	}
	facade := NewExchangeRatesGRPCFacade(client)

	rates, err := facade.GetExchangeRates(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, map[string]float32{"USD": 1.0, "EUR": 0.9}, rates)
}

func TestGetExchangeRates_Error(t *testing.T) {
	client := &fakeExchangeClient{err: errors.New("grpc error")}
	facade := NewExchangeRatesGRPCFacade(client)

	rates, err := facade.GetExchangeRates(context.Background())
	assert.Error(t, err)
	assert.Nil(t, rates)
}

func TestGetExchangeRateForCurrency(t *testing.T) {
	client := &fakeExchangeClient{rateForCurrency: 1.2}
	facade := NewExchangeRatesGRPCFacade(client)

	rate, err := facade.GetExchangeRateForCurrency(context.Background(), "USD", "EUR")
	assert.NoError(t, err)
	assert.Equal(t, float32(1.2), rate)
}

func TestGetExchangeRateForCurrency_Error(t *testing.T) {
	client := &fakeExchangeClient{err: errors.New("grpc error")}
	facade := NewExchangeRatesGRPCFacade(client)

	rate, err := facade.GetExchangeRateForCurrency(context.Background(), "USD", "EUR")
	assert.Error(t, err)
	assert.Equal(t, float32(0), rate)
}
