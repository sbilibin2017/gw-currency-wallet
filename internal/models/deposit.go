package models

// DepositRequest represents the JSON body for depositing funds
// swagger:model DepositRequest
type DepositRequest struct {
	// Amount to deposit
	// required: true
	// example: 100.0
	Amount float64 `json:"amount"`

	// Currency
	// required: true
	// example: USD
	Currency string `json:"currency"`
}

// DepositResponse represents a successful deposit response
// swagger:model DepositResponse
type DepositResponse struct {
	// Success message
	// example: Account topped up successfully
	Message string `json:"message"`

	// New balance of the user
	NewBalance CurrencyBalance `json:"new_balance"`
}

// DepositErrorResponse represents an error response for deposit
// swagger:model DepositErrorResponse
type DepositErrorResponse struct {
	// Error message
	// example: Invalid amount or currency
	Error string `json:"error"`
}
