package models

// BalanceResponse represents a successful response with user balances
// swagger:model BalanceResponse
type BalanceResponse struct {
	Balance CurrencyBalance `json:"balance"`
}

// CurrencyBalance represents balances for different currencies
// swagger:model CurrencyBalance
type CurrencyBalance struct {
	USD float64 `json:"USD" example:"100.0"`
	RUB float64 `json:"RUB" example:"5000.0"`
	EUR float64 `json:"EUR" example:"50.0"`
}

// BalanceErrorResponse represents an error response when fetching balance
// swagger:model BalanceErrorResponse
type BalanceErrorResponse struct {
	// Error message
	// example: Unauthorized
	Error string `json:"error"`
}
