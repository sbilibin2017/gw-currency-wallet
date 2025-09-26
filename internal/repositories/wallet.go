package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// WalletWriterRepository handles wallet write operations
type WalletWriterRepository struct {
	db       *sqlx.DB
	txGetter func(ctx context.Context) *sqlx.Tx
}

func NewWalletWriterRepository(db *sqlx.DB, txGetter func(ctx context.Context) *sqlx.Tx) *WalletWriterRepository {
	return &WalletWriterRepository{db: db, txGetter: txGetter}
}

// SaveDeposit performs an UPSERT: creates wallet if not exists, otherwise increases balance.
func (r *WalletWriterRepository) SaveDeposit(ctx context.Context, userID uuid.UUID, amount float64, currency string) error {
	query := `
		INSERT INTO wallets (wallet_id, user_id, currency, balance, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (user_id, currency)
		DO UPDATE SET balance = wallets.balance + EXCLUDED.balance, updated_at = NOW()
		RETURNING balance
	`

	var executor sqlx.ExtContext = r.db
	if r.txGetter != nil {
		if tx := r.txGetter(ctx); tx != nil {
			executor = tx
		}
	}

	var ignore float64
	err := sqlx.GetContext(ctx, executor, &ignore, query, uuid.New(), userID, currency, amount)
	return err
}

// SaveWithdraw performs an UPSERT-like withdrawal in a single query.
func (r *WalletWriterRepository) SaveWithdraw(ctx context.Context, userID uuid.UUID, amount float64, currency string) error {
	query := `
		INSERT INTO wallets (wallet_id, user_id, currency, balance, created_at, updated_at)
		VALUES ($1, $2, $3, 0, NOW(), NOW())
		ON CONFLICT (user_id, currency)
		DO UPDATE SET balance = wallets.balance - $4, updated_at = NOW()
		WHERE wallets.balance >= $4
		RETURNING balance
	`

	var executor sqlx.ExtContext = r.db
	if r.txGetter != nil {
		if tx := r.txGetter(ctx); tx != nil {
			executor = tx
		}
	}

	var ignore float64
	err := sqlx.GetContext(ctx, executor, &ignore, query, uuid.New(), userID, currency, amount)
	if err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return err
	}
	return nil
}

// WalletReaderRepository handles wallet read operations
type WalletReaderRepository struct {
	db *sqlx.DB
}

// NewWalletReaderRepository creates a new reader repository
func NewWalletReaderRepository(db *sqlx.DB) *WalletReaderRepository {
	return &WalletReaderRepository{db: db}
}

// GetByUserID retrieves all wallets for a given user as a map[currency]balance
func (r *WalletReaderRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (map[string]float64, error) {
	const query = `
		SELECT currency, balance
		FROM wallets
		WHERE user_id = $1
	`

	var wallets []struct {
		Currency string  `db:"currency"`
		Balance  float64 `db:"balance"`
	}

	if err := r.db.SelectContext(ctx, &wallets, query, userID); err != nil {
		return nil, err
	}

	balances := make(map[string]float64, len(wallets))
	for _, w := range wallets {
		balances[w.Currency] = w.Balance
	}

	return balances, nil
}
