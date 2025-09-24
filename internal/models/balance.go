package models

// BalanceResponse represents a successful response with user balances
// swagger:model BalanceResponse
type BalanceResponse struct {
	// User balances
	Balance *CurrencyBalance `json:"balance"`
}

// BalanceErrorResponse represents an error response when fetching balance
// swagger:model BalanceErrorResponse
type BalanceErrorResponse struct {
	// Error message
	// example: Unauthorized
	Error string `json:"error"`
}
