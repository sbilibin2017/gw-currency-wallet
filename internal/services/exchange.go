package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
)

// ExchangeRateForCurrencyReader fetches current exchange rates from an external service
type ExchangeRateForCurrencyReader interface {
	GetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string) (float32, error)
}

// ExchangeRateForCurrencyCashReader fetches cached exchange rates
type ExchangeRateForCurrencyCashReader interface {
	GetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string) (float32, error)
	SetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string, rate float32) error
}

// WalletWriter defines wallet operations used by services.
type WalletWriter interface {
	SaveDeposit(ctx context.Context, userID uuid.UUID, amount float64, currency string) (float64, error)
	SaveWithdraw(ctx context.Context, userID uuid.UUID, amount float64, currency string) (float64, error)
}

// WalletReader defines wallet read operations used by services.
type WalletReader interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (map[string]float64, error)
}

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type ExchangeService struct {
	wallet       WalletWriter
	walletReader WalletReader
	reader       ExchangeRateForCurrencyReader
	cashReader   ExchangeRateForCurrencyCashReader
}

// NewExchangeService creates a new service instance
func NewExchangeService(
	wallet WalletWriter,
	walletReader WalletReader,
	reader ExchangeRateForCurrencyReader,
	cashReader ExchangeRateForCurrencyCashReader,
) *ExchangeService {
	return &ExchangeService{
		wallet:       wallet,
		walletReader: walletReader,
		reader:       reader,
		cashReader:   cashReader,
	}
}

// Exchange performs a currency exchange
func (svc *ExchangeService) Exchange(
	ctx context.Context,
	userID uuid.UUID,
	fromCurrency, toCurrency string,
	amount float64,
) (exchangedAmount float32, newBalance map[string]float64, err error) {
	rate, err := svc.cashReader.GetExchangeRateForCurrency(ctx, fromCurrency, toCurrency)
	if err != nil {
		rate, err = svc.reader.GetExchangeRateForCurrency(ctx, fromCurrency, toCurrency)
		if err != nil {
			logger.Log.Error(err)
			return 0, nil, err
		}
		if err := svc.cashReader.SetExchangeRateForCurrency(ctx, fromCurrency, toCurrency, rate); err != nil {
			logger.Log.Error(err)
		}
	}

	_, err = svc.wallet.SaveWithdraw(ctx, userID, amount, fromCurrency)
	if err != nil {
		return 0, nil, ErrInsufficientFunds
	}

	exchangedAmount = float32(amount) * rate
	_, err = svc.wallet.SaveDeposit(ctx, userID, float64(exchangedAmount), toCurrency)
	if err != nil {
		logger.Log.Error(err)
		return exchangedAmount, nil, err
	}

	newBalance, err = svc.walletReader.GetByUserID(ctx, userID)
	if err != nil {
		logger.Log.Error(err)
		return exchangedAmount, nil, err
	}

	return exchangedAmount, newBalance, nil
}
