package middlewares

import (
	"context"
	"net/http"

	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
)

// Tokener defines the minimal interface needed by the middleware
type Tokener interface {
	GetTokenFromRequest(ctx context.Context, r *http.Request) (string, error)
	Validate(ctx context.Context, tokenString string) error
}

// AuthMiddleware returns a middleware that validates JWT using a JWTProvider
func AuthMiddleware(tokener Tokener) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			tokenString, err := tokener.GetTokenFromRequest(ctx, r)
			if err != nil {
				logger.Log.Errorw("authorization failed", "err", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if err := tokener.Validate(ctx, tokenString); err != nil {
				logger.Log.Errorw("authorization failed", "err", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
