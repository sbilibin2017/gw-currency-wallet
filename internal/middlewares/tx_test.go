package middlewares

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestTxMiddleware_Success(t *testing.T) {
	// Create sqlmock
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	// Expect Begin and Commit
	mock.ExpectBegin()
	mock.ExpectCommit()

	// Next handler should receive tx in context
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		tx := GetTxFromContext(r.Context())
		assert.NotNil(t, tx)
		w.WriteHeader(http.StatusOK)
	})

	handler := TxMiddleware(sqlxDB)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTxMiddleware_BeginError(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// Close db so Begin fails
	db.Close()

	handler := TxMiddleware(sqlxDB)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestTxMiddleware_CommitError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	// Begin succeeds, Commit fails
	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(sql.ErrConnDone)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handler := TxMiddleware(sqlxDB)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTxMiddleware_Panic(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectBegin()
	mock.ExpectRollback()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := TxMiddleware(sqlxDB)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	assert.Panics(t, func() {
		handler.ServeHTTP(rr, req)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}
