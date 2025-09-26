package repositories

import (
	"context"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

type UserReadRepository struct {
	db *sqlx.DB
}

func NewUserReadRepository(db *sqlx.DB) *UserReadRepository {
	return &UserReadRepository{db: db}
}

func (r *UserReadRepository) GetByUsernameOrEmail(ctx context.Context, username, email *string) (*models.UserDB, error) {
	const query = `
		SELECT user_id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE ($1::VARCHAR IS NULL OR username = $1)
		  AND ($2::VARCHAR IS NULL OR email = $2)
		LIMIT 1
	`

	var user models.UserDB
	err := r.db.GetContext(ctx, &user, query, username, email)

	// Log with query in single line
	logger.Log.Infow(
		"query", strings.Join(strings.Fields(query), " "),
		"args", []any{username, email},
		"result", user,
		"error", err,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

type UserWriteRepository struct {
	db *sqlx.DB
}

func NewUserWriteRepository(db *sqlx.DB) *UserWriteRepository {
	return &UserWriteRepository{db: db}
}

func (r *UserWriteRepository) Save(ctx context.Context, username, password, email string) error {
	query := `
		INSERT INTO users (username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (username) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
		    email = EXCLUDED.email,
		    updated_at = NOW()
	`
	args := []any{username, email, password}

	res, err := r.db.ExecContext(ctx, query, args...)
	var rowsAffected int64
	if res != nil {
		rowsAffected, _ = res.RowsAffected()
	}

	// Log with query in single line
	logger.Log.Infow(
		"query", strings.Join(strings.Fields(query), " "),
		"args", args,
		"result", rowsAffected,
		"error", err,
	)

	return err
}
