package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
)

// RegisterHandler handles user registration
// @Summary Register a new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Register Request"
// @Success 201 {object} models.RegisterResponse
// @Failure 400 {object} models.RegisterErrorResponse
// @Router /register [post]
func NewRegisterHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.RegisterRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(models.RegisterResponse{
			Message: "User registered successfully",
		})
	}
}
