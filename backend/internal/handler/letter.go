package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/httpx"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/middleware"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/service"
)

// LetterHandler menangani draf surat dan alias penerima.
type LetterHandler struct {
	letters *service.LetterService
}

// NewLetterHandler membuat LetterHandler.
func NewLetterHandler(letters *service.LetterService) *LetterHandler {
	return &LetterHandler{letters: letters}
}

// Aliases menangani GET /api/v1/recipients/aliases
func (h *LetterHandler) Aliases(w http.ResponseWriter, r *http.Request) {
	items, err := h.letters.Aliases(r.Context(), middleware.CurrentUser(r.Context()))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// Create menangani POST /api/v1/letters (body boleh kosong = draf kosong).
func (h *LetterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in service.LetterInput
	if !httpx.DecodeOptionalJSON(w, r, &in) {
		return
	}
	d, err := h.letters.Create(r.Context(), middleware.CurrentUser(r.Context()), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/letters/"+d.ID)
	httpx.SetETag(w, d.RowVersion)
	httpx.JSON(w, http.StatusCreated, d)
}

// Get menangani GET /api/v1/letters/{letterId}
func (h *LetterHandler) Get(w http.ResponseWriter, r *http.Request) {
	d, err := h.letters.Get(r.Context(), middleware.CurrentUser(r.Context()), chi.URLParam(r, "letterId"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.SetETag(w, d.RowVersion)
	httpx.JSON(w, http.StatusOK, d)
}

// Update menangani PUT /api/v1/letters/{letterId} (wajib If-Match, mengganti seluruh isi draf).
func (h *LetterHandler) Update(w http.ResponseWriter, r *http.Request) {
	ifMatch, present, err := httpx.ParseIfMatch(r.Header.Get("If-Match"))
	if !present {
		httpx.PreconditionRequired(w)
		return
	}
	if err != nil {
		httpx.Validation(w, "Header If-Match tidak valid. Kirim nilai ETag dari data terakhir.")
		return
	}
	var in service.LetterInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	d, err := h.letters.Update(r.Context(), middleware.CurrentUser(r.Context()), chi.URLParam(r, "letterId"), ifMatch, in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.SetETag(w, d.RowVersion)
	httpx.JSON(w, http.StatusOK, d)
}
