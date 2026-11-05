package surat

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
)

// withFinalPDF menyimpan PDF resmi untuk surat dari sentLetter, seolah sudah diekspor editor.
func withFinalPDF(t *testing.T, svc *Service) []byte {
	t.Helper()
	data := minimalPDF(7)
	if _, _, err := svc.SaveFinalPDF(context.Background(), andiUser, FinalPDFInput{LetterID: testLetterID, Data: data}); err != nil {
		t.Fatalf("SaveFinalPDF: %v", err)
	}
	return data
}

func TestViewDocumentNotRecordedAsDownload(t *testing.T) {
	svc, _, _, inbox, audit := sentLetter(true)
	want := withFinalPDF(t, svc)
	inbox.holders = map[int64]bool{50: true}
	recipient := &auth.CurrentUser{ID: 50, Company: auth.UserCompany{Code: "PSP"}}
	ctx := context.Background()

	for _, u := range []*auth.CurrentUser{andiUser, recipient} {
		f, err := svc.ViewDocument(ctx, u, testLetterID, 0)
		if err != nil {
			t.Fatalf("user %d: %v", u.ID, err)
		}
		got, _ := io.ReadAll(f.Body)
		if f.ContentType != PDFContentType || !bytes.Equal(got, want) {
			t.Errorf("user %d: tampilan bukan PDF resmi: %+v", u.ID, f)
		}
	}
	for _, e := range audit.entries {
		if e.Action == "DOWNLOAD" {
			t.Errorf("tampilan tercatat sebagai unduhan: %+v", e)
		}
	}
	if _, err := svc.ViewDocument(ctx, outsiderUser, testLetterID, 0); !errors.Is(err, ErrLetterNotFound) {
		t.Errorf("orang luar: error = %v", err)
	}
}

// documentRouter menyusun route naskah surat seperti di cmd/api, untuk uji handler.
func documentRouter(svc *Service) http.Handler {
	h := NewHandler(svc, UploadOptions{})
	r := chi.NewRouter()
	r.Get("/letters/{letterId}/document", h.Document)
	r.Get("/letters/{letterId}/document/filled", h.FilledDocument)
	r.Put("/letters/{letterId}/document/final", h.SaveFinalPDF)
	r.Get("/letters/{letterId}/document/status", h.OfficialStatus)
	return r
}

func serve(r http.Handler, req *http.Request) *httptest.ResponseRecorder {
	req = req.WithContext(auth.WithCurrentUser(req.Context(), andiUser))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestDocumentHandlerDisposition(t *testing.T) {
	svc, _, _, _, audit := sentLetter(true)
	withFinalPDF(t, svc)
	r := documentRouter(svc)

	for _, tc := range []struct {
		query, disposition string
		downloads          int // jumlah DOWNLOAD di audit setelah request
	}{
		{"?inline=1", "inline", 0},
		{"?inline=true", "inline", 0},
		{"", "attachment", 1},
		{"?inline=0", "attachment", 2},
	} {
		rec := serve(r, httptest.NewRequest(http.MethodGet, "/letters/"+testLetterID+"/document"+tc.query, nil))
		cd := rec.Header().Get("Content-Disposition")
		if rec.Code != http.StatusOK || !strings.HasPrefix(cd, tc.disposition+";") || !strings.Contains(cd, "015-PSP-X-2026.pdf") ||
			rec.Header().Get("Content-Type") != PDFContentType || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%q: status=%d header=%v", tc.query, rec.Code, rec.Header())
		}
		n := 0
		for _, e := range audit.entries {
			if e.Action == "DOWNLOAD" {
				n++
			}
		}
		if n != tc.downloads {
			t.Errorf("%q: DOWNLOAD di audit = %d, want %d", tc.query, n, tc.downloads)
		}
	}
	if rec := serve(r, httptest.NewRequest(http.MethodGet, "/letters/"+testLetterID+"/document?inline=ya", nil)); rec.Code != http.StatusBadRequest {
		t.Errorf("inline tidak valid: status = %d, want 400", rec.Code)
	}
}

func TestFinalPDFHandlers(t *testing.T) {
	svc, _, _, _, _ := sentLetter(true)
	r := documentRouter(svc)
	base := "/letters/" + testLetterID + "/document"

	// Sebelum PDF diunggah: unduhan 409, status belum siap, naskah terisi tersedia
	if rec := serve(r, httptest.NewRequest(http.MethodGet, base+"/status", nil)); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"pdfReady":false`) || !strings.Contains(rec.Body.String(), `"copyFor":null`) {
		t.Errorf("status sebelum PDF: %d %s", rec.Code, rec.Body)
	}
	if rec := serve(r, httptest.NewRequest(http.MethodGet, base, nil)); rec.Code != http.StatusConflict {
		t.Errorf("unduh sebelum PDF ada: status = %d, want 409", rec.Code)
	}
	rec := serve(r, httptest.NewRequest(http.MethodGet, base+"/filled", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != DocxContentType ||
		!strings.HasPrefix(rec.Header().Get("Content-Disposition"), "inline;") || !bytes.HasPrefix(rec.Body.Bytes(), zipSignature) {
		t.Errorf("naskah terisi: status=%d header=%v", rec.Code, rec.Header())
	}

	upload := func(data []byte) *httptest.ResponseRecorder {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		fw, _ := mw.CreateFormFile("file", "naskah.pdf")
		fw.Write(data)
		mw.Close()
		req := httptest.NewRequest(http.MethodPut, base+"/final", &body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		return serve(r, req)
	}
	first := minimalPDF(3)
	if rec := upload(first); rec.Code != http.StatusCreated {
		t.Fatalf("unggah pertama: status = %d, body %s", rec.Code, rec.Body)
	}
	if rec := upload(minimalPDF(4)); rec.Code != http.StatusOK {
		t.Errorf("unggah kedua: status = %d, want 200 (PDF pertama tetap dipakai)", rec.Code)
	}
	if rec := upload([]byte("bukan pdf")); rec.Code != http.StatusBadRequest {
		t.Errorf("bukan PDF: status = %d, want 400", rec.Code)
	}
	if rec := serve(r, httptest.NewRequest(http.MethodGet, base+"/status", nil)); !strings.Contains(rec.Body.String(), `"pdfReady":true`) {
		t.Errorf("status setelah PDF: %s", rec.Body)
	}
	if rec := serve(r, httptest.NewRequest(http.MethodGet, base, nil)); rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), first) {
		t.Errorf("unduhan bukan PDF pertama: status=%d", rec.Code)
	}
}
