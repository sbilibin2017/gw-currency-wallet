package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// Error variables
var (
	ErrUserAlreadyExists  = errors.New("username or email already exists")
	ErrUserDoesNotExist   = errors.New("username does not exist")
	ErrInvalidCredentials = errors.New("invalid username or password")
)

// UserReader defines read-only operations for users.
type UserReader interface {
	GetByUsernameOrEmail(ctx context.Context, username *string, email *string) (*models.UserDB, error)
}

// UserWriter defines write operations for users.
type UserWriter interface {
	Save(ctx context.Context, username string, password string, email string) error
}

// JWTGenerator defines an interface for generating JWT tokens.
type JWTGenerator interface {
	Generate(ctx context.Context, userID uuid.UUID) (string, error)
}

// AuthService handles registration and login.
type AuthService struct {
	reader UserReader
	writer UserWriter
	jwt    JWTGenerator
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(reader UserReader, writer UserWriter, jwt JWTGenerator) *AuthService {
	return &AuthService{
		reader: reader,
		writer: writer,
		jwt:    jwt,
	}
}

// Register registers a new user.
func (svc *AuthService) Register(ctx context.Context, username, password, email string) error {
	user, err := svc.reader.GetByUsernameOrEmail(ctx, &username, &email)
	if err != nil {
		logger.Log.Errorw("failed to check user exists", "err", err)
		return err
	}
	if user != nil {
		logger.Log.Errorw("user already exists", "username", username, "email", email)
		return ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Log.Errorw("failed to hash password", "err", err)
		return err
	}

	if err := svc.writer.Save(ctx, username, string(hashedPassword), email); err != nil {
		logger.Log.Errorw("failed to save user", "err", err)
		return err
	}

	return nil
}

// Login authenticates a user and returns a JWT token.
func (svc *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := svc.reader.GetByUsernameOrEmail(ctx, &username, nil)
	if err != nil {
		logger.Log.Errorw("failed to get user", "err", err)
		return "", err
	}
	if user == nil {
		logger.Log.Errorw("user does not exist", "username", username)
		return "", ErrUserDoesNotExist
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		logger.Log.Errorw("invalid credentials", "username", username)
		return "", ErrInvalidCredentials
	}

	token, err := svc.jwt.Generate(ctx, user.UserID)
	if err != nil {
		logger.Log.Errorw("failed to generate JWT", "err", err)
		return "", err
	}

	return token, nil
}
