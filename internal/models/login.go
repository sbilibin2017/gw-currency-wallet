package models

// LoginRequest represents the JSON body for user login
// swagger:model LoginRequest
type LoginRequest struct {
	// Username
	// required: true
	// example: john_doe
	Username string `json:"username"`

	// Password
	// required: true
	// example: secret123
	Password string `json:"password"`
}

// LoginResponse represents a successful login response
// swagger:model LoginResponse
type LoginResponse struct {
	// JWT token
	// example: JWT_TOKEN
	Token string `json:"token"`
}

// LoginErrorResponse represents an error response for login
// swagger:model LoginErrorResponse
type LoginErrorResponse struct {
	// Error message
	// example: Invalid username or password
	Error string `json:"error"`
}
