package surat

import (
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
	"office.backend/internal/platform/middleware"
)

// multipartOverhead adalah ruang tambahan body multipart di luar isi file (boundary, header).
const multipartOverhead = 1 << 20

// SaveDocument menangani PUT /api/v1/letters/{letterId}/document (multipart/form-data, field
// "file" berisi .docx). Response 201 berisi versi baru dan ETag surat yang baru.
func (h *Handler) SaveDocument(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxDocumentBytes+multipartOverhead)
	file, header, err := r.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.Validation(w, "Ukuran naskah maksimal 10 MB")
			return
		}
		httpx.Validation(w, "Kirim naskah sebagai multipart/form-data dengan field file")
		return
	}
	defer file.Close()
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxDocumentBytes+1))
	if err != nil {
		httpx.Validation(w, "File naskah tidak bisa dibaca")
		return
	}

	v, rowVer, err := h.letters.SaveDocument(r.Context(), auth.CurrentUserFrom(r.Context()), DocumentInput{
		LetterID: chi.URLParam(r, "letterId"),
		FileName: header.Filename,
		Data:     data,
		ClientIP: middleware.ClientIPFrom(r.Context()),
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SetETag(w, rowVer)
	httpx.JSON(w, http.StatusCreated, v)
}

// Versions menangani GET /api/v1/letters/{letterId}/versions
func (h *Handler) Versions(w http.ResponseWriter, r *http.Request) {
	items, err := h.letters.Versions(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "letterId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// Document menangani GET /api/v1/letters/{letterId}/document?version= (unduh .docx).
func (h *Handler) Document(w http.ResponseWriter, r *http.Request) {
	version := 0
	if raw := r.URL.Query().Get("version"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			httpx.Validation(w, "Parameter version harus angka mulai dari 1")
			return
		}
		version = n
	}
	f, err := h.letters.OpenDocument(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "letterId"), version)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	defer f.Body.Close()

	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(f.Size, 10))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": f.Name}))
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, f.Body); err != nil {
		log.Printf("mengirim naskah: %v", err) // header sudah terkirim, tidak bisa membalas error
	}
}
