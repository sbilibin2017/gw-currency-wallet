package repositories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestExchangeRateCacheRepository(t *testing.T) {
	ctx := context.Background()

	// Start Redis container
	req := testcontainers.ContainerRequest{
		Image:        "redis:7.0-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoError(t, err)
	defer redisC.Terminate(ctx)

	// Get container host and port
	host, err := redisC.Host(ctx)
	assert.NoError(t, err)
	port, err := redisC.MappedPort(ctx, "6379")
	assert.NoError(t, err)

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port.Port()),
	})
	defer rdb.Close()

	// Ping to ensure connection
	err = rdb.Ping(ctx).Err()
	assert.NoError(t, err)

	repo := NewExchangeRateCacheRepository(rdb, 2*time.Second)

	t.Run("Set and Get exchange rate", func(t *testing.T) {
		from, to := "USD", "EUR"
		rate := float32(1.23)

		err := repo.SetExchangeRateForCurrency(ctx, from, to, rate)
		assert.NoError(t, err)

		got, err := repo.GetExchangeRateForCurrency(ctx, from, to)
		assert.NoError(t, err)
		assert.Equal(t, rate, got)
	})

	t.Run("Get missing key returns error", func(t *testing.T) {
		_, err := repo.GetExchangeRateForCurrency(ctx, "ABC", "XYZ")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exchange rate not found")
	})

	t.Run("Cached value expires", func(t *testing.T) {
		from, to := "GBP", "USD"
		rate := float32(1.5)

		err := repo.SetExchangeRateForCurrency(ctx, from, to, rate)
		assert.NoError(t, err)

		// Wait for expiration (2s)
		time.Sleep(3 * time.Second)

		_, err = repo.GetExchangeRateForCurrency(ctx, from, to)
		assert.Error(t, err)
	})
}
