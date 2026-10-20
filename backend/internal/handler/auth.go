package handler

import (
	"net/http"
	"time"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/httpx"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/middleware"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/service"
)

// AuthHandler menangani login dan data user yang login.
type AuthHandler struct {
	auth     *service.AuthService
	tokenTTL time.Duration
}

// NewAuthHandler membuat AuthHandler.
func NewAuthHandler(auth *service.AuthService, tokenTTL time.Duration) *AuthHandler {
	return &AuthHandler{auth: auth, tokenTTL: tokenTTL}
}

type loginRequest struct {
	EmployeeNumber string `json:"employeeNumber"`
	Password       string `json:"password"`
}

type loginResponse struct {
	AccessToken string             `json:"accessToken"`
	TokenType   string             `json:"tokenType"`
	ExpiresIn   int64              `json:"expiresIn"`
	User        *model.CurrentUser `json:"user"`
}

// Login menangani POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	res, err := h.auth.Login(r.Context(), service.LoginInput{
		EmployeeNumber: req.EmployeeNumber,
		Password:       req.Password,
		ClientIP:       middleware.ClientIPFrom(r.Context()),
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	httpx.JSON(w, http.StatusOK, loginResponse{
		AccessToken: res.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(h.tokenTTL.Seconds()),
		User:        res.User,
	})
}

// Logout menangani POST /api/v1/auth/logout: semua token user (semua perangkat) tidak berlaku lagi.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.auth.Logout(r.Context(), middleware.CurrentUser(r.Context())); err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// Me menangani GET /api/v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, middleware.CurrentUser(r.Context()))
}
