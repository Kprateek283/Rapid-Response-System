package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	authDto "github.com/google-hackathon/rapid-response/internal/api/dto/auth"
	authService "github.com/google-hackathon/rapid-response/internal/service/auth"
)

// 1. Create the struct
type AuthHandler struct {
	service *authService.AuthService
}

// 2. Create the constructor (This fixes the 'NewAuthHandler' error in main.go)
func NewAuthHandler(service *authService.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// 3. Attach all functions as methods to the struct (h *AuthHandler)
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req authDto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.service.Authenticate(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error":"Invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	if err := h.service.BlacklistToken(r.Context(), tokenString); err != nil {
		http.Error(w, `{"error":"Failed to logout"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"Successfully logged out"}`))
}

func (h *AuthHandler) CreateGroupManager(w http.ResponseWriter, r *http.Request) {
	var req authDto.CreateManagerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.service.CreateUser(r.Context(), req, "GROUP_MANAGER")
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) CreateHotelManager(w http.ResponseWriter, r *http.Request) {
	var req authDto.CreateManagerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.service.CreateUser(r.Context(), req, "HOTEL_MANAGER")
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
