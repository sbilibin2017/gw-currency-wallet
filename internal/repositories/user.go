package repositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
	"go.uber.org/zap"
)

// UserReadRepository implements read-only operations
type UserReadRepository struct {
	db  *sqlx.DB
	log *zap.SugaredLogger
}

// NewUserReadRepository creates a new read repository
func NewUserReadRepository(db *sqlx.DB, log *zap.SugaredLogger) *UserReadRepository {
	return &UserReadRepository{db: db, log: log}
}

// GetByUsernameOrEmail retrieves a user by username or email
func (r *UserReadRepository) GetByUsernameOrEmail(
	ctx context.Context,
	username *string,
	email *string,
) (*models.UserDB, error) {
	r.log.Infow("fetching user by username or email", "username", username, "email", email)

	query, args := buildGetByUsernameOrEmailQuery(username, email)

	var user models.UserDB
	if err := r.db.GetContext(ctx, &user, query, args...); err != nil {
		r.log.Warnw("user not found", "error", err)
		return nil, err
	}

	r.log.Infow("user fetched successfully", "username", user.Username, "email", user.Email)
	return &user, nil
}

// buildGetByUsernameOrEmailQuery constructs the SQL query and arguments
func buildGetByUsernameOrEmailQuery(username *string, email *string) (string, []any) {
	var sb strings.Builder
	var args []interface{}
	argPos := 1

	sb.WriteString("SELECT id, username, email, password, created_at, updated_at FROM users WHERE 1=1")

	if username != nil {
		sb.WriteString(fmt.Sprintf(" AND username = $%d", argPos))
		args = append(args, *username)
		argPos++
	}
	if email != nil {
		sb.WriteString(fmt.Sprintf(" AND email = $%d", argPos))
		args = append(args, *email)
		argPos++
	}

	return sb.String(), args
}

// UserWriteRepository implements write operations
type UserWriteRepository struct {
	db  *sqlx.DB
	log *zap.SugaredLogger
}

// NewUserWriteRepository creates a new write repository
func NewUserWriteRepository(db *sqlx.DB, log *zap.SugaredLogger) *UserWriteRepository {
	return &UserWriteRepository{db: db, log: log}
}

// Save inserts a new user or updates existing one if username conflict
func (r *UserWriteRepository) Save(
	ctx context.Context,
	username, password, email string,
) error {
	r.log.Infow("saving user", "username", username, "email", email)

	query, args := buildUserSaveQuery(username, email, password)

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		r.log.Errorw("failed to save user", "error", err)
		return err
	}

	r.log.Infow("user saved successfully", "username", username, "email", email)
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
	args := []interface{}{username, email, password}

	return query, args
}
