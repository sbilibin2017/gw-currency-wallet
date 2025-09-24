package models

import (
	"time"

	"github.com/google/uuid"
)

// UserDB represents a user record in the database
type UserDB struct {
	UserID    uuid.UUID `json:"id" db:"id"`                 // Primary key
	Username  string    `json:"username" db:"username"`     // Unique username
	Email     string    `json:"email" db:"email"`           // User email
	Password  string    `json:"password" db:"password"`     // Hashed password
	CreatedAt time.Time `json:"created_at" db:"created_at"` // Creation timestamp
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"` // Last update timestamp
}
