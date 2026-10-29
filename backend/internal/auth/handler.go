package auth

import (
	"net/http"
	"time"

	"office.backend/internal/platform/httpx"
	"office.backend/internal/platform/middleware"
)

// Handler menangani login dan data user yang login.
type Handler struct {
	auth     *Service
	tokenTTL time.Duration
}

// NewHandler membuat Handler.
func NewHandler(auth *Service, tokenTTL time.Duration) *Handler {
	return &Handler{auth: auth, tokenTTL: tokenTTL}
}

type loginRequest struct {
	EmployeeNumber string `json:"employeeNumber"`
	Password       string `json:"password"`
}

type loginResponse struct {
	AccessToken string       `json:"accessToken"`
	TokenType   string       `json:"tokenType"`
	ExpiresIn   int64        `json:"expiresIn"`
	User        *CurrentUser `json:"user"`
}

// Login menangani POST /api/v1/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	res, err := h.auth.Login(r.Context(), LoginInput{
		EmployeeNumber: req.EmployeeNumber,
		Password:       req.Password,
		ClientIP:       middleware.ClientIPFrom(r.Context()),
	})
	if err != nil {
		writeError(w, r, err)
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
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.auth.Logout(r.Context(), CurrentUserFrom(r.Context())); err != nil {
		writeError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// Me menangani GET /api/v1/auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, CurrentUserFrom(r.Context()))
}
