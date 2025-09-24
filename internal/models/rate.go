package models

// Rates represents exchange rates for supported currencies
// swagger:model Rates
type Rates struct {
	USD float64 `json:"USD" example:"1.0"`
	RUB float64 `json:"RUB" example:"90.0"`
	EUR float64 `json:"EUR" example:"0.85"`
}

// RatesResponse represents a successful response with exchange rates
// swagger:model RatesResponse
type RatesResponse struct {
	// Exchange rates
	Rates Rates `json:"rates"`
}

// RatesErrorResponse represents an error response when fetching exchange rates
// swagger:model RatesErrorResponse
type RatesErrorResponse struct {
	// Error message
	// example: Failed to retrieve exchange rates
	Error string `json:"error"`
}
