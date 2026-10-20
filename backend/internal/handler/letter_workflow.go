package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/httpx"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/middleware"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/service"
)

// transitionFunc adalah satu aksi alur persetujuan di LetterService.
type transitionFunc func(ctx context.Context, user *model.CurrentUser, in service.TransitionInput) (*model.LetterDetail, error)

// noteBody adalah body opsional aksi approve/return.
type noteBody struct {
	Note *string `json:"note"`
}

// Submit menangani POST /api/v1/letters/{letterId}/submit (tanpa body).
func (h *LetterHandler) Submit(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.letters.Submit, false)
}

// Approve menangani POST /api/v1/letters/{letterId}/approve dengan body opsional {"note"}.
func (h *LetterHandler) Approve(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.letters.Approve, true)
}

// Return menangani POST /api/v1/letters/{letterId}/return dengan body {"note"} (catatan wajib).
func (h *LetterHandler) Return(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.letters.Return, true)
}

// transition menjalankan aksi alur lalu menjawab 200 + detail surat terbaru + ETag.
func (h *LetterHandler) transition(w http.ResponseWriter, r *http.Request, run transitionFunc, readNote bool) {
	var body noteBody
	if readNote && !httpx.DecodeOptionalJSON(w, r, &body) {
		return
	}
	d, err := run(r.Context(), middleware.CurrentUser(r.Context()), service.TransitionInput{
		LetterID: chi.URLParam(r, "letterId"),
		Note:     body.Note,
		ClientIP: middleware.ClientIPFrom(r.Context()),
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.SetETag(w, d.RowVersion)
	httpx.JSON(w, http.StatusOK, d)
}
