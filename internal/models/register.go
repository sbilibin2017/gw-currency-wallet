package models

// RegisterRequest represents the JSON body for user registration
// swagger:model RegisterRequest
type RegisterRequest struct {
	// Username
	// required: true
	// example: john_doe
	Username string `json:"username"`

	// Password
	// required: true
	// example: secret123
	Password string `json:"password"`

	// Email
	// required: true
	// example: john@example.com
	Email string `json:"email"`
}

// RegisterResponse represents a successful registration response
// swagger:model RegisterResponse
type RegisterResponse struct {
	// Success message
	// example: User registered successfully
	Message string `json:"message"`
}

// RegisterErrorResponse represents an error response for registration
// swagger:model RegisterErrorResponse
type RegisterErrorResponse struct {
	// Error message
	// example: Username or email already exists
	Error string `json:"error"`
}
