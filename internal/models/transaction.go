package models

// Transaction описывает перевод средств
type Transaction struct {
	TransactionID string  `json:"transaction_id" bson:"transaction_id"`
	Timestamp     int64   `json:"timestamp" bson:"timestamp"`
	Amount        float64 `json:"amount" bson:"amount"`
	UserID        string  `json:"user_id" bson:"user_id"`
	Operation     string  `json:"operation" bson:"operation"`
}
