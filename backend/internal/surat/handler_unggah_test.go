package surat

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
	"office.backend/internal/platform/upload"
)

// multipartBody membuat body multipart berisi field teks dan (bila fileName tidak kosong) file.
func multipartBody(fields map[string]string, fileName string, data []byte) (*bytes.Buffer, string) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	if fileName != "" {
		fw, _ := mw.CreateFormFile("file", fileName)
		fw.Write(data)
	}
	mw.Close()
	return &body, mw.FormDataContentType()
}

func uploadRequest(method, path string, fields map[string]string, fileName string, data []byte, u *auth.CurrentUser) *http.Request {
	body, ct := multipartBody(fields, fileName, data)
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", ct)
	return req.WithContext(auth.WithCurrentUser(req.Context(), u))
}

func TestUploadQueueHandler(t *testing.T) {
	store := letterAt(StatusDraft, 0, 0)
	limiter := upload.NewLimiter(2<<20, 200*time.Millisecond) // jatah kecil supaya mudah penuh
	h := NewHandler(newTestLetterService(store, newFakeRefs()), UploadOptions{Limiter: limiter, ReadTimeout: time.Minute, Timeout: 2 * time.Minute})
	r := chi.NewRouter()
	r.Put("/letters/{letterId}/document", h.SaveDocument)
	path := "/letters/" + testLetterID + "/document"
	send := func(fileName string, data []byte) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, uploadRequest(http.MethodPut, path, nil, fileName, data, andiUser))
		return rec
	}

	if rec := send("Naskah.docx", testDocx); rec.Code != http.StatusCreated || len(store.versions) != 1 {
		t.Fatalf("unggahan biasa: status=%d body=%s", rec.Code, rec.Body)
	}

	// Jatah sedang dipakai unggahan lain: request menunggu lalu ditolak 503 dengan saran coba lagi
	hold, _ := limiter.Acquire(context.Background(), 2<<20)
	start := time.Now()
	rec := send("Naskah.docx", testDocx)
	if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" ||
		!strings.Contains(rec.Body.String(), "Server sedang memproses banyak unggahan") || time.Since(start) < 200*time.Millisecond {
		t.Errorf("antrean penuh: status=%d retry=%q body=%s", rec.Code, rec.Header().Get("Retry-After"), rec.Body)
	}
	hold()

	// Penolakan validasi tetap mengembalikan jatah (request berikutnya langsung jalan)
	for name, tc := range map[string]struct {
		file string
		data []byte
		want string
	}{
		"tanpa file":       {"", nil, "multipart/form-data dengan field file"},
		"lebih dari 25 MB": {"Besar.docx", append(append([]byte{}, testDocx...), make([]byte, MaxUploadBytes+multipartOverhead)...), "maksimal 25 MB"},
		"bukan docx":       {"Naskah.pdf", testDocx, "wajib berupa file .docx"},
	} {
		if rec := send(tc.file, tc.data); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), tc.want) {
			t.Errorf("%s: status=%d body=%s", name, rec.Code, rec.Body)
		}
	}
	if rec := send("Naskah2.docx", testDocx); rec.Code != http.StatusCreated {
		t.Errorf("jatah tidak kembali setelah request gagal: status=%d", rec.Code)
	}
	if release, err := limiter.Acquire(context.Background(), 2<<20); err != nil {
		t.Errorf("seluruh jatah harus kembali: %v", err)
	} else {
		release()
	}
}

func TestUploadTemplateFormFields(t *testing.T) {
	refs := newFakeRefs()
	h := NewHandler(newTestLetterService(&fakeLetterStore{}, refs), UploadOptions{Limiter: upload.NewLimiter(50<<20, time.Second)})
	r := chi.NewRouter()
	r.Post("/templates", h.CreateTemplate)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, uploadRequest(http.MethodPost, "/templates",
		map[string]string{"name": "Memo Pabrik", "description": "Memo singkat", "letterTypeId": "12"}, "memo.docx", testDocx, templateUser))
	var got struct{ Data Template }
	json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != http.StatusCreated || got.Data.Name != "Memo Pabrik" || got.Data.Description == nil ||
		*got.Data.Description != "Memo singkat" || got.Data.LetterType == nil {
		t.Errorf("isian form template tidak terbaca: status=%d body=%s", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, uploadRequest(http.MethodPost, "/templates",
		map[string]string{"name": strings.Repeat("x", maxFieldBytes+1)}, "memo.docx", testDocx, templateUser))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "terlalu panjang") {
		t.Errorf("isian terlalu panjang: status=%d body=%s", rec.Code, rec.Body)
	}
}
