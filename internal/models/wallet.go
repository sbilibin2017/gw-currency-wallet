package models

import (
	"time"

	"github.com/google/uuid"
)

// Supported currency codes
const (
	USD = "USD"
	RUB = "RUB"
	EUR = "EUR"
)

// WalletDB represents a wallet row in the database
type WalletDB struct {
	WalletID  uuid.UUID `json:"wallet_id" db:"wallet_id"`   // Unique wallet identifier
	UserID    uuid.UUID `json:"user_id" db:"user_id"`       // Identifier of the wallet's owner
	Currency  string    `json:"currency" db:"currency"`     // Currency code (e.g., USD, RUB, EUR)
	Balance   float64   `json:"balance" db:"balance"`       // Current balance in the wallet
	CreatedAt time.Time `json:"created_at" db:"created_at"` // Timestamp when the wallet was created
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"` // Timestamp of the last wallet update
}
