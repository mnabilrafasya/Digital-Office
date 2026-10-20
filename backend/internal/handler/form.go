package handler

import (
	"net/http"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/httpx"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/middleware"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/service"
)

// FormHandler menyediakan data untuk form Buat Surat.
type FormHandler struct {
	form *service.FormService
}

// NewFormHandler membuat FormHandler.
func NewFormHandler(form *service.FormService) *FormHandler {
	return &FormHandler{form: form}
}

// Lookups menangani GET /api/v1/lookups?category=...
func (h *FormHandler) Lookups(w http.ResponseWriter, r *http.Request) {
	items, err := h.form.Lookups(r.Context(), middleware.CurrentUser(r.Context()), r.URL.Query().Get("category"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// Units menangani GET /api/v1/units?q=...
func (h *FormHandler) Units(w http.ResponseWriter, r *http.Request) {
	items, err := h.form.Units(r.Context(), middleware.CurrentUser(r.Context()), r.URL.Query().Get("q"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// Aliases menangani GET /api/v1/recipients/aliases
func (h *FormHandler) Aliases(w http.ResponseWriter, r *http.Request) {
	items, err := h.form.Aliases(r.Context(), middleware.CurrentUser(r.Context()))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// Candidates menangani GET /api/v1/users/candidates?type=REVIEWER|APPROVER&q=...&limit=...
func (h *FormHandler) Candidates(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	items, err := h.form.Candidates(r.Context(), middleware.CurrentUser(r.Context()), service.CandidateQuery{
		Type:   query.Get("type"),
		Search: query.Get("q"),
		Limit:  query.Get("limit"),
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}
