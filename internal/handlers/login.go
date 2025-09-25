package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	"github.com/sbilibin2017/gw-currency-wallet/internal/services"
)

// Loginer defines the interface that the login service must implement.
type Loginer interface {
	Login(ctx context.Context, username, password string) (string, error)
}

// LoginRequest represents the JSON body for user login
// swagger:model LoginRequest
type LoginRequest struct {
	// Username
	// required: true
	// default: john_doe
	Username string `json:"username"`

	// Password
	// required: true
	// default: secret123
	Password string `json:"password"`
}

// LoginResponse represents a successful login response
// swagger:model LoginResponse
type LoginResponse struct {
	// JWT token
	// default: JWT_TOKEN
	Token string `json:"token"`
}

// LoginErrorResponse represents an error response for login
// swagger:model LoginErrorResponse
type LoginErrorResponse struct {
	// Error message
	// default: Invalid username or password
	Error string `json:"error"`
}

// NewLoginHandler returns an HTTP handler for user login.
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param loginRequest body handlers.LoginRequest true "Login Request"
// @Success 200 {object} handlers.LoginResponse "JWT token returned"
// @Failure 400 {object} handlers.LoginErrorResponse "Invalid request body"
// @Failure 401 {object} handlers.LoginErrorResponse "Invalid username or password"
// @Router /login [post]
func NewLoginHandler(svc Loginer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(LoginErrorResponse{
				Error: "invalid request body",
			})
			return
		}

		token, err := svc.Login(r.Context(), req.Username, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrInvalidCredentials),
				errors.Is(err, services.ErrUserDoesNotExist):
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(LoginErrorResponse{
					Error: "Invalid username or password",
				})
			default:
				logger.Log.Errorw("internal server error", "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(LoginErrorResponse{
					Error: "Internal server error",
				})
			}
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(LoginResponse{
			Token: token,
		})
	}
}
