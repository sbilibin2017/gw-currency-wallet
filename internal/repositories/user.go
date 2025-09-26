package repositories

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

// UserReadRepository implements read-only operations
type UserReadRepository struct {
	db *sqlx.DB
}

// NewUserReadRepository creates a new read repository
func NewUserReadRepository(db *sqlx.DB) *UserReadRepository {
	return &UserReadRepository{db: db}
}

// GetByUsernameOrEmail fetches a user by username or email
func (r *UserReadRepository) GetByUsernameOrEmail(
	ctx context.Context,
	username *string,
	email *string,
) (*models.UserDB, error) {
	const query = `
		SELECT user_id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE ($1::VARCHAR IS NULL OR username = $1)
		  AND ($2::VARCHAR IS NULL OR email = $2)
		LIMIT 1
	`

	var user models.UserDB
	err := r.db.GetContext(ctx, &user, query, username, email)
	if err != nil {
		logger.Log.Errorw("failed to get user by username or email", "username", username, "email", email, "error", err)
		return nil, err
	}

	return &user, nil
}

// UserWriteRepository implements write operations
type UserWriteRepository struct {
	db *sqlx.DB
}

// NewUserWriteRepository creates a new write repository
func NewUserWriteRepository(db *sqlx.DB) *UserWriteRepository {
	return &UserWriteRepository{db: db}
}

// Save inserts a new user or updates existing one if username conflict
func (r *UserWriteRepository) Save(
	ctx context.Context,
	username, password, email string,
) error {
	query, args := buildUserSaveQuery(username, email, password)

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		logger.Log.Errorw("failed to save user", "error", err)
		return err
	}

	return nil
}

// buildUserSaveQuery constructs the SQL insert/update query and arguments
func buildUserSaveQuery(username, email, password string) (string, []any) {
	query := `
		INSERT INTO users (username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (username) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
		    email = EXCLUDED.email,
		    updated_at = NOW()
	`
	args := []any{username, email, password}

	logger.Log.Infow("query", query, "args", args)

	return query, args
}
