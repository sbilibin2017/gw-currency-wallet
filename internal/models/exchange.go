package models

// ExchangeRequest represents the JSON body for currency exchange
// swagger:model ExchangeRequest
type ExchangeRequest struct {
	// Source currency
	// required: true
	// example: USD
	FromCurrency string `json:"from_currency"`

	// Target currency
	// required: true
	// example: EUR
	ToCurrency string `json:"to_currency"`

	// Amount to exchange
	// required: true
	// example: 100.0
	Amount float64 `json:"amount"`
}

// ExchangeResponse represents a successful currency exchange response
// swagger:model ExchangeResponse
type ExchangeResponse struct {
	// Success message
	// example: Exchange successful
	Message string `json:"message"`

	// Amount received after exchange
	// example: 85.0
	ExchangedAmount float64 `json:"exchanged_amount"`

	// New balance after exchange
	NewBalance CurrencyBalance `json:"new_balance"`
}

// ExchangeErrorResponse represents an error response for currency exchange
// swagger:model ExchangeErrorResponse
type ExchangeErrorResponse struct {
	// Error message
	// example: Insufficient funds or invalid currencies
	Error string `json:"error"`
}
