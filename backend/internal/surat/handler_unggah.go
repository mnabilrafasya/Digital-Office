package surat

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"office.backend/internal/platform/httpx"
	"office.backend/internal/platform/upload"
)

// Semua unggahan (naskah, template, lampiran, PDF resmi) dibaca lewat readUpload: menunggu giliran
// di antrean unggahan (jatah memori bersama), lalu membaca field file sekali ke memori tanpa file
// sementara. Request unggahan mendapat batas waktu sendiri yang lebih panjang dari request lain,
// supaya file besar dari jaringan lambat dan waktu menunggu giliran tidak terputus.

// Batas request multipart.
const (
	multipartOverhead = 1 << 20  // ruang tambahan di luar isi file (boundary, header, field teks)
	maxFieldBytes     = 64 << 10 // field teks (nama, deskripsi template, dan sejenisnya)
	initialBufBytes   = 64 << 10 // kapasitas awal bila ukuran request tidak diketahui
	busyRetryAfter    = "5"      // saran coba lagi (detik) saat antrean penuh
)

// UploadOptions mengatur antrean dan batas waktu request unggahan. Limiter nil = tanpa antrean,
// timeout 0 = memakai batas waktu server biasa.
type UploadOptions struct {
	Limiter     *upload.Limiter
	ReadTimeout time.Duration // batas waktu mengirim isi request
	Timeout     time.Duration // batas waktu total request (termasuk menunggu giliran)
}

// uploadForm adalah isi request unggahan.
type uploadForm struct {
	data     []byte // nil = field file tidak dikirim
	fileName string
	values   map[string]string
	release  func() // mengembalikan jatah antrean; wajib dipanggil saat request selesai
}

// value mengembalikan field teks; nil bila field tidak dikirim.
func (f *uploadForm) value(key string) *string {
	v, ok := f.values[key]
	if !ok {
		return nil
	}
	return &v
}

// readUpload membaca request multipart/form-data: field "file" (maksimal maxBytes) dan field teks
// lain. fileRequired=false: field file boleh tidak dikirim. ok=false berarti response error sudah
// ditulis; selain itu pemanggil wajib memanggil form.release (defer) setelah unggahan diproses.
func (h *Handler) readUpload(w http.ResponseWriter, r *http.Request, maxBytes int, label string, fileRequired bool) (form *uploadForm, ok bool) {
	rc := http.NewResponseController(w)
	now := time.Now()
	if h.uploads.ReadTimeout > 0 {
		_ = rc.SetReadDeadline(now.Add(h.uploads.ReadTimeout)) // HTTP/2 dan test tidak mendukung: abaikan
	}
	if h.uploads.Timeout > 0 {
		_ = rc.SetWriteDeadline(now.Add(h.uploads.Timeout))
	}

	limit := int64(maxBytes) + multipartOverhead
	reserve := limit
	if r.ContentLength > 0 && r.ContentLength < limit {
		reserve = r.ContentLength
	}
	release := func() {}
	if h.uploads.Limiter != nil {
		var err error
		if release, err = h.uploads.Limiter.Acquire(r.Context(), reserve); err != nil {
			if errors.Is(err, upload.ErrBusy) {
				w.Header().Set("Retry-After", busyRetryAfter)
			}
			httpx.WriteError(w, r, err)
			return nil, false
		}
	}
	defer func() {
		if !ok {
			release()
		}
	}()

	r.Body = http.MaxBytesReader(w, r.Body, limit)
	invalid := "Kirim " + label + " sebagai multipart/form-data dengan field file"
	mr, err := r.MultipartReader()
	if err != nil {
		httpx.Validation(w, invalid)
		return nil, false
	}
	form = &uploadForm{values: map[string]string{}, release: release}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			uploadReadError(w, err, label, maxBytes, invalid)
			return nil, false
		}
		name, isFile := part.FormName(), part.FileName() != ""
		switch {
		case name == "file" && isFile && form.data == nil:
			// Kapasitas awal sebesar request: isi file langsung masuk tanpa penyalinan ulang
			buf := bytes.NewBuffer(make([]byte, 0, min(reserve, limit, max(r.ContentLength, initialBufBytes))))
			if _, err := buf.ReadFrom(io.LimitReader(part, int64(maxBytes)+1)); err != nil {
				uploadReadError(w, err, label, maxBytes, "File "+label+" tidak bisa dibaca")
				return nil, false
			}
			form.data, form.fileName = buf.Bytes(), part.FileName()
		case name != "" && !isFile:
			v, err := io.ReadAll(io.LimitReader(part, maxFieldBytes+1))
			if err != nil || len(v) > maxFieldBytes {
				uploadReadError(w, err, label, maxBytes, "Isian "+name+" terlalu panjang")
				return nil, false
			}
			form.values[name] = string(v)
		}
		part.Close()
	}
	if form.data == nil && fileRequired {
		httpx.Validation(w, invalid)
		return nil, false
	}
	return form, true
}

// uploadReadError menulis error saat membaca request unggahan: request melebihi batas ukuran
// dijawab dengan batas ukurannya, selain itu message.
func uploadReadError(w http.ResponseWriter, err error, label string, maxBytes int, message string) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		httpx.Validation(w, fmt.Sprintf("Ukuran %s maksimal %d MB", label, maxBytes>>20))
		return
	}
	httpx.Validation(w, message)
}
