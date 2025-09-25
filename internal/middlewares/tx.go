package middlewares

import (
	"context"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
)

// TxMiddleware wraps an HTTP handler with a database transaction
func TxMiddleware(db *sqlx.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tx, err := db.Beginx()
			if err != nil {
				logger.Log.Errorw("failed to begin transaction", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			defer func() {
				if rec := recover(); rec != nil {
					tx.Rollback()
					panic(rec)
				}
			}()

			ctx := setTxToContext(r.Context(), tx)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)

			if err := tx.Commit(); err != nil {
				logger.Log.Errorw("failed to commit transaction", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		})
	}
}

// contextKey is an unexported type for keys in context
type contextKey struct{}

var txKey = contextKey{}

// setTxToContext stores a transaction in the context
func setTxToContext(ctx context.Context, tx *sqlx.Tx) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// GetTxFromContext retrieves the transaction from the context. Returns nil if not present.
func GetTxFromContext(ctx context.Context) *sqlx.Tx {
	tx, _ := ctx.Value(txKey).(*sqlx.Tx)
	return tx
}
