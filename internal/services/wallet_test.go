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
	txn := Transaction{
		TransactionID: "txn-123",
		Amount:        1000,
		Sender:        "user-1",
		Receiver:      "deposit",
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

	tests := []struct {
		name        string
		mockSetup   func(ctrl *gomock.Controller) WalletReader
		expectedUSD float64
		expectedRUB float64
		expectedEUR float64
		expectErr   bool
	}{
		{
			name: "successful fetch",
			mockSetup: func(ctrl *gomock.Controller) WalletReader {
				mockReader := NewMockWalletReader(ctrl)
				mockReader.EXPECT().GetByUserID(ctx, userID).Return(map[string]float64{
					models.USD: 120,
					models.RUB: 3000,
					models.EUR: 50,
				}, nil)
				return mockReader
			},
			expectedUSD: 120,
			expectedRUB: 3000,
			expectedEUR: 50,
			expectErr:   false,
		},
		{
			name: "read error",
			mockSetup: func(ctrl *gomock.Controller) WalletReader {
				mockReader := NewMockWalletReader(ctrl)
				mockReader.EXPECT().GetByUserID(ctx, userID).Return(nil, errors.New("db error"))
				return mockReader
			},
			expectedUSD: 0,
			expectedRUB: 0,
			expectedEUR: 0,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			reader := tt.mockSetup(ctrl)
			svc := &WalletService{
				readRepo: reader,
			}

			usd, rub, eur, err := svc.GetUserBalance(ctx, userID)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Equal(t, 0.0, usd)
				assert.Equal(t, 0.0, rub)
				assert.Equal(t, 0.0, eur)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUSD, usd)
				assert.Equal(t, tt.expectedRUB, rub)
				assert.Equal(t, tt.expectedEUR, eur)
			}
		})
	}
}

func TestWalletService_GetExchangeRates(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		mockSetup   func(ctrl *gomock.Controller) ExchangeRateReader
		expectedUSD float32
		expectedRUB float32
		expectedEUR float32
		expectErr   bool
	}{
		{
			name: "successful fetch",
			mockSetup: func(ctrl *gomock.Controller) ExchangeRateReader {
				mockRate := NewMockExchangeRateReader(ctrl)
				mockRate.EXPECT().GetExchangeRates(ctx).Return(map[string]float32{
					models.USD: 1.0,
					models.RUB: 90.0,
					models.EUR: 0.85,
				}, nil)
				return mockRate
			},
			expectedUSD: 1.0,
			expectedRUB: 90.0,
			expectedEUR: 0.85,
			expectErr:   false,
		},
		{
			name: "fetch error",
			mockSetup: func(ctrl *gomock.Controller) ExchangeRateReader {
				mockRate := NewMockExchangeRateReader(ctrl)
				mockRate.EXPECT().GetExchangeRates(ctx).Return(nil, errors.New("rate error"))
				return mockRate
			},
			expectedUSD: 0,
			expectedRUB: 0,
			expectedEUR: 0,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			rateReader := tt.mockSetup(ctrl)
			svc := &WalletService{
				rateRepo: rateReader,
			}

			usd, rub, eur, err := svc.GetExchangeRates(ctx)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Equal(t, float32(0), usd)
				assert.Equal(t, float32(0), rub)
				assert.Equal(t, float32(0), eur)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUSD, usd)
				assert.Equal(t, tt.expectedRUB, rub)
				assert.Equal(t, tt.expectedEUR, eur)
			}
		})
	}
}

func TestWalletService_Exchange(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWrite := NewMockWalletWriter(ctrl)
	mockRead := NewMockWalletReader(ctrl)
	mockRate := NewMockExchangeRateReader(ctrl)
	mockCache := NewMockExchangeRateCacheReader(ctrl)

	// Настройка кеша и курса
	mockCache.EXPECT().GetExchangeRateForCurrency(ctx, "USD", "EUR").Return(float32(0), errors.New("cache miss"))
	mockRate.EXPECT().GetExchangeRateForCurrency(ctx, "USD", "EUR").Return(float32(0.9), nil)
	mockCache.EXPECT().SetExchangeRateForCurrency(ctx, "USD", "EUR", float32(0.9)).Return(nil)

	// Списание и зачисление
	mockWrite.EXPECT().SaveWithdraw(ctx, userID, 100.0, "USD").Return(nil)
	mockWrite.EXPECT().SaveDeposit(ctx, userID, float64(90.0), "EUR").Return(nil)

	// Получение баланса
	mockRead.EXPECT().GetByUserID(ctx, userID).Return(map[string]float64{
		models.USD: 900,
		models.RUB: 0,
		models.EUR: 90,
	}, nil)

	svc := NewWalletService(mockWrite, mockRead, mockRate, mockCache, nil)
	exchanged, usd, rub, eur, err := svc.Exchange(ctx, userID, "USD", "EUR", 100)

	assert.NoError(t, err)
	assert.Equal(t, float32(90.0), exchanged)
	assert.Equal(t, 900.0, usd)
	assert.Equal(t, 0.0, rub)
	assert.Equal(t, 90.0, eur)
}
