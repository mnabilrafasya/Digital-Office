package organisasi

import (
	"net/http"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
)

// Handler menyediakan data organisasi untuk form Buat Surat.
type Handler struct {
	svc *Service
}

// NewHandler membuat Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Units menangani GET /api/v1/units?q=...
func (h *Handler) Units(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Units(r.Context(), auth.CurrentUserFrom(r.Context()), r.URL.Query().Get("q"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// Candidates menangani GET /api/v1/users/candidates?type=REVIEWER|APPROVER&q=...&limit=...
func (h *Handler) Candidates(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	items, err := h.svc.Candidates(r.Context(), auth.CurrentUserFrom(r.Context()), CandidateQuery{
		Type:   query.Get("type"),
		Search: query.Get("q"),
		Limit:  query.Get("limit"),
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// Aliases menangani GET /api/v1/recipients/aliases
func (h *Handler) Aliases(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Aliases(r.Context(), auth.CurrentUserFrom(r.Context()))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}
