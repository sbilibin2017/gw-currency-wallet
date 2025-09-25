package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// --- Setup Postgres ---
func setupPostgres(t *testing.T) (*sqlx.DB, func()) {
	logger.Initialize("debug")
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		Env:          map[string]string{"POSTGRES_PASSWORD": "secret", "POSTGRES_DB": "testdb", "POSTGRES_USER": "postgres"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoError(t, err)

	host, err := container.Host(ctx)
	assert.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432")
	assert.NoError(t, err)

	dsn := fmt.Sprintf("postgres://postgres:secret@%s:%s/testdb?sslmode=disable", host, port.Port())
	db, err := sqlx.Connect("pgx", dsn)
	assert.NoError(t, err)

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	migrations := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`,
		`CREATE TABLE IF NOT EXISTS users (
			user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			username VARCHAR(50) NOT NULL UNIQUE,
			email VARCHAR(100) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS wallets (
			wallet_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			currency CHAR(3) NOT NULL,
			balance NUMERIC(20,2) NOT NULL DEFAULT 0.0,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE (user_id, currency)
		);`,
	}

	for _, m := range migrations {
		_, err = db.Exec(m)
		assert.NoError(t, err)
	}

	return db, func() {
		db.Close()
		container.Terminate(ctx)
	}
}

// --- Helper ---
func getBalance(t *testing.T, db *sqlx.DB, userID uuid.UUID, currency string) float64 {
	var balance float64
	err := db.Get(&balance, `SELECT balance FROM wallets WHERE user_id=$1 AND currency=$2`, userID, currency)
	assert.NoError(t, err)
	return balance
}

// --- Deposit Tests ---
func TestSaveDeposit(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()
	ctx := context.Background()

	userID := uuid.New()
	_, err := db.Exec(`INSERT INTO users (user_id, username, email, password_hash) VALUES ($1, $2, $3, $4)`,
		userID, "alice", "alice@example.com", "password123")
	assert.NoError(t, err)

	writer := NewWalletWriterRepository(db, nil)

	balance, err := writer.SaveDeposit(ctx, userID, 100, "USD")
	assert.NoError(t, err)
	assert.Equal(t, 100.0, balance)
	assert.Equal(t, 100.0, getBalance(t, db, userID, "USD"))

	balance, err = writer.SaveDeposit(ctx, userID, 50, "USD")
	assert.NoError(t, err)
	assert.Equal(t, 150.0, balance)
	assert.Equal(t, 150.0, getBalance(t, db, userID, "USD"))
}

// --- Withdraw Tests ---
func TestSaveWithdraw(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()
	ctx := context.Background()

	userID := uuid.New()
	_, err := db.Exec(`INSERT INTO users (user_id, username, email, password_hash) VALUES ($1, $2, $3, $4)`,
		userID, "bob", "bob@example.com", "password123")
	assert.NoError(t, err)

	writer := NewWalletWriterRepository(db, nil)

	// Deposit first
	_, err = writer.SaveDeposit(ctx, userID, 200, "USD")
	assert.NoError(t, err)

	balance, err := writer.SaveWithdraw(ctx, userID, 80, "USD")
	assert.NoError(t, err)
	assert.Equal(t, 120.0, balance)
	assert.Equal(t, 120.0, getBalance(t, db, userID, "USD"))

	balance, err = writer.SaveWithdraw(ctx, userID, 50, "USD")
	assert.NoError(t, err)
	assert.Equal(t, 70.0, balance)
	assert.Equal(t, 70.0, getBalance(t, db, userID, "USD"))

	_, err = writer.SaveWithdraw(ctx, userID, 100, "USD")
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Equal(t, 70.0, getBalance(t, db, userID, "USD"))
}

// --- Concurrency Tests ---
func TestSaveDepositConcurrency(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()
	ctx := context.Background()

	userID := uuid.New()
	_, _ = db.Exec(`INSERT INTO users (user_id, username, email, password_hash) VALUES ($1, $2, $3, $4)`,
		userID, "concurrent", "concurrent@example.com", "pass")

	writer := NewWalletWriterRepository(db, nil)

	const numGoroutines = 1000
	const amount = 1.0
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = writer.SaveDeposit(ctx, userID, amount, "USD")
		}()
	}
	wg.Wait()

	assert.Equal(t, float64(numGoroutines)*amount, getBalance(t, db, userID, "USD"))
}

func TestSaveWithdrawConcurrency(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()
	ctx := context.Background()

	userID := uuid.New()
	initial := 1000.0
	_, _ = db.Exec(`INSERT INTO users (user_id, username, email, password_hash) VALUES ($1, $2, $3, $4)`,
		userID, "concurrent2", "concurrent2@example.com", "pass")

	writer := NewWalletWriterRepository(db, nil)

	// Deposit first
	_, err := writer.SaveDeposit(ctx, userID, initial, "USD")
	assert.NoError(t, err)

	const numGoroutines = 1000
	const amount = 1.0
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = writer.SaveWithdraw(ctx, userID, amount, "USD")
		}()
	}
	wg.Wait()

	assert.Equal(t, initial-float64(numGoroutines)*amount, getBalance(t, db, userID, "USD"))
}

// --- WalletReaderRepository Tests ---
func TestWalletReaderRepository_GetByUserID(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()
	ctx := context.Background()

	userID := uuid.New()

	// Insert user
	_, err := db.Exec(`INSERT INTO users (user_id, username, email, password_hash) VALUES ($1, $2, $3, $4)`,
		userID, "alice", "alice@example.com", "password123")
	assert.NoError(t, err)

	// Insert wallets
	walletsData := []struct {
		currency string
		balance  float64
	}{
		{"USD", 100.0},
		{"EUR", 50.0},
		{"RUB", 5000.0},
	}

	for _, w := range walletsData {
		_, err := db.Exec(`INSERT INTO wallets (user_id, currency, balance) VALUES ($1, $2, $3)`,
			userID, w.currency, w.balance)
		assert.NoError(t, err)
	}

	reader := NewWalletReaderRepository(db)

	t.Run("Get all balances for existing user", func(t *testing.T) {
		balances, err := reader.GetByUserID(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, balances, len(walletsData))

		for _, w := range walletsData {
			assert.Equal(t, w.balance, balances[w.currency])
		}
	})

	t.Run("Return empty map for unknown user", func(t *testing.T) {
		unknownUser := uuid.New()
		balances, err := reader.GetByUserID(ctx, unknownUser)
		assert.NoError(t, err)
		assert.Empty(t, balances)
	})
}
