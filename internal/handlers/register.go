package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// Registerer defines the interface that the service must implement.
type Registerer interface {
	Register(ctx context.Context, username, password, email string) error
}

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

// NewRegisterHandler returns an HTTP handler for user registration.
// @Summary Register a new user
// @Description Creates a new user account. Ensures unique username and email. Password is hashed before storing.
// @Tags auth
// @Accept json
// @Produce json
// @Param registerRequest body handlers.RegisterRequest true "User registration request"
// @Success 201 {object} handlers.RegisterResponse "User successfully registered"
// @Failure 400 {object} handlers.RegisterErrorResponse "Username or email already exists / invalid request"
// @Router /register [post]
func NewRegisterHandler(svc Registerer, log *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterRequest

		log.Infow("decoding register request", "method", r.Method, "path", r.URL.Path)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Errorw("failed to decode request body", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(RegisterErrorResponse{
				Error: "invalid request body",
			})
			return
		}

		log.Infow("calling register service", "username", req.Username, "email", req.Email)
		if err := svc.Register(r.Context(), req.Username, req.Password, req.Email); err != nil {
			log.Warnw("registration failed", "username", req.Username, "email", req.Email, "error", err)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(RegisterErrorResponse{
				Error: err.Error(),
			})
			return
		}

		log.Infow("user registered successfully", "username", req.Username, "email", req.Email)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(RegisterResponse{
			Message: "User registered successfully",
		})
	}
}
