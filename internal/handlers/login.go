package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

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

// Loginer defines the interface that the login service must implement.
type Loginer interface {
	Login(ctx context.Context, username, password string) (string, error)
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
func NewLoginHandler(svc Loginer, log *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest

		log.Infow("decoding login request", "method", r.Method, "path", r.URL.Path)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Errorw("failed to decode request body", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(LoginErrorResponse{
				Error: "invalid request body",
			})
			return
		}

		log.Infow("calling login service", "username", req.Username)
		token, err := svc.Login(r.Context(), req.Username, req.Password)
		if err != nil {
			log.Warnw("login failed", "username", req.Username, "error", err)
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(LoginErrorResponse{
				Error: "Invalid username or password",
			})
			return
		}

		log.Infow("login successful", "username", req.Username)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(LoginResponse{
			Token: token,
		})
	}
}
