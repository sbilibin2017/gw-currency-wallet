package services

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestExchangeService_Exchange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := uuid.New()

	type args struct {
		fromCurrency string
		toCurrency   string
		amount       float64
	}

	tests := []struct {
		name            string
		args            args
		mockSetup       func() *ExchangeService
		expectedAmount  float32
		expectedErr     error
		expectedBalance map[string]float64
	}{
		{
			name: "success_using_cached_rate",
			args: args{"USD", "EUR", 100.0},
			mockSetup: func() *ExchangeService {
				mockWalletWriter := NewMockWalletWriter(ctrl)
				mockWalletReader := NewMockWalletReader(ctrl)
				mockReader := NewMockExchangeRateForCurrencyReader(ctrl)
				mockCash := NewMockExchangeRateForCurrencyCashReader(ctrl)

				// cache hit
				mockCash.EXPECT().
					GetExchangeRateForCurrency(ctx, "USD", "EUR").
					Return(float32(0.9), nil)

				mockWalletWriter.EXPECT().
					SaveWithdraw(ctx, userID, 100.0, "USD").
					Return(0.0, nil)

				mockWalletWriter.EXPECT().
					SaveDeposit(ctx, userID, 90.0, "EUR").
					Return(90.0, nil)

				mockWalletReader.EXPECT().
					GetByUserID(ctx, userID).
					Return(map[string]float64{"USD": 900.0, "EUR": 90.0}, nil)

				return NewExchangeService(mockWalletWriter, mockWalletReader, mockReader, mockCash)
			},
			expectedAmount:  90.0,
			expectedErr:     nil,
			expectedBalance: map[string]float64{"USD": 900.0, "EUR": 90.0},
		},
		{
			name: "external_rate_failure",
			args: args{"USD", "EUR", 100.0},
			mockSetup: func() *ExchangeService {
				mockWalletWriter := NewMockWalletWriter(ctrl)
				mockWalletReader := NewMockWalletReader(ctrl)
				mockReader := NewMockExchangeRateForCurrencyReader(ctrl)
				mockCash := NewMockExchangeRateForCurrencyCashReader(ctrl)

				// cache miss
				mockCash.EXPECT().
					GetExchangeRateForCurrency(ctx, "USD", "EUR").
					Return(float32(0), errors.New("cache miss"))

				// external service failure
				mockReader.EXPECT().
					GetExchangeRateForCurrency(ctx, "USD", "EUR").
					Return(float32(0), errors.New("external service failure"))

				return NewExchangeService(mockWalletWriter, mockWalletReader, mockReader, mockCash)
			},
			expectedAmount:  0.0,
			expectedErr:     errors.New("external service failure"),
			expectedBalance: nil,
		},
		{
			name: "insufficient_funds",
			args: args{"USD", "EUR", 1000.0},
			mockSetup: func() *ExchangeService {
				mockWalletWriter := NewMockWalletWriter(ctrl)
				mockWalletReader := NewMockWalletReader(ctrl)
				mockReader := NewMockExchangeRateForCurrencyReader(ctrl)
				mockCash := NewMockExchangeRateForCurrencyCashReader(ctrl)

				mockCash.EXPECT().
					GetExchangeRateForCurrency(ctx, "USD", "EUR").
					Return(float32(0.9), nil)

				mockWalletWriter.EXPECT().
					SaveWithdraw(ctx, userID, 1000.0, "USD").
					Return(0.0, errors.New("insufficient funds"))

				return NewExchangeService(mockWalletWriter, mockWalletReader, mockReader, mockCash)
			},
			expectedAmount:  0.0,
			expectedErr:     ErrInsufficientFunds,
			expectedBalance: nil,
		},
		{
			name: "deposit_failure",
			args: args{"USD", "EUR", 100.0},
			mockSetup: func() *ExchangeService {
				mockWalletWriter := NewMockWalletWriter(ctrl)
				mockWalletReader := NewMockWalletReader(ctrl)
				mockReader := NewMockExchangeRateForCurrencyReader(ctrl)
				mockCash := NewMockExchangeRateForCurrencyCashReader(ctrl)

				mockCash.EXPECT().
					GetExchangeRateForCurrency(ctx, "USD", "EUR").
					Return(float32(0.9), nil)

				mockWalletWriter.EXPECT().
					SaveWithdraw(ctx, userID, 100.0, "USD").
					Return(0.0, nil)

				mockWalletWriter.EXPECT().
					SaveDeposit(ctx, userID, 90.0, "EUR").
					Return(0.0, errors.New("deposit failed"))

				return NewExchangeService(mockWalletWriter, mockWalletReader, mockReader, mockCash)
			},
			expectedAmount:  90.0,
			expectedErr:     errors.New("deposit failed"),
			expectedBalance: nil,
		},
		{
			name: "wallet_reader_failure",
			args: args{"USD", "EUR", 100.0},
			mockSetup: func() *ExchangeService {
				mockWalletWriter := NewMockWalletWriter(ctrl)
				mockWalletReader := NewMockWalletReader(ctrl)
				mockReader := NewMockExchangeRateForCurrencyReader(ctrl)
				mockCash := NewMockExchangeRateForCurrencyCashReader(ctrl)

				mockCash.EXPECT().
					GetExchangeRateForCurrency(ctx, "USD", "EUR").
					Return(float32(0.9), nil)

				mockWalletWriter.EXPECT().
					SaveWithdraw(ctx, userID, 100.0, "USD").
					Return(0.0, nil)

				mockWalletWriter.EXPECT().
					SaveDeposit(ctx, userID, 90.0, "EUR").
					Return(90.0, nil)

				mockWalletReader.EXPECT().
					GetByUserID(ctx, userID).
					Return(nil, errors.New("wallet reader failed"))

				return NewExchangeService(mockWalletWriter, mockWalletReader, mockReader, mockCash)
			},
			expectedAmount:  90.0,
			expectedErr:     errors.New("wallet reader failed"),
			expectedBalance: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := tt.mockSetup()
			amount, balance, err := svc.Exchange(ctx, userID, tt.args.fromCurrency, tt.args.toCurrency, tt.args.amount)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedAmount, amount)
			assert.Equal(t, tt.expectedBalance, balance)
		})
	}
}
