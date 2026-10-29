package surat

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
)

// Handler menangani surat: alias penerima, draf, daftar, riwayat, dan alur persetujuan.
type Handler struct {
	letters *Service
}

// NewHandler membuat Handler.
func NewHandler(letters *Service) *Handler {
	return &Handler{letters: letters}
}

// Create menangani POST /api/v1/letters (body boleh kosong = draf kosong).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in LetterInput
	if !httpx.DecodeOptionalJSON(w, r, &in) {
		return
	}
	d, err := h.letters.Create(r.Context(), auth.CurrentUserFrom(r.Context()), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/letters/"+d.ID)
	httpx.SetETag(w, d.RowVersion)
	httpx.JSON(w, http.StatusCreated, d)
}

// Get menangani GET /api/v1/letters/{letterId}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	d, err := h.letters.Get(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "letterId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SetETag(w, d.RowVersion)
	httpx.JSON(w, http.StatusOK, d)
}

// ApprovalQR menangani GET /api/v1/letters/{letterId}/qr (gambar PNG QR persetujuan).
func (h *Handler) ApprovalQR(w http.ResponseWriter, r *http.Request) {
	png, err := h.letters.ApprovalQR(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "letterId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

// Update menangani PUT /api/v1/letters/{letterId} (wajib If-Match, mengganti seluruh isi draf).
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ifMatch, present, err := httpx.ParseIfMatch(r.Header.Get("If-Match"))
	if !present {
		httpx.PreconditionRequired(w)
		return
	}
	if err != nil {
		httpx.Validation(w, "Header If-Match tidak valid. Kirim nilai ETag dari data terakhir.")
		return
	}
	var in LetterInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	d, err := h.letters.Update(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "letterId"), ifMatch, in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SetETag(w, d.RowVersion)
	httpx.JSON(w, http.StatusOK, d)
}

// List menangani GET /api/v1/letters?scope=mine|tasks&status=...&q=...&page=...&pageSize=...
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, err := httpx.ParsePage(query)
	if err != nil {
		httpx.Validation(w, err.Error())
		return
	}
	items, total, err := h.letters.List(r.Context(), auth.CurrentUserFrom(r.Context()), LetterListQuery{
		Scope:    query.Get("scope"),
		Status:   query.Get("status"),
		Search:   query.Get("q"),
		Page:     page.Page,
		PageSize: page.PageSize,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSONPage(w, items, httpx.PageMeta{Page: page.Page, PageSize: page.PageSize, Total: total})
}

// History menangani GET /api/v1/letters/{letterId}/history
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	items, err := h.letters.History(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "letterId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}
