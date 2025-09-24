package models

// ExchangeRates represents exchange rates for supported currencies
// swagger:model ExchangeRates
type ExchangeRates struct {
	USD float64 `json:"USD" example:"1.0"`
	RUB float64 `json:"RUB" example:"90.0"`
	EUR float64 `json:"EUR" example:"0.85"`
}

// ExchangeRatesResponse represents a successful response with exchange rates
// swagger:model ExchangeRatesResponse
type ExchangeRatesResponse struct {
	// Exchange rates
	Rates ExchangeRates `json:"rates"`
}

// ExchangeRatesErrorResponse represents an error response when fetching exchange rates
// swagger:model ExchangeRatesErrorResponse
type ExchangeRatesErrorResponse struct {
	// Error message
	// example: Failed to retrieve exchange rates
	Error string `json:"error"`
}
