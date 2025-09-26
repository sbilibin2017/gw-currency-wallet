package services

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestWalletService_Deposit(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	writer := NewMockWalletWriter(ctrl)
	reader := NewMockWalletReader(ctrl)
	kafka := NewMockKafkaWriter(ctrl)

	// Успешный депозит
	writer.EXPECT().SaveDeposit(ctx, userID, 50000.0, models.USD).Return(nil)
	reader.EXPECT().GetByUserID(ctx, userID).Return(map[string]float64{
		models.USD: 50000,
		models.RUB: 0,
		models.EUR: 0,
	}, nil)
	kafka.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).Return(nil)

	svc := NewWalletService(writer, reader, nil, nil, kafka)
	usd, rub, eur, err := svc.Deposit(ctx, userID, 50000, models.USD)

	assert.NoError(t, err)
	assert.Equal(t, 50000.0, usd)
	assert.Equal(t, 0.0, rub)
	assert.Equal(t, 0.0, eur)
}

func TestWalletService_Withdraw(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	writer := NewMockWalletWriter(ctrl)
	reader := NewMockWalletReader(ctrl)
	kafka := NewMockKafkaWriter(ctrl)

	// Успешное снятие
	writer.EXPECT().SaveWithdraw(ctx, userID, 1000.0, models.USD).Return(nil)
	reader.EXPECT().GetByUserID(ctx, userID).Return(map[string]float64{
		models.USD: 4000,
		models.RUB: 0,
		models.EUR: 0,
	}, nil)
	kafka.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).Return(nil)

	svc := NewWalletService(writer, reader, nil, nil, kafka)
	usd, rub, eur, err := svc.Withdraw(ctx, userID, 1000, models.USD)

	assert.NoError(t, err)
	assert.Equal(t, 4000.0, usd)
	assert.Equal(t, 0.0, rub)
	assert.Equal(t, 0.0, eur)
}

func TestWalletService_Exchange_Errors(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWrite := NewMockWalletWriter(ctrl)
	mockRead := NewMockWalletReader(ctrl)
	mockRate := NewMockExchangeRateReader(ctrl)
	mockCache := NewMockExchangeRateCacheReader(ctrl)

	svc := NewWalletService(mockWrite, mockRead, mockRate, mockCache, nil)

	// 1. Ошибка получения курса
	mockCache.EXPECT().GetExchangeRateForCurrency(ctx, "USD", "EUR").Return(float32(0), errors.New("cache miss"))
	mockRate.EXPECT().GetExchangeRateForCurrency(ctx, "USD", "EUR").Return(float32(0), errors.New("rate fetch error"))
	_, _, _, _, err := svc.Exchange(ctx, userID, "USD", "EUR", 100)
	assert.EqualError(t, err, "rate fetch error")

	// 2. Ошибка списания
	mockCache.EXPECT().GetExchangeRateForCurrency(ctx, "USD", "EUR").Return(float32(0), errors.New("cache miss"))
	mockRate.EXPECT().GetExchangeRateForCurrency(ctx, "USD", "EUR").Return(float32(0.9), nil)
	mockCache.EXPECT().SetExchangeRateForCurrency(ctx, "USD", "EUR", float32(0.9)).Return(nil)
	mockWrite.EXPECT().SaveWithdraw(ctx, userID, 100.0, "USD").Return(errors.New("insufficient"))
	_, _, _, _, err = svc.Exchange(ctx, userID, "USD", "EUR", 100)
	assert.Equal(t, ErrInsufficientFunds, err)

	// 3. Ошибка депозита
	mockCache.EXPECT().GetExchangeRateForCurrency(ctx, "USD", "EUR").Return(float32(0), errors.New("cache miss"))
	mockRate.EXPECT().GetExchangeRateForCurrency(ctx, "USD", "EUR").Return(float32(0.9), nil)
	mockCache.EXPECT().SetExchangeRateForCurrency(ctx, "USD", "EUR", float32(0.9)).Return(nil)
	mockWrite.EXPECT().SaveWithdraw(ctx, userID, 100.0, "USD").Return(nil)
	mockWrite.EXPECT().SaveDeposit(ctx, userID, float64(90.0), "EUR").Return(errors.New("deposit error"))
	_, _, _, _, err = svc.Exchange(ctx, userID, "USD", "EUR", 100)
	assert.EqualError(t, err, "deposit error")

	// 4. Ошибка чтения баланса
	mockCache.EXPECT().GetExchangeRateForCurrency(ctx, "USD", "EUR").Return(float32(0), errors.New("cache miss"))
	mockRate.EXPECT().GetExchangeRateForCurrency(ctx, "USD", "EUR").Return(float32(0.9), nil)
	mockCache.EXPECT().SetExchangeRateForCurrency(ctx, "USD", "EUR", float32(0.9)).Return(nil)
	mockWrite.EXPECT().SaveWithdraw(ctx, userID, 100.0, "USD").Return(nil)
	mockWrite.EXPECT().SaveDeposit(ctx, userID, float64(90.0), "EUR").Return(nil)
	mockRead.EXPECT().GetByUserID(ctx, userID).Return(nil, errors.New("read balance error"))
	_, _, _, _, err = svc.Exchange(ctx, userID, "USD", "EUR", 100)
	assert.EqualError(t, err, "read balance error")
}

func TestWalletService_publishTransaction(t *testing.T) {
	ctx := context.Background()
	txn := models.Transaction{
		TransactionID: "txn-123",
		Amount:        1000,
		UserID:        "user-1",
		Operation:     "deposit",
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockKafka := NewMockKafkaWriter(ctrl)
	svc := &WalletService{kafkaWriter: mockKafka}

	// Проверяем успешный вызов
	mockKafka.EXPECT().WriteMessages(ctx, gomock.Any()).Return(nil).Times(1)
	svc.publishTransaction(ctx, txn)

	// Проверяем ошибку публикации
	mockKafka.EXPECT().WriteMessages(ctx, gomock.Any()).Return(errors.New("kafka error")).Times(1)
	svc.publishTransaction(ctx, txn)

	// Проверяем nil KafkaWriter — не должно паниковать
	svc = &WalletService{}
	svc.publishTransaction(ctx, txn)
}

func TestWalletService_GetUserBalance(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReader := NewMockWalletReader(ctrl)
	mockReader.EXPECT().GetByUserID(ctx, userID).Return(map[string]float64{
		models.USD: 100,
		models.RUB: 5000,
		models.EUR: 50,
	}, nil)

	svc := &WalletService{
		readRepo: mockReader,
	}

	usd, rub, eur, err := svc.GetUserBalance(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 100.0, usd)
	assert.Equal(t, 5000.0, rub)
	assert.Equal(t, 50.0, eur)
}

func TestWalletService_GetUserBalance_Error(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReader := NewMockWalletReader(ctrl)
	mockReader.EXPECT().GetByUserID(ctx, userID).Return(nil, errors.New("db error"))

	svc := &WalletService{
		readRepo: mockReader,
	}

	usd, rub, eur, err := svc.GetUserBalance(ctx, userID)
	assert.Error(t, err)
	assert.Equal(t, 0.0, usd)
	assert.Equal(t, 0.0, rub)
	assert.Equal(t, 0.0, eur)
}

func TestWalletService_GetExchangeRates(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRate := NewMockExchangeRateReader(ctrl)
	mockRate.EXPECT().GetExchangeRates(ctx).Return(map[string]float32{
		models.USD: 1.0,
		models.RUB: 95.0,
		models.EUR: 0.92,
	}, nil)

	svc := &WalletService{
		rateRepo: mockRate,
	}

	usd, rub, eur, err := svc.GetExchangeRates(ctx)
	assert.NoError(t, err)
	assert.Equal(t, float32(1.0), usd)
	assert.Equal(t, float32(95.0), rub)
	assert.Equal(t, float32(0.92), eur)
}

func TestWalletService_GetExchangeRates_Error(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRate := NewMockExchangeRateReader(ctrl)
	mockRate.EXPECT().GetExchangeRates(ctx).Return(nil, errors.New("fetch error"))

	svc := &WalletService{
		rateRepo: mockRate,
	}

	usd, rub, eur, err := svc.GetExchangeRates(ctx)
	assert.Error(t, err)
	assert.Equal(t, float32(0), usd)
	assert.Equal(t, float32(0), rub)
	assert.Equal(t, float32(0), eur)
}
