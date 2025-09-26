package repositories

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupUserPostgresContainer(t *testing.T) (*sqlx.DB, func()) {
	t.Helper()

	req := tc.ContainerRequest{
		Image:        "postgres:15-alpine",
		Env:          map[string]string{"POSTGRES_PASSWORD": "password", "POSTGRES_DB": "testdb", "POSTGRES_USER": "postgres"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
	}

	container, err := tc.GenericContainer(context.Background(), tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoError(t, err)

	host, _ := container.Host(context.Background())
	port, _ := container.MappedPort(context.Background(), "5432")

	dsn := fmt.Sprintf("postgres://postgres:password@%s:%d/testdb?sslmode=disable", host, port.Int())

	var db *sqlx.DB
	for i := 0; i < 10; i++ {
		db, err = sqlx.Connect("pgx", dsn)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	assert.NoError(t, err)

	schema := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

	CREATE TABLE IF NOT EXISTS users (
		user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		username VARCHAR(50) NOT NULL UNIQUE,
		email VARCHAR(100) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	`
	_, err = db.Exec(schema)
	assert.NoError(t, err)

	teardown := func() {
		db.Close()
		container.Terminate(context.Background())
	}

	return db, teardown
}

func TestUserWriteRepository_Save(t *testing.T) {
	db, teardown := setupUserPostgresContainer(t)
	defer teardown()

	repo := NewUserWriteRepository(db)
	ctx := context.Background()

	err := repo.Save(ctx, "alice", "password123", "alice@example.com")
	assert.NoError(t, err)

	var user struct {
		Username     string `db:"username"`
		Email        string `db:"email"`
		PasswordHash string `db:"password_hash"`
	}
	err = db.Get(&user, "SELECT username, email, password_hash FROM users WHERE username=$1", "alice")
	assert.NoError(t, err)

	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "password123", user.PasswordHash)
}

func TestUserReadRepository_GetByUsernameOrEmail(t *testing.T) {
	db, teardown := setupUserPostgresContainer(t)
	defer teardown()

	writeRepo := NewUserWriteRepository(db)
	readRepo := NewUserReadRepository(db)
	ctx := context.Background()

	writeRepo.Save(ctx, "charlie", "secret", "charlie@example.com")
	writeRepo.Save(ctx, "dave", "secret2", "dave@example.com")

	t.Run("ByUsername", func(t *testing.T) {
		username := "charlie"
		user, err := readRepo.GetByUsernameOrEmail(ctx, &username, nil)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "charlie", user.Username)
	})

	t.Run("ByEmail", func(t *testing.T) {
		email := "dave@example.com"
		user, err := readRepo.GetByUsernameOrEmail(ctx, nil, &email)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "dave", user.Username)
	})

	t.Run("ByUsernameAndEmail", func(t *testing.T) {
		username := "charlie"
		email := "charlie@example.com"
		user, err := readRepo.GetByUsernameOrEmail(ctx, &username, &email)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "charlie", user.Username)
	})

	t.Run("NotFound", func(t *testing.T) {
		username := "nonexistent"
		user, err := readRepo.GetByUsernameOrEmail(ctx, &username, nil)
		assert.Error(t, err) // sql.ErrNoRows
		assert.Nil(t, user)
	})
}
