package jwt

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
)

// JWT provides methods to generate and validate JWT tokens.
type JWT struct {
	secretKey string
	exp       time.Duration
}

// Claims represents the JWT claims structure with UUID UserID.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// Opt defines a functional option for JWT.
type Opt func(*JWT)

// WithSecretKey sets the secret key for signing.
func WithSecretKey(secret string) Opt {
	return func(j *JWT) {
		j.secretKey = secret
	}
}

// WithExpiration sets the token expiration duration.
func WithExpiration(d time.Duration) Opt {
	return func(j *JWT) {
		j.exp = d
	}
}

// New creates a new JWT with provided options.
func New(opts ...Opt) *JWT {
	j := &JWT{
		secretKey: "default-secret",
		exp:       time.Hour,
	}
	for _, opt := range opts {
		opt(j)
	}
	return j
}

// Generate creates a JWT token for a given userID.
func (j *JWT) Generate(ctx context.Context, userID uuid.UUID) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.exp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(j.secretKey))
	if err != nil {
		logger.Log.Errorw("failed to generate JWT token", "err", err, "userID", userID)
		return "", err
	}
	return signed, nil
}

// Validate parses and validates the token string, returning an error if invalid.
func (j *JWT) Validate(ctx context.Context, tokenString string) error {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(j.secretKey), nil
	})
	if err != nil {
		logger.Log.Errorw("JWT validation failed", "err", err)
		return err
	}

	if !token.Valid {
		logger.Log.Error("JWT validation failed: token invalid")
		return errors.New("invalid token")
	}

	return nil
}

// GetClaims parses the token and returns strongly-typed Claims with UUID UserID.
func (j *JWT) GetClaims(ctx context.Context, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(j.secretKey), nil
	})
	if err != nil {
		logger.Log.Errorw("failed to parse JWT", "err", err)
		return nil, err
	}

	if !token.Valid {
		logger.Log.Error("invalid JWT token")
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// GetTokenFromRequest extracts the token from Authorization header.
func (j *JWT) GetTokenFromRequest(ctx context.Context, r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		err := errors.New("authorization header missing")
		logger.Log.Warn(err.Error())
		return "", err
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		err := errors.New("invalid authorization header format")
		logger.Log.Warn(err.Error())
		return "", err
	}

	return parts[1], nil
}
