package jwt

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWT provides methods to generate and validate JWT tokens.
type JWT struct {
	SecretKey string        // Secret key for signing tokens
	Exp       time.Duration // Token expiration duration
}

// New creates a new JWT instance
func New(secretKey string, expiration time.Duration) *JWT {
	return &JWT{
		SecretKey: secretKey,
		Exp:       expiration,
	}
}

// Generate creates a JWT token for a given userID (uuid.UUID)
func (j *JWT) Generate(ctx context.Context, userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(j.Exp).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.SecretKey))
}

// GetUserID parses the token string and returns the userID (uuid.UUID) if valid
func (j *JWT) GetUserID(ctx context.Context, tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(j.SecretKey), nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if userIDStr, ok := claims["user_id"].(string); ok {
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return uuid.Nil, errors.New("invalid user_id format")
			}
			return userID, nil
		}
		return uuid.Nil, errors.New("user_id not found in token")
	}
	return uuid.Nil, errors.New("invalid token")
}

// GetTokenFromRequest extracts the token string from the Authorization header
func (j *JWT) GetTokenFromRequest(ctx context.Context, r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header missing")
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("invalid authorization header format")
	}

	return parts[1], nil
}
