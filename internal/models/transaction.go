package models

// Transaction represents a financial transaction, including amount, user, timestamp, and operation type.
type Transaction struct {
	TransactionID string  `json:"transaction_id" bson:"transaction_id"` // TransactionID is a unique identifier for the transaction.
	Timestamp     int64   `json:"timestamp" bson:"timestamp"`           // Timestamp is the Unix timestamp (in seconds) when the transaction occurred.
	Amount        float64 `json:"amount" bson:"amount"`                 // Amount is the monetary value of the transaction.
	UserID        string  `json:"user_id" bson:"user_id"`               // UserID is the identifier of the user who initiated the transaction.
	Operation     string  `json:"operation" bson:"operation"`           // Operation describes the type of transaction, e.g., "deposit", "withdrawal", or "transfer".
}
