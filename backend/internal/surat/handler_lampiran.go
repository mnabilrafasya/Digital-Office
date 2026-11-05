package surat

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
	"office.backend/internal/platform/middleware"
)

// AddAttachment menangani POST /api/v1/letters/{letterId}/attachments (multipart/form-data,
// field "file"). Response 201 berisi lampiran baru dan ETag surat yang baru.
func (h *Handler) AddAttachment(w http.ResponseWriter, r *http.Request) {
	data, fileName, ok := readUpload(w, r, MaxAttachmentBytes, "lampiran", true)
	if !ok {
		return
	}
	a, rowVer, err := h.letters.AddAttachment(r.Context(), auth.CurrentUserFrom(r.Context()), AttachmentInput{
		LetterID: chi.URLParam(r, "letterId"),
		FileName: fileName,
		Data:     data,
		ClientIP: middleware.ClientIPFrom(r.Context()),
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SetETag(w, rowVer)
	httpx.JSON(w, http.StatusCreated, a)
}

// Attachment menangani GET /api/v1/letters/{letterId}/attachments/{attachmentId} (unduh file).
func (h *Handler) Attachment(w http.ResponseWriter, r *http.Request) {
	f, err := h.letters.OpenAttachment(r.Context(), auth.CurrentUserFrom(r.Context()),
		chi.URLParam(r, "letterId"), chi.URLParam(r, "attachmentId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	sendFile(w, f, "attachment", "lampiran")
}

// RemoveAttachment menangani DELETE /api/v1/letters/{letterId}/attachments/{attachmentId}.
// Response 204 dengan ETag surat yang baru.
func (h *Handler) RemoveAttachment(w http.ResponseWriter, r *http.Request) {
	rowVer, err := h.letters.RemoveAttachment(r.Context(), auth.CurrentUserFrom(r.Context()),
		chi.URLParam(r, "letterId"), chi.URLParam(r, "attachmentId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SetETag(w, rowVer)
	httpx.NoContent(w)
}
