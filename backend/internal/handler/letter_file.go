package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/httpx"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/middleware"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

type uploadURLRequest struct {
	FileName string `json:"fileName"`
	Size     int64  `json:"size"`
}

type completeUploadRequest struct {
	UploadToken string `json:"uploadToken"`
}

// AttachmentUploadURL menangani POST /api/v1/letters/{letterId}/attachments/upload-url.
func (h *LetterHandler) AttachmentUploadURL(w http.ResponseWriter, r *http.Request) {
	h.uploadURL(w, r, model.UploadAttachment)
}

// VersionUploadURL menangani POST /api/v1/letters/{letterId}/versions/upload-url.
func (h *LetterHandler) VersionUploadURL(w http.ResponseWriter, r *http.Request) {
	h.uploadURL(w, r, model.UploadVersion)
}

func (h *LetterHandler) uploadURL(w http.ResponseWriter, r *http.Request, kind string) {
	id, ok := letterID(w, r)
	if !ok {
		return
	}
	var req uploadURLRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	ticket, err := h.letters.RequestUpload(r.Context(), middleware.CurrentUser(r.Context()), id, kind, req.FileName, req.Size)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ticket)
}

// CompleteAttachment menangani POST /api/v1/letters/{letterId}/attachments.
func (h *LetterHandler) CompleteAttachment(w http.ResponseWriter, r *http.Request) {
	h.completeUpload(w, r, model.UploadAttachment)
}

// CompleteVersion menangani POST /api/v1/letters/{letterId}/versions.
func (h *LetterHandler) CompleteVersion(w http.ResponseWriter, r *http.Request) {
	h.completeUpload(w, r, model.UploadVersion)
}

// completeUpload menjawab 201 saat pertama kali dicatat dan 200 untuk konfirmasi ulang (idempoten).
func (h *LetterHandler) completeUpload(w http.ResponseWriter, r *http.Request, kind string) {
	id, ok := letterID(w, r)
	if !ok {
		return
	}
	var req completeUploadRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if req.UploadToken == "" {
		httpx.Validation(w, "uploadToken wajib diisi")
		return
	}
	res, err := h.letters.CompleteUpload(r.Context(), middleware.CurrentUser(r.Context()), id, kind, req.UploadToken)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	status := http.StatusOK
	if res.Created {
		status = http.StatusCreated
	}
	httpx.SetETag(w, res.RowVersion)
	if res.Attachment != nil {
		httpx.JSON(w, status, res.Attachment)
		return
	}
	httpx.JSON(w, status, res.Version)
}

// DeleteAttachment menangani DELETE /api/v1/letters/{letterId}/attachments/{attachmentId} (204 + ETag).
func (h *LetterHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, ok := letterID(w, r)
	if !ok {
		return
	}
	rv, err := h.letters.DeleteAttachment(r.Context(), middleware.CurrentUser(r.Context()), id, chi.URLParam(r, "attachmentId"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.SetETag(w, rv)
	httpx.NoContent(w)
}

// AttachmentDownload menangani GET /api/v1/letters/{letterId}/attachments/{attachmentId}/download-url.
func (h *LetterHandler) AttachmentDownload(w http.ResponseWriter, r *http.Request) {
	id, ok := letterID(w, r)
	if !ok {
		return
	}
	link, err := h.letters.AttachmentDownload(r.Context(), middleware.CurrentUser(r.Context()), id, chi.URLParam(r, "attachmentId"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, link)
}

// Versions menangani GET /api/v1/letters/{letterId}/versions.
func (h *LetterHandler) Versions(w http.ResponseWriter, r *http.Request) {
	id, ok := letterID(w, r)
	if !ok {
		return
	}
	items, err := h.letters.Versions(r.Context(), middleware.CurrentUser(r.Context()), id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// VersionDownload menangani GET /api/v1/letters/{letterId}/versions/{versionNumber}/download-url.
func (h *LetterHandler) VersionDownload(w http.ResponseWriter, r *http.Request) {
	id, ok := letterID(w, r)
	if !ok {
		return
	}
	n, err := strconv.Atoi(chi.URLParam(r, "versionNumber"))
	if err != nil || n < 1 {
		httpx.NotFoundMessage(w, "Versi dokumen tidak ditemukan")
		return
	}
	link, err := h.letters.VersionDownload(r.Context(), middleware.CurrentUser(r.Context()), id, n)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, link)
}
