package auth

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	userRepo "github.com/radamesvaz/bakery-app/internal/repository/user"
	"github.com/radamesvaz/bakery-app/internal/services/auth"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
)

type LoginHandler struct {
	UserRepo    userRepo.UserRepository
	AuthService authService.AuthService
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (lh *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	req := LoginRequest{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, err := lh.UserRepo.GetUserByEmail(req.Email)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	if err := lh.AuthService.ComparePasswords(user.Password, req.Password); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	secret := os.Getenv("JWT_SECRET")
	expMinutes := os.Getenv("JWT_EXPIRATION_MINUTES")
	exp, err := strconv.Atoi(expMinutes)
	if err != nil {
		http.Error(w, "could not get the expMinutes from env", http.StatusInternalServerError)
		return
	}

	authService := auth.New(secret, exp)

	token, err := authService.GenerateJWT(user.ID, user.IDRole, user.Email)
	if err != nil {
		http.Error(w, "Could not generate token", http.StatusInternalServerError)
		return
	}

	resp := LoginResponse{Token: token}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
