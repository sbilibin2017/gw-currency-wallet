package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	"github.com/sbilibin2017/gw-currency-wallet/internal/services"
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
	// default: john_doe
	Username string `json:"username"`

	// Password
	// required: true
	// default: secret123
	Password string `json:"password"`

	// Email
	// required: true
	// default: john@example.com
	Email string `json:"email"`
}

// RegisterResponse represents a successful registration response
// swagger:model RegisterResponse
type RegisterResponse struct {
	// Success message
	// default: User registered successfully
	Message string `json:"message"`
}

// RegisterErrorResponse represents an error response for registration
// swagger:model RegisterErrorResponse
type RegisterErrorResponse struct {
	// Error message
	// default: Username or email already exists
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
func NewRegisterHandler(svc Registerer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(RegisterErrorResponse{
				Error: "Username or email already exists",
			})
			return
		}

		err := svc.Register(r.Context(), req.Username, req.Password, req.Email)
		if err != nil {
			switch err {
			case services.ErrUserAlreadyExists:
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(RegisterErrorResponse{
					Error: "Username or email already exists",
				})
			default:
				logger.Log.Errorw("internal server error", "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(RegisterErrorResponse{
					Error: "Internal server error",
				})
			}
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(RegisterResponse{
			Message: "User registered successfully",
		})
	}
}
