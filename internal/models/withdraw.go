package models

// WithdrawRequest represents the JSON body for withdrawing funds
// swagger:model WithdrawRequest
type WithdrawRequest struct {
	// Amount to withdraw
	// required: true
	// example: 50.0
	Amount float64 `json:"amount"`

	// Currency
	// required: true
	// example: USD
	Currency string `json:"currency"`
}

// WithdrawResponse represents a successful withdrawal response
// swagger:model WithdrawResponse
type WithdrawResponse struct {
	// Success message
	// example: Withdrawal successful
	Message string `json:"message"`

	// New balance of the user
	NewBalance CurrencyBalance `json:"new_balance"`
}

// WithdrawErrorResponse represents an error response for withdrawal
// swagger:model WithdrawErrorResponse
type WithdrawErrorResponse struct {
	// Error message
	// example: Insufficient funds or invalid amount
	Error string `json:"error"`
}
