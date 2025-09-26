package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// WalletWriter defines persistence methods required for deposits & withdrawals.
type WalletWriter interface {
	SaveDeposit(ctx context.Context, userID uuid.UUID, amount float64, currency string) error
	SaveWithdraw(ctx context.Context, userID uuid.UUID, amount float64, currency string) error
}

// WalletReader defines required storage operations for balances.
type WalletReader interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (map[string]float64, error)
}

// ExchangeRateReader fetches current exchange rates.
type ExchangeRateReader interface {
	GetExchangeRates(ctx context.Context) (map[string]float32, error)
	GetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string) (float32, error)
}

// ExchangeRateCacheReader fetches cached exchange rates.
type ExchangeRateCacheReader interface {
	GetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string) (float32, error)
	SetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string, rate float32) error
}

// WalletService holds business logic for deposits, withdrawals, balances, and exchanges.
type WalletService struct {
	writeRepo WalletWriter
	readRepo  WalletReader
	rateRepo  ExchangeRateReader
	cacheRepo ExchangeRateCacheReader
}

// NewWalletService creates a new WalletService instance.
func NewWalletService(
	writeRepo WalletWriter,
	readRepo WalletReader,
	rateRepo ExchangeRateReader,
	cacheRepo ExchangeRateCacheReader,
) *WalletService {
	return &WalletService{
		writeRepo: writeRepo,
		readRepo:  readRepo,
		rateRepo:  rateRepo,
		cacheRepo: cacheRepo,
	}
}

// Deposit adds funds to the user’s balance and returns updated balances.
func (s *WalletService) Deposit(ctx context.Context, userID uuid.UUID, amount float64, currency string) (usd, rub, eur float64, err error) {
	if err := s.writeRepo.SaveDeposit(ctx, userID, amount, currency); err != nil {
		return 0, 0, 0, err
	}

	balances, err := s.readRepo.GetByUserID(ctx, userID)
	if err != nil {
		return 0, 0, 0, err
	}

	usd, rub, eur = balances[models.USD], balances[models.RUB], balances[models.EUR]
	return usd, rub, eur, nil
}

// Withdraw removes funds from the user’s balance and returns updated balances.
func (s *WalletService) Withdraw(ctx context.Context, userID uuid.UUID, amount float64, currency string) (usd, rub, eur float64, err error) {
	if err := s.writeRepo.SaveWithdraw(ctx, userID, amount, currency); err != nil {
		return 0, 0, 0, err
	}

	balances, err := s.readRepo.GetByUserID(ctx, userID)
	if err != nil {
		return 0, 0, 0, err
	}

	usd, rub, eur = balances[models.USD], balances[models.RUB], balances[models.EUR]
	return usd, rub, eur, nil
}

// GetUserBalance fetches a user’s balances in typed form.
func (s *WalletService) GetUserBalance(ctx context.Context, userID uuid.UUID) (usd, rub, eur float64, err error) {
	balances, err := s.readRepo.GetByUserID(ctx, userID)
	if err != nil {
		return 0, 0, 0, err
	}

	usd, rub, eur = balances[models.USD], balances[models.RUB], balances[models.EUR]
	return usd, rub, eur, nil
}

// GetExchangeRates fetches current exchange rates directly from the rate repository.
func (s *WalletService) GetExchangeRates(ctx context.Context) (usd, rub, eur float32, err error) {
	rates, err := s.rateRepo.GetExchangeRates(ctx)
	if err != nil {
		return 0, 0, 0, err
	}

	usd, rub, eur = rates[models.USD], rates[models.RUB], rates[models.EUR]
	return usd, rub, eur, nil
}

// Exchange performs a currency exchange using cached rates or external rates.
func (s *WalletService) Exchange(ctx context.Context, userID uuid.UUID, fromCurrency, toCurrency string, amount float64) (exchangedAmount float32, usd, rub, eur float64, err error) {
	rate, err := s.cacheRepo.GetExchangeRateForCurrency(ctx, fromCurrency, toCurrency)
	if err != nil {
		rate, err = s.rateRepo.GetExchangeRateForCurrency(ctx, fromCurrency, toCurrency)
		if err != nil {
			logger.Log.Error(err)
			return 0, 0, 0, 0, err
		}
		if err := s.cacheRepo.SetExchangeRateForCurrency(ctx, fromCurrency, toCurrency, rate); err != nil {
			logger.Log.Error(err)
		}
	}

	if err := s.writeRepo.SaveWithdraw(ctx, userID, amount, fromCurrency); err != nil {
		return 0, 0, 0, 0, ErrInsufficientFunds
	}

	exchangedAmount = float32(amount) * rate
	if err := s.writeRepo.SaveDeposit(ctx, userID, float64(exchangedAmount), toCurrency); err != nil {
		logger.Log.Error(err)
		return exchangedAmount, 0, 0, 0, err
	}

	balances, err := s.readRepo.GetByUserID(ctx, userID)
	if err != nil {
		logger.Log.Error(err)
		return exchangedAmount, 0, 0, 0, err
	}

	usd, rub, eur = balances[models.USD], balances[models.RUB], balances[models.EUR]
	return exchangedAmount, usd, rub, eur, nil
}
