package surat

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
	"office.backend/internal/platform/middleware"
)

// transitionFunc adalah satu aksi alur persetujuan di Service.
type transitionFunc func(ctx context.Context, user *auth.CurrentUser, in TransitionInput) (*LetterDetail, error)

// noteBody adalah body opsional aksi approve/return.
type noteBody struct {
	Note *string `json:"note"`
}

// Submit menangani POST /api/v1/letters/{letterId}/submit (tanpa body).
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.letters.Submit, false)
}

// Approve menangani POST /api/v1/letters/{letterId}/approve dengan body opsional {"note"}.
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.letters.Approve, true)
}

// Return menangani POST /api/v1/letters/{letterId}/return dengan body {"note"} (catatan wajib).
func (h *Handler) Return(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.letters.Return, true)
}

// transition menjalankan aksi alur lalu menjawab 200 + detail surat terbaru + ETag.
func (h *Handler) transition(w http.ResponseWriter, r *http.Request, run transitionFunc, readNote bool) {
	var body noteBody
	if readNote && !httpx.DecodeOptionalJSON(w, r, &body) {
		return
	}
	d, err := run(r.Context(), auth.CurrentUserFrom(r.Context()), TransitionInput{
		LetterID: chi.URLParam(r, "letterId"),
		Note:     body.Note,
		ClientIP: middleware.ClientIPFrom(r.Context()),
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SetETag(w, d.RowVersion)
	httpx.JSON(w, http.StatusOK, d)
}
