package surat

import (
	"errors"
	"fmt"
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

// Batas request multipart.
const (
	multipartOverhead = 1 << 20  // ruang tambahan di luar isi file (boundary, header, field teks)
	multipartMemory   = 32 << 20 // bagian request yang disimpan di memori; sisanya file sementara
)

// SaveDocument menangani PUT /api/v1/letters/{letterId}/document (multipart/form-data, field
// "file" berisi .docx). Response 201 berisi versi baru dan ETag surat yang baru.
func (h *Handler) SaveDocument(w http.ResponseWriter, r *http.Request) {
	data, fileName, ok := readUpload(w, r, MaxDocumentBytes, "naskah", true)
	if !ok {
		return
	}
	v, rowVer, err := h.letters.SaveDocument(r.Context(), auth.CurrentUserFrom(r.Context()), DocumentInput{
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

// Document menangani GET /api/v1/letters/{letterId}/document?version=&inline= (.docx sebelum
// terbit, PDF resmi sesudahnya). inline=1 mengirim file untuk ditampilkan di browser (tidak
// dicatat sebagai unduhan); tanpa inline file dikirim sebagai unduhan.
func (h *Handler) Document(w http.ResponseWriter, r *http.Request) {
	inline := false
	if raw := r.URL.Query().Get("inline"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			httpx.Validation(w, "Parameter inline harus 1 atau 0")
			return
		}
		inline = v
	}
	version := 0
	if raw := r.URL.Query().Get("version"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			httpx.Validation(w, "Parameter version harus angka mulai dari 1")
			return
		}
		version = n
	}
	open, disposition := h.letters.OpenDocument, "attachment"
	if inline {
		open, disposition = h.letters.ViewDocument, "inline"
	}
	f, err := open(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "letterId"), version)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	sendFile(w, f, disposition, "naskah")
}

// readUpload membaca request multipart/form-data: field "file" (maksimal maxBytes) dan field teks
// lain (dibaca pemanggil lewat formValue). required=false: field file boleh tidak dikirim (data
// nil). ok=false berarti response error sudah ditulis.
func readUpload(w http.ResponseWriter, r *http.Request, maxBytes int, label string, required bool) (data []byte, fileName string, ok bool) {
	invalid := "Kirim " + label + " sebagai multipart/form-data dengan field file"
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes)+multipartOverhead)
	if err := r.ParseMultipartForm(multipartMemory); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.Validation(w, fmt.Sprintf("Ukuran %s maksimal %d MB", label, maxBytes>>20))
			return nil, "", false
		}
		httpx.Validation(w, invalid)
		return nil, "", false
	}
	defer r.MultipartForm.RemoveAll() // field teks tetap bisa dibaca; yang dihapus hanya file sementara

	file, header, err := r.FormFile("file")
	if errors.Is(err, http.ErrMissingFile) && !required {
		return nil, "", true
	}
	if err != nil {
		httpx.Validation(w, invalid)
		return nil, "", false
	}
	defer file.Close()
	if data, err = io.ReadAll(io.LimitReader(file, int64(maxBytes)+1)); err != nil {
		httpx.Validation(w, "File "+label+" tidak bisa dibaca")
		return nil, "", false
	}
	return data, header.Filename, true
}

// formValue membaca field teks multipart; nil bila field tidak dikirim. Dipanggil setelah readUpload.
func formValue(r *http.Request, key string) *string {
	if r.MultipartForm == nil {
		return nil
	}
	v, ok := r.MultipartForm.Value[key]
	if !ok || len(v) == 0 {
		return nil
	}
	return &v[0]
}

// sendFile mengirim file dan menutup Body-nya. disposition "attachment" (diunduh) atau "inline"
// (ditampilkan browser); what dipakai di log.
func sendFile(w http.ResponseWriter, f *DocumentFile, disposition, what string) {
	defer f.Body.Close()
	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(f.Size, 10))
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": f.Name}))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff") // browser wajib mengikuti Content-Type
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, f.Body); err != nil {
		log.Printf("mengirim %s: %v", what, err) // header sudah terkirim, tidak bisa membalas error
	}
}

// FilledDocument menangani GET /api/v1/letters/{letterId}/document/filled (.docx terisi surat
// terbit, untuk ditampilkan editor dalam mode baca dan diekspor menjadi PDF resmi).
func (h *Handler) FilledDocument(w http.ResponseWriter, r *http.Request) {
	f, err := h.letters.FilledDocument(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "letterId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	sendFile(w, f, "inline", "naskah terisi")
}

// SaveFinalPDF menangani PUT /api/v1/letters/{letterId}/document/final (multipart/form-data,
// field "file" berisi PDF hasil ekspor editor). 201 bila PDF resmi baru disimpan, 200 bila sudah
// ada sebelumnya (PDF yang tersimpan tetap dipakai).
func (h *Handler) SaveFinalPDF(w http.ResponseWriter, r *http.Request) {
	data, _, ok := readUpload(w, r, MaxFinalPDFBytes, "PDF resmi", true)
	if !ok {
		return
	}
	v, created, err := h.letters.SaveFinalPDF(r.Context(), auth.CurrentUserFrom(r.Context()), FinalPDFInput{
		LetterID: chi.URLParam(r, "letterId"),
		Data:     data,
		ClientIP: middleware.ClientIPFrom(r.Context()),
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.JSON(w, status, v)
}

// OfficialStatus menangani GET /api/v1/letters/{letterId}/document/status (surat terbit): salinan
// milik siapa yang dilihat user dan apakah PDF resminya sudah ada.
func (h *Handler) OfficialStatus(w http.ResponseWriter, r *http.Request) {
	st, err := h.letters.OfficialStatus(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "letterId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}
