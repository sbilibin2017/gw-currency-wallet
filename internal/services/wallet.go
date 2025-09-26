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

// Threshold для больших транзакций
const largeAmountThreshold = 30000

// WalletWriter определяет методы записи для депозитов и снятий.
type WalletWriter interface {
	SaveDeposit(ctx context.Context, userID uuid.UUID, amount float64, currency string) error
	SaveWithdraw(ctx context.Context, userID uuid.UUID, amount float64, currency string) error
}

// WalletReader определяет методы чтения балансов.
type WalletReader interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (map[string]float64, error)
}

// ExchangeRateReader получает курсы валют.
type ExchangeRateReader interface {
	GetExchangeRates(ctx context.Context) (map[string]float32, error)
	GetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string) (float32, error)
}

// ExchangeRateCacheReader кеширует курсы валют.
type ExchangeRateCacheReader interface {
	GetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string) (float32, error)
	SetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string, rate float32) error
}

// KafkaWriter определяет абстракцию для kafka.Writer
type KafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// Transaction описывает перевод средств
type Transaction struct {
	TransactionID string  `json:"transaction_id" bson:"transaction_id"`
	Timestamp     int64   `json:"timestamp" bson:"timestamp"`
	Amount        float64 `json:"amount" bson:"amount"`
	Sender        string  `json:"sender" bson:"sender"`
	Receiver      string  `json:"receiver" bson:"receiver"`
}

// WalletService содержит бизнес-логику кошелька и публикацию в Kafka
type WalletService struct {
	writeRepo   WalletWriter
	readRepo    WalletReader
	rateRepo    ExchangeRateReader
	cacheRepo   ExchangeRateCacheReader
	kafkaWriter KafkaWriter
}

// NewWalletService создаёт новый WalletService
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

// ----------------------- Internal Kafka Publishing -----------------------

// publishTransaction публикует крупную транзакцию в Kafka
func (s *WalletService) publishTransaction(ctx context.Context, txn Transaction) {
	if s.kafkaWriter == nil {
		logger.Log.Warnw("Kafka writer not configured, skipping publishing", "transaction_id", txn.TransactionID)
		return
	}

	if txn.Amount < largeAmountThreshold {
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

// ----------------------- Wallet Operations -----------------------

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

	// Публикация в Kafka
	txn := Transaction{
		TransactionID: uuid.NewString(),
		Timestamp:     time.Now().Unix(),
		Amount:        amount,
		Sender:        userID.String(),
		Receiver:      "deposit",
	}
	s.publishTransaction(ctx, txn)

	return usd, rub, eur, nil
}

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

	// Публикация в Kafka
	txn := Transaction{
		TransactionID: uuid.NewString(),
		Timestamp:     time.Now().Unix(),
		Amount:        amount,
		Sender:        userID.String(),
		Receiver:      "withdraw",
	}
	s.publishTransaction(ctx, txn)

	return usd, rub, eur, nil
}

func (s *WalletService) GetUserBalance(ctx context.Context, userID uuid.UUID) (usd, rub, eur float64, err error) {
	balances, err := s.readRepo.GetByUserID(ctx, userID)
	if err != nil {
		logger.Log.Errorw("failed to get user balances", "userID", userID, "error", err)
		return 0, 0, 0, err
	}
	usd, rub, eur = balances[models.USD], balances[models.RUB], balances[models.EUR]
	return usd, rub, eur, nil
}

func (s *WalletService) GetExchangeRates(ctx context.Context) (usd, rub, eur float32, err error) {
	rates, err := s.rateRepo.GetExchangeRates(ctx)
	if err != nil {
		logger.Log.Errorw("failed to get exchange rates", "error", err)
		return 0, 0, 0, err
	}
	usd, rub, eur = rates[models.USD], rates[models.RUB], rates[models.EUR]
	return usd, rub, eur, nil
}

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
