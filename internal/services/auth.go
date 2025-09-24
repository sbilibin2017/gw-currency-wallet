package services

import (
	"context"
	"errors"

	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
	"go.uber.org/zap"
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

// AuthService provides methods for user registration and login.
type AuthService struct {
	reader UserReader
	writer UserWriter
	jwt    *jwt.JWT
	log    *zap.SugaredLogger
}

// NewAuthService creates a new AuthService with reader, writer, JWT, and logger.
func NewAuthService(
	reader UserReader,
	writer UserWriter,
	jwt *jwt.JWT,
	log *zap.SugaredLogger,
) *AuthService {
	return &AuthService{
		reader: reader,
		writer: writer,
		jwt:    jwt,
		log:    log,
	}
}

// Register registers a new user in the system.
func (svc *AuthService) Register(
	ctx context.Context,
	username, password, email string,
) error {
	svc.log.Infow("checking if user exists", "username", username, "email", email)
	user, err := svc.reader.GetByUsernameOrEmail(ctx, &username, &email)
	if err != nil && !errors.Is(err, ErrUserAlreadyExists) {
		svc.log.Errorw("failed to check if user exists", "error", err)
		return err
	}
	if user != nil {
		svc.log.Warnw("user already exists", "username", username, "email", email)
		return ErrUserAlreadyExists
	}

	svc.log.Infow("hashing password for new user", "username", username)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		svc.log.Errorw("failed to hash password", "error", err)
		return err
	}

	svc.log.Infow("saving new user to database", "username", username, "email", email)
	if err := svc.writer.Save(ctx, username, string(hashedPassword), email); err != nil {
		svc.log.Errorw("failed to save user", "error", err)
		return err
	}

	svc.log.Infow("user registered successfully", "username", username, "email", email)
	return nil
}

// Login authenticates a user and returns a JWT token.
func (svc *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	svc.log.Infow("retrieving user by username", "username", username)
	user, err := svc.reader.GetByUsernameOrEmail(ctx, &username, nil)
	if err != nil {
		svc.log.Warnw("user not found", "username", username)
		return "", err
	}

	if user == nil {
		return "", ErrUserDoesNotExist
	}

	svc.log.Infow("verifying password for user", "username", username)
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		svc.log.Warnw("invalid password", "username", username)
		return "", ErrInvalidCredentials
	}

	svc.log.Infow("generating JWT token for user", "username", username, "userID", user.UserID)
	token, err := svc.jwt.Generate(ctx, user.UserID)
	if err != nil {
		svc.log.Errorw("failed to generate JWT token", "error", err)
		return "", err
	}

	svc.log.Infow("login successful", "username", username)
	return token, nil
}
