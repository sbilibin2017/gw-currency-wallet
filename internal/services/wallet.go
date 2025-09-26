package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

var (
	// ErrInsufficientFunds is returned when a user tries to withdraw or exchange more than their balance.
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// WalletWriter defines persistence methods required for deposits and withdrawals.
type WalletWriter interface {
	SaveDeposit(ctx context.Context, userID uuid.UUID, amount float64, currency string) error
	SaveWithdraw(ctx context.Context, userID uuid.UUID, amount float64, currency string) error
}

// WalletReader defines storage operations required to fetch user balances.
type WalletReader interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (map[string]float64, error)
}

// ExchangeRateReader fetches current exchange rates from an external source.
type ExchangeRateReader interface {
	GetExchangeRates(ctx context.Context) (map[string]float32, error)
	GetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string) (float32, error)
}

// ExchangeRateCacheReader handles cached exchange rates.
type ExchangeRateCacheReader interface {
	GetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string) (float32, error)
	SetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string, rate float32) error
}

// WalletService contains business logic for deposits, withdrawals, balances, and currency exchanges.
type WalletService struct {
	writeRepo WalletWriter
	readRepo  WalletReader
	rateRepo  ExchangeRateReader
	cacheRepo ExchangeRateCacheReader
}

// NewWalletService creates a new WalletService instance with required repositories.
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

// Deposit adds funds to the user's balance for the specified currency.
func (s *WalletService) Deposit(ctx context.Context, userID uuid.UUID, amount float64, currency string) (usd, rub, eur float64, err error) {
	if err := s.writeRepo.SaveDeposit(ctx, userID, amount, currency); err != nil {
		logger.Log.Errorw("failed to save deposit", "userID", userID, "amount", amount, "currency", currency, "error", err)
		return 0, 0, 0, err
	}

	balances, err := s.readRepo.GetByUserID(ctx, userID)
	if err != nil {
		logger.Log.Errorw("failed to get balances after deposit", "userID", userID, "error", err)
		return 0, 0, 0, err
	}

	usd, rub, eur = balances[models.USD], balances[models.RUB], balances[models.EUR]
	return usd, rub, eur, nil
}

// Withdraw removes funds from the user's balance for the specified currency.
func (s *WalletService) Withdraw(ctx context.Context, userID uuid.UUID, amount float64, currency string) (usd, rub, eur float64, err error) {
	if err := s.writeRepo.SaveWithdraw(ctx, userID, amount, currency); err != nil {
		logger.Log.Errorw("failed to save withdrawal", "userID", userID, "amount", amount, "currency", currency, "error", err)
		return 0, 0, 0, err
	}

	balances, err := s.readRepo.GetByUserID(ctx, userID)
	if err != nil {
		logger.Log.Errorw("failed to get balances after withdrawal", "userID", userID, "error", err)
		return 0, 0, 0, err
	}

	usd, rub, eur = balances[models.USD], balances[models.RUB], balances[models.EUR]
	return usd, rub, eur, nil
}

// GetUserBalance fetches the current balances for the given user.
func (s *WalletService) GetUserBalance(ctx context.Context, userID uuid.UUID) (usd, rub, eur float64, err error) {
	balances, err := s.readRepo.GetByUserID(ctx, userID)
	if err != nil {
		logger.Log.Errorw("failed to get user balances", "userID", userID, "error", err)
		return 0, 0, 0, err
	}

	usd, rub, eur = balances[models.USD], balances[models.RUB], balances[models.EUR]
	return usd, rub, eur, nil
}

// GetExchangeRates fetches current exchange rates for USD, RUB, and EUR.
func (s *WalletService) GetExchangeRates(ctx context.Context) (usd, rub, eur float32, err error) {
	rates, err := s.rateRepo.GetExchangeRates(ctx)
	if err != nil {
		logger.Log.Errorw("failed to get exchange rates", "error", err)
		return 0, 0, 0, err
	}

	usd, rub, eur = rates[models.USD], rates[models.RUB], rates[models.EUR]
	return usd, rub, eur, nil
}

// Exchange converts an amount from one currency to another using cached or live exchange rates.
func (s *WalletService) Exchange(ctx context.Context, userID uuid.UUID, fromCurrency, toCurrency string, amount float64) (exchangedAmount float32, usd, rub, eur float64, err error) {
	rate, err := s.cacheRepo.GetExchangeRateForCurrency(ctx, fromCurrency, toCurrency)
	if err != nil {
		rate, err = s.rateRepo.GetExchangeRateForCurrency(ctx, fromCurrency, toCurrency)
		if err != nil {
			logger.Log.Errorw("failed to get exchange rate", "from", fromCurrency, "to", toCurrency, "error", err)
			return 0, 0, 0, 0, err
		}

		if err := s.cacheRepo.SetExchangeRateForCurrency(ctx, fromCurrency, toCurrency, rate); err != nil {
			logger.Log.Errorw("failed to cache exchange rate", "from", fromCurrency, "to", toCurrency, "rate", rate, "error", err)
		}
	}

	if err := s.writeRepo.SaveWithdraw(ctx, userID, amount, fromCurrency); err != nil {
		logger.Log.Errorw("failed to withdraw for exchange", "userID", userID, "amount", amount, "currency", fromCurrency, "error", err)
		return 0, 0, 0, 0, ErrInsufficientFunds
	}

	exchangedAmount = float32(amount) * rate
	if err := s.writeRepo.SaveDeposit(ctx, userID, float64(exchangedAmount), toCurrency); err != nil {
		logger.Log.Errorw("failed to deposit exchanged amount", "userID", userID, "amount", exchangedAmount, "currency", toCurrency, "error", err)
		return exchangedAmount, 0, 0, 0, err
	}

	balances, err := s.readRepo.GetByUserID(ctx, userID)
	if err != nil {
		logger.Log.Errorw("failed to get balances after exchange", "userID", userID, "error", err)
		return exchangedAmount, 0, 0, 0, err
	}

	usd, rub, eur = balances[models.USD], balances[models.RUB], balances[models.EUR]
	return exchangedAmount, usd, rub, eur, nil
}
