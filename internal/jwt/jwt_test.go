package jwt

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestJWT_GenerateAndValidate(t *testing.T) {
	secret := "test-secret"
	j := New(WithSecretKey(secret), WithExpiration(time.Minute))

	userID := uuid.New()
	ctx := context.Background()

	token, err := j.Generate(ctx, userID)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Valid token should pass validation
	err = j.Validate(ctx, token)
	assert.NoError(t, err)

	// Extract claims
	claims, err := j.GetClaims(ctx, token)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestJWT_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	j := New(WithSecretKey(secret), WithExpiration(-time.Minute)) // already expired

	userID := uuid.New()
	ctx := context.Background()

	token, err := j.Generate(ctx, userID)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validation should fail
	err = j.Validate(ctx, token)
	assert.Error(t, err)

	claims, err := j.GetClaims(ctx, token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWT_InvalidToken(t *testing.T) {
	j := New(WithSecretKey("secret"))
	ctx := context.Background()

	// Totally invalid string
	err := j.Validate(ctx, "invalid.token.string")
	assert.Error(t, err)

	claims, err := j.GetClaims(ctx, "invalid.token.string")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWT_GetTokenFromRequest(t *testing.T) {
	j := New()
	ctx := context.Background()

	tests := []struct {
		name          string
		header        string
		expectedToken string
		expectError   bool
	}{
		{"ValidBearer", "Bearer mytoken123", "mytoken123", false},
		{"LowercaseBearer", "bearer mytoken123", "mytoken123", false},
		{"NoHeader", "", "", true},
		{"InvalidFormat", "Token mytoken123", "", true},
		{"TooManyParts", "Bearer a b c", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			token, err := j.GetTokenFromRequest(ctx, req)
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}

func TestJWT_Validate_WrongSecret(t *testing.T) {
	// Create token with one secret
	j1 := New(WithSecretKey("secret1"))
	j2 := New(WithSecretKey("secret2"))
	ctx := context.Background()

	userID := uuid.New()
	token, err := j1.Generate(ctx, userID)
	assert.NoError(t, err)

	// Validate with wrong secret should fail
	err = j2.Validate(ctx, token)
	assert.Error(t, err)
}
