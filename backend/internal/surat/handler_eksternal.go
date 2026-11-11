package surat

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
	"office.backend/internal/platform/middleware"
)

// UploadSigned menangani PUT /api/v1/letters/{letterId}/document/signed (multipart/form-data: file
// PDF scan bertanda tangan basah). Hanya approver terakhir surat eksternal berstatus APPROVED.
// Response 200 berisi detail surat (status SENT).
func (h *Handler) UploadSigned(w http.ResponseWriter, r *http.Request) {
	form, ok := h.readUpload(w, r, MaxFinalPDFBytes, "Scan surat", true)
	if !ok {
		return
	}
	defer form.release()
	d, err := h.letters.UploadSigned(r.Context(), auth.CurrentUserFrom(r.Context()), FinalPDFInput{
		LetterID: chi.URLParam(r, "letterId"),
		Data:     form.data,
		ClientIP: middleware.ClientIPFrom(r.Context()),
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SetETag(w, d.RowVersion)
	httpx.JSON(w, http.StatusOK, d)
}

// CreateIncoming menangani POST /api/v1/letters/incoming (multipart/form-data): file (PDF scan),
// title, sender, referenceNumber, letterDate (YYYY-MM-DD), userIds dan aliasLevelIds (ID dipisah
// koma), note. Response 201 berisi detail surat masuk (status RECEIVED).
func (h *Handler) CreateIncoming(w http.ResponseWriter, r *http.Request) {
	form, ok := h.readUpload(w, r, MaxFinalPDFBytes, "Scan surat", true)
	if !ok {
		return
	}
	defer form.release()
	text := func(key string) string {
		if v := form.value(key); v != nil {
			return *v
		}
		return ""
	}
	d, err := h.letters.CreateIncoming(r.Context(), auth.CurrentUserFrom(r.Context()), IncomingInput{
		Title:           text("title"),
		Sender:          text("sender"),
		ReferenceNumber: text("referenceNumber"),
		LetterDate:      text("letterDate"),
		UserIDs:         strings.Split(text("userIds"), ","),
		AliasLevelIDs:   strings.Split(text("aliasLevelIds"), ","),
		Note:            form.value("note"),
		FileName:        form.fileName,
		Data:            form.data,
		ClientIP:        middleware.ClientIPFrom(r.Context()),
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SetETag(w, d.RowVersion)
	httpx.JSON(w, http.StatusCreated, d)
}
