package models

// CurrencyBalance represents balances for different currencies
// swagger:model CurrencyBalance
type CurrencyBalance struct {
	// Balance in USD
	// example: 100.0
	USD float64 `json:"USD"`

	// Balance in RUB
	// example: 5000.0
	RUB float64 `json:"RUB"`

	// Balance in EUR
	// example: 50.0
	EUR float64 `json:"EUR"`
}
