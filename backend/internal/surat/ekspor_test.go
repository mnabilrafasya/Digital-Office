package surat

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// sentLetter menyiapkan surat terbit (nomor 015/TI/PSP/X/2026, disetujui Agus) yang naskah
// .docx-nya sudah tersimpan sebagai versi 1, beserta service dengan pembuat PDF palsu.
func sentLetter(withDraft bool) (*Service, *fakeLetterStore, *fakePDF, *fakeInbox) {
	store := letterAt(StatusSent, 0, 4)
	store.detail.LetterNumber = ptr("015/TI/PSP/X/2026")
	store.detail.Approval = &Approval{ApprovedAt: time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC),
		ApprovedBy: PersonRef{ID: 3, Name: "Agus", PositionName: "SVP"}}
	files, pdf, inbox := &fakeFiles{}, &fakePDF{}, &fakeInbox{}
	if withDraft {
		files.objects = map[string][]byte{"draf-1": makeDocx("<w:t>Nomor: ###NoSurat###</w:t>")}
		store.versions = []DocumentVersion{{VersionNumber: 1, Type: VersionDraft, FileName: "Naskah.docx", ObjectKey: "draf-1"}}
	}
	refs := newFakeRefs()
	svc := NewService(fakeTx{}, nil, Repos{
		Letters: func(db.DBTX) Store { return store }, Audit: func(db.DBTX) AuditStore { return &fakeAudit{} },
		Inbox: func(db.DBTX) InboxStore { return inbox }, Directory: refs, Lookups: refs, Templates: refs,
		Files: files, PDF: pdf,
	}, time.UTC)
	return svc, store, pdf, inbox
}

func TestFinalPDFCreatedOnceAndReused(t *testing.T) {
	svc, store, pdf, inbox := sentLetter(true)
	inbox.holders = map[int64]bool{50: true}
	recipient := &auth.CurrentUser{ID: 50, Company: auth.UserCompany{Code: "PSP"}}
	ctx := context.Background()

	for _, u := range []*auth.CurrentUser{andiUser, recipient, agusUser} {
		f, err := svc.OpenDocument(ctx, u, testLetterID, 0)
		if err != nil {
			t.Fatalf("user %d: %v", u.ID, err)
		}
		got, _ := io.ReadAll(f.Body)
		if !bytes.HasPrefix(got, []byte("%PDF:PK")) || !bytes.HasSuffix(got, []byte("+QR")) ||
			!bytes.Contains(got, []byte("Nomor: 015/TI/PSP/X/2026")) || bytes.Contains(got, []byte("Nomor: ###NoSurat###")) ||
			f.ContentType != PDFContentType || f.Name != "015-TI-PSP-X-2026.pdf" {
			t.Errorf("user %d: unduhan salah: %q %+v", u.ID, got, f)
		}
	}
	if pdf.conversions != 1 {
		t.Errorf("konversi = %d kali, want 1 (PDF dipakai ulang)", pdf.conversions)
	}
	if v := store.versions[1]; len(store.versions) != 2 || v.Type != VersionFinal || v.CreatedByRole != RoleApprover || v.CreatedBy.ID != 3 {
		t.Errorf("versi FINAL salah: %+v", store.versions)
	}
	if _, err := svc.OpenDocument(ctx, outsiderUser, testLetterID, 0); !errors.Is(err, ErrLetterNotFound) {
		t.Errorf("bukan penerima: error = %v", err)
	}
}

// makeDocx membuat .docx minimal (tanpa kompresi, supaya isinya bisa dicek apa adanya).
func makeDocx(documentXML string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range map[string]string{
		"[Content_Types].xml": "<Types/>",
		"word/document.xml":   documentXML,
		"word/styles.xml":     "<w:t>###NoSurat###</w:t>", // bukan bagian teks naskah: tidak diubah
	} {
		w, _ := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		w.Write([]byte(content))
	}
	zw.Close()
	return buf.Bytes()
}

func TestFillLetterNumber(t *testing.T) {
	out, err := fillLetterNumber(makeDocx("<w:t>No ###NoSurat###, lagi ###NoSurat###</w:t>"), "015/TI/PSP/X/2026 & <A>")
	if err != nil {
		t.Fatal(err)
	}
	zr, _ := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	files := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		data, _ := io.ReadAll(rc)
		files[f.Name] = string(data)
	}
	if want := "<w:t>No 015/TI/PSP/X/2026 &amp; &lt;A&gt;, lagi 015/TI/PSP/X/2026 &amp; &lt;A&gt;</w:t>"; files["word/document.xml"] != want {
		t.Errorf("document.xml = %q, want %q", files["word/document.xml"], want)
	}
	if files["word/styles.xml"] != "<w:t>###NoSurat###</w:t>" || len(files) != 3 {
		t.Errorf("bagian lain tidak boleh berubah: %v", files)
	}
	if _, err := fillLetterNumber([]byte("bukan zip"), "1"); err == nil {
		t.Error("naskah rusak harus error")
	}
}

func TestFinalPDFRejected(t *testing.T) {
	ctx := context.Background()
	var is *apperr.InvalidStateError

	svc, _, _, _ := sentLetter(true)
	if _, err := svc.OpenDocument(ctx, andiUser, testLetterID, 1); !errors.As(err, &is) || !strings.Contains(is.Message, "PDF resmi") {
		t.Errorf(".docx lama setelah terbit: error = %v", err)
	}

	svc, store, _, _ := sentLetter(false)
	if _, err := svc.OpenDocument(ctx, andiUser, testLetterID, 0); !errors.As(err, &is) || !strings.Contains(is.Message, "belum diunggah") {
		t.Errorf("tanpa naskah: error = %v", err)
	}
	if len(store.versions) != 0 {
		t.Errorf("tidak boleh ada versi FINAL: %+v", store.versions)
	}

	svc, store, pdf, _ := sentLetter(true)
	pdf.down = true
	var ue *apperr.UnavailableError
	if _, err := svc.OpenDocument(ctx, andiUser, testLetterID, 0); !errors.As(err, &ue) {
		t.Errorf("Gotenberg mati: error = %v, want UnavailableError", err)
	}
	if len(store.versions) != 1 {
		t.Errorf("PDF gagal dibuat tidak boleh tersimpan: %+v", store.versions)
	}
}
