package handler

import (
	"net/http"
	"time"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/httpx"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/middleware"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/service"
)

type AuthHandler struct {
	auth     *service.AuthService
	tokenTTL time.Duration
}

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

	res, err := h.auth.Login(r.Context(), req.EmployeeNumber, req.Password)
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

// Me menangani GET /api/v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, middleware.CurrentUser(r.Context()))
}
