package repositories

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
)

// ExchangeRateCacheRepository provides cached exchange rates using Redis
type ExchangeRateCacheRepository struct {
	client *redis.Client
	exp    time.Duration // expiration duration for cached rates
}

// NewExchangeRateCacheRepository creates a new repository instance with optional TTL
func NewExchangeRateCacheRepository(client *redis.Client, expiration time.Duration) *ExchangeRateCacheRepository {
	return &ExchangeRateCacheRepository{
		client: client,
		exp:    expiration,
	}
}

// GetExchangeRateForCurrency fetches a cached exchange rate between two currencies
func (r *ExchangeRateCacheRepository) GetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string) (float32, error) {
	key := fmt.Sprintf("exchange_rate:%s:%s", fromCurrency, toCurrency)

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("exchange rate not found in cache for %s->%s", fromCurrency, toCurrency)
		}
		logger.Log.Errorw("failed to get exchange rate from cache", "key", key, "error", err)
		return 0, err
	}

	rate, err := strconv.ParseFloat(val, 32)
	if err != nil {
		logger.Log.Errorw("invalid cached exchange rate value", "key", key, "value", val, "error", err)
		return 0, err
	}

	return float32(rate), nil
}

// SetExchangeRateForCurrency caches a new exchange rate in Redis with expiration
func (r *ExchangeRateCacheRepository) SetExchangeRateForCurrency(ctx context.Context, fromCurrency, toCurrency string, rate float32) error {
	key := fmt.Sprintf("exchange_rate:%s:%s", fromCurrency, toCurrency)
	err := r.client.Set(ctx, key, fmt.Sprintf("%f", rate), r.exp).Err()
	if err != nil {
		logger.Log.Errorw("failed to set exchange rate in cache", "key", key, "rate", rate, "error", err)
		return err
	}
	return nil
}
