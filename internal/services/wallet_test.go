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

	tests := []struct {
		name        string
		currency    string
		amount      float64
		mockSetup   func(t *testing.T) (*WalletService, func())
		expectedUSD float64
		expectedRUB float64
		expectedEUR float64
		expectErr   bool
	}{
		{
			name:     "success USD deposit",
			currency: models.USD,
			amount:   100,
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				writer := NewMockWalletWriter(ctrl)
				reader := NewMockWalletReader(ctrl)

				writer.EXPECT().SaveDeposit(ctx, userID, 100.0, models.USD).Return(nil)
				reader.EXPECT().GetByUserID(ctx, userID).Return(map[string]float64{
					models.USD: 100,
					models.RUB: 0,
					models.EUR: 0,
				}, nil)

				svc := NewWalletService(writer, reader, nil, nil)
				return svc, func() { ctrl.Finish() }
			},
			expectedUSD: 100,
			expectedRUB: 0,
			expectedEUR: 0,
			expectErr:   false,
		},
		{
			name:     "deposit fails",
			currency: models.RUB,
			amount:   50,
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				writer := NewMockWalletWriter(ctrl)
				writer.EXPECT().SaveDeposit(ctx, userID, 50.0, models.RUB).Return(errors.New("db error"))
				svc := NewWalletService(writer, nil, nil, nil)
				return svc, func() { ctrl.Finish() }
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, cleanup := tt.mockSetup(t)
			defer cleanup()

			usd, rub, eur, err := svc.Deposit(ctx, userID, tt.amount, tt.currency)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUSD, usd)
				assert.Equal(t, tt.expectedRUB, rub)
				assert.Equal(t, tt.expectedEUR, eur)
			}
		})
	}
}

func TestWalletService_Withdraw(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	tests := []struct {
		name        string
		currency    string
		amount      float64
		mockSetup   func(t *testing.T) (*WalletService, func())
		expectedUSD float64
		expectedRUB float64
		expectedEUR float64
		expectErr   bool
	}{
		{
			name:     "success EUR withdrawal",
			currency: models.EUR,
			amount:   30,
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				writer := NewMockWalletWriter(ctrl)
				reader := NewMockWalletReader(ctrl)

				writer.EXPECT().SaveWithdraw(ctx, userID, 30.0, models.EUR).Return(nil)
				reader.EXPECT().GetByUserID(ctx, userID).Return(map[string]float64{
					models.USD: 0,
					models.RUB: 0,
					models.EUR: 70,
				}, nil)

				return NewWalletService(writer, reader, nil, nil), func() { ctrl.Finish() }
			},
			expectedUSD: 0,
			expectedRUB: 0,
			expectedEUR: 70,
			expectErr:   false,
		},
		{
			name:     "withdraw fails",
			currency: models.USD,
			amount:   50,
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				writer := NewMockWalletWriter(ctrl)
				writer.EXPECT().SaveWithdraw(ctx, userID, 50.0, models.USD).Return(errors.New("insufficient funds"))
				return NewWalletService(writer, nil, nil, nil), func() { ctrl.Finish() }
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, cleanup := tt.mockSetup(t)
			defer cleanup()

			usd, rub, eur, err := svc.Withdraw(ctx, userID, tt.amount, tt.currency)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUSD, usd)
				assert.Equal(t, tt.expectedRUB, rub)
				assert.Equal(t, tt.expectedEUR, eur)
			}
		})
	}
}

func TestWalletService_GetUserBalance(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	tests := []struct {
		name        string
		mockSetup   func(t *testing.T) (*WalletService, func())
		expectedUSD float64
		expectedRUB float64
		expectedEUR float64
		expectErr   bool
	}{
		{
			name: "success balance fetch",
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				reader := NewMockWalletReader(ctrl)
				reader.EXPECT().GetByUserID(ctx, userID).Return(map[string]float64{
					models.USD: 120,
					models.RUB: 3000,
					models.EUR: 50,
				}, nil)
				return NewWalletService(nil, reader, nil, nil), func() { ctrl.Finish() }
			},
			expectedUSD: 120,
			expectedRUB: 3000,
			expectedEUR: 50,
			expectErr:   false,
		},
		{
			name: "fetch fails",
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				reader := NewMockWalletReader(ctrl)
				reader.EXPECT().GetByUserID(ctx, userID).Return(nil, errors.New("db error"))
				return NewWalletService(nil, reader, nil, nil), func() { ctrl.Finish() }
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, cleanup := tt.mockSetup(t)
			defer cleanup()

			usd, rub, eur, err := svc.GetUserBalance(ctx, userID)
			if tt.expectErr {
				assert.Error(t, err)
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

	tests := []struct {
		name          string
		fromCurrency  string
		toCurrency    string
		amount        float64
		mockSetup     func(t *testing.T) (*WalletService, func())
		expectedUSD   float64
		expectedRUB   float64
		expectedEUR   float64
		expectedExchg float32
		expectErr     bool
	}{
		{
			name:         "success cache hit",
			fromCurrency: models.USD,
			toCurrency:   models.EUR,
			amount:       100,
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				writer := NewMockWalletWriter(ctrl)
				reader := NewMockWalletReader(ctrl)
				cache := NewMockExchangeRateCacheReader(ctrl)

				cache.EXPECT().GetExchangeRateForCurrency(ctx, models.USD, models.EUR).Return(float32(0.9), nil)
				writer.EXPECT().SaveWithdraw(ctx, userID, 100.0, models.USD).Return(nil)
				writer.EXPECT().SaveDeposit(ctx, userID, 90.0, models.EUR).Return(nil)
				reader.EXPECT().GetByUserID(ctx, userID).Return(map[string]float64{
					models.USD: 0,
					models.RUB: 0,
					models.EUR: 90,
				}, nil)

				return NewWalletService(writer, reader, nil, cache), func() { ctrl.Finish() }
			},
			expectedExchg: 90,
			expectedUSD:   0,
			expectedRUB:   0,
			expectedEUR:   90,
			expectErr:     false,
		},
		{
			name:         "cache miss and external rate fetch fails",
			fromCurrency: models.USD,
			toCurrency:   models.RUB,
			amount:       50,
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				cache := NewMockExchangeRateCacheReader(ctrl)
				rateRepo := NewMockExchangeRateReader(ctrl)

				cache.EXPECT().GetExchangeRateForCurrency(ctx, models.USD, models.RUB).Return(float32(0), errors.New("cache miss"))
				rateRepo.EXPECT().GetExchangeRateForCurrency(ctx, models.USD, models.RUB).Return(float32(0), errors.New("external rate error"))

				return NewWalletService(nil, nil, rateRepo, cache), func() { ctrl.Finish() }
			},
			expectErr: true,
		},
		{
			name:         "cache miss but cache set fails",
			fromCurrency: models.USD,
			toCurrency:   models.EUR,
			amount:       100,
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				writer := NewMockWalletWriter(ctrl)
				reader := NewMockWalletReader(ctrl)
				cache := NewMockExchangeRateCacheReader(ctrl)
				rateRepo := NewMockExchangeRateReader(ctrl)

				cache.EXPECT().GetExchangeRateForCurrency(ctx, models.USD, models.EUR).Return(float32(0), errors.New("cache miss"))
				rateRepo.EXPECT().GetExchangeRateForCurrency(ctx, models.USD, models.EUR).Return(float32(0.9), nil)
				cache.EXPECT().SetExchangeRateForCurrency(ctx, models.USD, models.EUR, float32(0.9)).Return(errors.New("cache set error"))

				writer.EXPECT().SaveWithdraw(ctx, userID, 100.0, models.USD).Return(nil)
				writer.EXPECT().SaveDeposit(ctx, userID, 90.0, models.EUR).Return(nil)
				reader.EXPECT().GetByUserID(ctx, userID).Return(map[string]float64{
					models.USD: 0,
					models.RUB: 0,
					models.EUR: 90,
				}, nil)

				return NewWalletService(writer, reader, rateRepo, cache), func() { ctrl.Finish() }
			},
			expectedExchg: 90,
			expectedUSD:   0,
			expectedRUB:   0,
			expectedEUR:   90,
			expectErr:     false,
		},
		{
			name:         "withdraw fails",
			fromCurrency: models.RUB,
			toCurrency:   models.USD,
			amount:       50,
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				writer := NewMockWalletWriter(ctrl)
				cache := NewMockExchangeRateCacheReader(ctrl)

				cache.EXPECT().GetExchangeRateForCurrency(ctx, models.RUB, models.USD).Return(float32(0.013), nil)
				writer.EXPECT().SaveWithdraw(ctx, userID, 50.0, models.RUB).Return(errors.New("insufficient funds"))

				return NewWalletService(writer, nil, nil, cache), func() { ctrl.Finish() }
			},
			expectErr: true,
		},
		{
			name:         "deposit fails",
			fromCurrency: models.USD,
			toCurrency:   models.RUB,
			amount:       100,
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				writer := NewMockWalletWriter(ctrl)
				reader := NewMockWalletReader(ctrl)
				cache := NewMockExchangeRateCacheReader(ctrl)

				cache.EXPECT().GetExchangeRateForCurrency(ctx, models.USD, models.RUB).Return(float32(0.75), nil)
				writer.EXPECT().SaveWithdraw(ctx, userID, 100.0, models.USD).Return(nil)
				writer.EXPECT().SaveDeposit(ctx, userID, 75.0, models.RUB).Return(errors.New("deposit failed"))

				return NewWalletService(writer, reader, nil, cache), func() { ctrl.Finish() }
			},
			expectErr: true,
		},
		{
			name:         "balance fetch fails",
			fromCurrency: models.USD,
			toCurrency:   models.RUB,
			amount:       100,
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				writer := NewMockWalletWriter(ctrl)
				reader := NewMockWalletReader(ctrl)
				cache := NewMockExchangeRateCacheReader(ctrl)

				cache.EXPECT().GetExchangeRateForCurrency(ctx, models.USD, models.RUB).Return(float32(0.75), nil)
				writer.EXPECT().SaveWithdraw(ctx, userID, 100.0, models.USD).Return(nil)
				writer.EXPECT().SaveDeposit(ctx, userID, 75.0, models.RUB).Return(nil)
				reader.EXPECT().GetByUserID(ctx, userID).Return(nil, errors.New("db error"))

				return NewWalletService(writer, reader, nil, cache), func() { ctrl.Finish() }
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, cleanup := tt.mockSetup(t)
			defer cleanup()

			exchg, usd, rub, eur, err := svc.Exchange(ctx, userID, tt.fromCurrency, tt.toCurrency, tt.amount)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedExchg, exchg)
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
		mockSetup   func(t *testing.T) (*WalletService, func())
		expectedUSD float32
		expectedRUB float32
		expectedEUR float32
		expectErr   bool
	}{
		{
			name: "successfully fetch exchange rates",
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				rateRepo := NewMockExchangeRateReader(ctrl)

				rateRepo.EXPECT().GetExchangeRates(ctx).Return(map[string]float32{
					models.USD: 1.0,
					models.RUB: 90.0,
					models.EUR: 0.85,
				}, nil)

				svc := NewWalletService(nil, nil, rateRepo, nil)
				return svc, func() { ctrl.Finish() }
			},
			expectedUSD: 1.0,
			expectedRUB: 90.0,
			expectedEUR: 0.85,
			expectErr:   false,
		},
		{
			name: "fail to fetch exchange rates",
			mockSetup: func(t *testing.T) (*WalletService, func()) {
				ctrl := gomock.NewController(t)
				rateRepo := NewMockExchangeRateReader(ctrl)

				rateRepo.EXPECT().GetExchangeRates(ctx).Return(nil, errors.New("rate fetch error"))

				svc := NewWalletService(nil, nil, rateRepo, nil)
				return svc, func() { ctrl.Finish() }
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, cleanup := tt.mockSetup(t)
			defer cleanup()

			usd, rub, eur, err := svc.GetExchangeRates(ctx)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUSD, usd)
				assert.Equal(t, tt.expectedRUB, rub)
				assert.Equal(t, tt.expectedEUR, eur)
			}
		})
	}
}
