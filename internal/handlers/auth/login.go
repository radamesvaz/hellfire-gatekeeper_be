package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/radamesvaz/bakery-app/internal/errors"
	v "github.com/radamesvaz/bakery-app/internal/handlers/validators"
	userRepo "github.com/radamesvaz/bakery-app/internal/repository/user"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	uModel "github.com/radamesvaz/bakery-app/model/users"
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

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	Token   string `json:"token"`
	Message string `json:"message"`
}

func (lh *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	req := LoginRequest{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate email
	if valid := v.IsValidEmail(req.Email); !valid {
		http.Error(w, "Invalid Email", http.StatusBadRequest)
		return
	}

	// Login without tenant in path: use default tenant (1). Later, use /t/{tenant_slug}/login and resolve from context.
	const defaultTenantID = 1
	user, err := lh.UserRepo.GetUserByEmail(defaultTenantID, req.Email)
	if err != nil {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	if err := lh.AuthService.ComparePasswords(user.Password, req.Password); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	idRole := uModel.UserRole(user.IDRole)
	tenantID := user.TenantID
	token, err := lh.AuthService.GenerateJWT(user.ID, idRole, user.Email, &tenantID)
	if err != nil {
		http.Error(w, "Could not generate token", http.StatusInternalServerError)
		return
	}

	resp := LoginResponse{Token: token}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (lh *LoginHandler) Register(w http.ResponseWriter, r *http.Request) {
	req := RegisterRequest{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	// Validate email
	if valid := v.IsValidEmail(req.Email); !valid {
		http.Error(w, "Invalid Email", http.StatusBadRequest)
		return
	}

	// Validate password strength
	if err := v.ValidatePassword(req.Password); err != nil {
		if httpErr, ok := err.(*errors.HTTPError); ok {
			http.Error(w, httpErr.Error(), httpErr.StatusCode)
			return
		}
		http.Error(w, "Password does not meet security requirements", http.StatusBadRequest)
		return
	}

	// Check if email already exists (default tenant until path-based tenant for register)
	const defaultTenantID = 1
	exists, err := lh.UserRepo.EmailExists(defaultTenantID, req.Email)
	if err != nil {
		if httpErr, ok := err.(*errors.HTTPError); ok {
			http.Error(w, httpErr.Error(), httpErr.StatusCode)
			return
		}
		http.Error(w, "Error checking email", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	// Hash password
	hashedPassword, err := lh.AuthService.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Error processing password", http.StatusInternalServerError)
		return
	}

	// Create user (default tenant until path-based tenant for register)
	createUserReq := uModel.CreateUserRequest{
		TenantID: defaultTenantID,
		IDRole:   uModel.UserRoleAdmin, // Default role for new users (administrators)
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		Password: hashedPassword,
	}

	userID, err := lh.UserRepo.CreateUser(r.Context(), createUserReq)
	if err != nil {
		if httpErr, ok := err.(*errors.HTTPError); ok {
			http.Error(w, httpErr.Error(), httpErr.StatusCode)
			return
		}
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	// Generate token for the new user
	tenantIDPtr := func(v uint64) *uint64 { return &v }(defaultTenantID)
	token, err := lh.AuthService.GenerateJWT(userID, uModel.UserRoleAdmin, req.Email, tenantIDPtr)
	if err != nil {
		http.Error(w, "Could not generate token", http.StatusInternalServerError)
		return
	}

	resp := RegisterResponse{
		Token:   token,
		Message: "Admin user registered successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
