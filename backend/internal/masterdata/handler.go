package masterdata

import (
	"net/http"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
)

// Handler menangani endpoint data acuan.
type Handler struct {
	svc *Service
}

// NewHandler membuat Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Lookups menangani GET /api/v1/lookups?category=...
func (h *Handler) Lookups(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Lookups(r.Context(), auth.CurrentUserFrom(r.Context()), r.URL.Query().Get("category"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}
