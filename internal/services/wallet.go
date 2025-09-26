package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
	"github.com/segmentio/kafka-go"
)

var (
	// ErrInsufficientFunds is returned when a user tries to withdraw or exchange more than their balance.
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// WalletWriter defines methods for writing deposits and withdrawals.
type WalletWriter interface {
	SaveDeposit(ctx context.Context, userID uuid.UUID, amount float64, currency string) error  // Saves a deposit for a user
	SaveWithdraw(ctx context.Context, userID uuid.UUID, amount float64, currency string) error // Saves a withdrawal for a user
}

// WalletReader defines methods for reading user balances.
type WalletReader interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (map[string]float64, error) // Returns user balances by currency
}

// ExchangeRateReader retrieves exchange rates.
type ExchangeRateReader interface {
	GetExchangeRates(ctx context.Context) (map[string]float32, error)                                 // Returns current exchange rates
	GetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string) (float32, error) // Returns exchange rate for a currency pair
}

// ExchangeRateCacheReader caches exchange rates.
type ExchangeRateCacheReader interface {
	GetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string) (float32, error)    // Returns cached exchange rate
	SetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string, rate float32) error // Sets cached exchange rate
}

// KafkaWriter defines a Kafka writer abstraction.
type KafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error // Writes messages to Kafka
	Close() error                                                   // Closes the Kafka writer
}

// WalletService handles wallet operations and Kafka publishing.
type WalletService struct {
	writeRepo   WalletWriter
	readRepo    WalletReader
	rateRepo    ExchangeRateReader
	cacheRepo   ExchangeRateCacheReader
	kafkaWriter KafkaWriter
}

// NewWalletService creates a new WalletService.
func NewWalletService(
	writeRepo WalletWriter,
	readRepo WalletReader,
	rateRepo ExchangeRateReader,
	cacheRepo ExchangeRateCacheReader,
	kafkaWriter KafkaWriter,
) *WalletService {
	return &WalletService{
		writeRepo:   writeRepo,
		readRepo:    readRepo,
		rateRepo:    rateRepo,
		cacheRepo:   cacheRepo,
		kafkaWriter: kafkaWriter,
	}
}

// publishTransaction publishes a transaction to Kafka.
func (s *WalletService) publishTransaction(ctx context.Context, txn models.Transaction) {
	if s.kafkaWriter == nil {
		logger.Log.Warnw("Kafka writer not configured, skipping publishing", "transaction_id", txn.TransactionID)
		return
	}

	data, err := json.Marshal(txn)
	if err != nil {
		logger.Log.Errorw("Failed to marshal transaction for Kafka", "transaction_id", txn.TransactionID, "error", err)
		return
	}

	msg := kafka.Message{
		Key:   []byte(txn.TransactionID),
		Value: data,
	}

	if err := s.kafkaWriter.WriteMessages(ctx, msg); err != nil {
		logger.Log.Errorw("Failed to publish transaction to Kafka", "transaction_id", txn.TransactionID, "error", err)
	} else {
		logger.Log.Infow("Transaction published to Kafka", "transaction_id", txn.TransactionID, "amount", txn.Amount)
	}
}

// Deposit adds funds to a user's balance and publishes the transaction.
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

	txn := models.Transaction{
		TransactionID: uuid.NewString(),
		Timestamp:     time.Now().Unix(),
		Amount:        amount,
		UserID:        userID.String(),
		Operation:     "deposit",
	}
	s.publishTransaction(ctx, txn)

	return usd, rub, eur, nil
}

// Withdraw removes funds from a user's balance and publishes the transaction.
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

	txn := models.Transaction{
		TransactionID: uuid.NewString(),
		Timestamp:     time.Now().Unix(),
		Amount:        amount,
		UserID:        userID.String(),
		Operation:     "withdraw",
	}
	s.publishTransaction(ctx, txn)

	return usd, rub, eur, nil
}

// GetUserBalance returns the user's balance in all currencies.
func (s *WalletService) GetUserBalance(ctx context.Context, userID uuid.UUID) (usd, rub, eur float64, err error) {
	balances, err := s.readRepo.GetByUserID(ctx, userID)
	if err != nil {
		logger.Log.Errorw("failed to get user balances", "userID", userID, "error", err)
		return 0, 0, 0, err
	}
	usd, rub, eur = balances[models.USD], balances[models.RUB], balances[models.EUR]
	return usd, rub, eur, nil
}

// GetExchangeRates returns current exchange rates for USD, RUB, and EUR.
func (s *WalletService) GetExchangeRates(ctx context.Context) (usd, rub, eur float32, err error) {
	rates, err := s.rateRepo.GetExchangeRates(ctx)
	if err != nil {
		logger.Log.Errorw("failed to get exchange rates", "error", err)
		return 0, 0, 0, err
	}
	usd, rub, eur = rates[models.USD], rates[models.RUB], rates[models.EUR]
	return usd, rub, eur, nil
}

// Exchange performs currency exchange for a user and publishes the transaction.
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

	txn := models.Transaction{
		TransactionID: uuid.NewString(),
		Timestamp:     time.Now().Unix(),
		Amount:        amount,
		UserID:        userID.String(),
		Operation:     "exchange",
	}
	s.publishTransaction(ctx, txn)

	return exchangedAmount, usd, rub, eur, nil
}
