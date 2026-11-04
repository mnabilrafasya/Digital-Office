package surat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

var testPDF = []byte("%PDF-1.7 isi")

func TestParseAttachment(t *testing.T) {
	for file, want := range map[string]string{
		"Anggaran.PDF":  PDFContentType,
		"rab.xlsx":      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"paparan.pptx":  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"naskah.docx":   DocxContentType,
		"lama/data.doc": "application/msword",
	} {
		data := testDocx
		switch {
		case strings.HasSuffix(strings.ToLower(file), ".pdf"):
			data = testPDF
		case strings.HasSuffix(file, ".doc"):
			data = append(append([]byte{}, oleSignature...), "isi"...)
		}
		name, ct, err := parseAttachment(file, data)
		if err != nil || ct != want || strings.Contains(name, "/") {
			t.Errorf("%s: name=%q type=%q err=%v", file, name, ct, err)
		}
	}
	for _, tc := range []struct {
		name, file string
		data       []byte
		want       string
	}{
		{"format lain", "virus.exe", testPDF, "PDF, DOC, DOCX, XLSX, atau PPTX"},
		{"tanpa ekstensi", "lampiran", testPDF, "PDF, DOC"},
		{"isi tidak cocok", "data.pdf", testDocx, "tidak sesuai"},
		{"kosong", "data.pdf", nil, "kosong"},
		{"terlalu besar", "data.pdf", append(append([]byte{}, testPDF...), make([]byte, MaxAttachmentBytes)...), "maksimal 25 MB"},
	} {
		_, _, err := parseAttachment(tc.file, tc.data)
		var ve *apperr.ValidationError
		if !errors.As(err, &ve) || !strings.Contains(ve.Message, tc.want) {
			t.Errorf("%s: error = %v, want %q", tc.name, err, tc.want)
		}
	}
}

func TestAttachmentAccess(t *testing.T) {
	add := func(store *fakeLetterStore, u *auth.CurrentUser) error {
		_, _, err := newTestLetterService(store, newFakeRefs()).AddAttachment(context.Background(), u,
			AttachmentInput{LetterID: testLetterID, FileName: "rab.pdf", Data: testPDF})
		return err
	}
	isState := func(err error) bool { var e *apperr.InvalidStateError; return errors.As(err, &e) }
	reviewerStore := letterAt(StatusDraft, 0, 0)
	reviewerStore.participants = map[int64]bool{9: true}

	for _, tc := range []struct {
		name  string
		store *fakeLetterStore
		user  *auth.CurrentUser
		check func(error) bool // nil = berhasil
	}{
		{"drafter saat draf", letterAt(StatusDraft, 0, 0), andiUser, nil},
		{"drafter saat dikembalikan", letterAt(StatusReturnedToDrafter, 15, 0), andiUser, nil},
		{"drafter saat direview", letterAt(StatusInReview, 9, 0), andiUser, isState},
		{"surat terbit", letterAt(StatusSent, 0, 4), andiUser, isState},
		{"reviewer", reviewerStore, fajarUser, func(err error) bool { return errors.Is(err, apperr.ErrForbidden) }},
		{"orang luar", letterAt(StatusDraft, 0, 0), outsiderUser, func(err error) bool { return errors.Is(err, ErrLetterNotFound) }},
		{"surat tidak ada", &fakeLetterStore{}, andiUser, func(err error) bool { return errors.Is(err, ErrLetterNotFound) }},
	} {
		err := add(tc.store, tc.user)
		if tc.check == nil {
			if err != nil || len(tc.store.attachments) != 1 {
				t.Errorf("%s: err=%v lampiran=%d", tc.name, err, len(tc.store.attachments))
			}
			continue
		}
		if !tc.check(err) || len(tc.store.attachments) != 0 {
			t.Errorf("%s: error = %v, lampiran tersimpan %d", tc.name, err, len(tc.store.attachments))
		}
	}

	full := letterAt(StatusDraft, 0, 0)
	for i := range MaxAttachments {
		full.attachments = append(full.attachments, Attachment{ID: fmt.Sprintf("a-%d", i)})
	}
	var ve *apperr.ValidationError
	if err := add(full, andiUser); !errors.As(err, &ve) || !strings.Contains(ve.Message, "maksimal 20") {
		t.Errorf("lebih dari %d lampiran: error = %v", MaxAttachments, err)
	}
}

func TestAttachmentLifecycle(t *testing.T) {
	store := letterAt(StatusDraft, 0, 0)
	svc, audit, inbox := newTestLetterServiceFull(store, newFakeRefs())
	files := svc.repos.Files.(*fakeFiles)
	ctx := context.Background()

	a, rowVer, err := svc.AddAttachment(ctx, andiUser, AttachmentInput{LetterID: testLetterID, FileName: `C:\data\RAB 2027.pdf`, Data: testPDF, ClientIP: "10.0.0.5"})
	if err != nil || a.FileName != "RAB 2027.pdf" || a.ContentType != PDFContentType || a.Size != int64(len(testPDF)) ||
		len(a.Checksum) != 64 || a.UploadedBy.ID != 15 || a.UploadedBy.Name != "Andi" || rowVer == nil {
		t.Fatalf("AddAttachment: %+v err=%v", a, err)
	}
	if string(files.objects[a.ID]) != string(testPDF) {
		t.Error("file lampiran tidak tersimpan di MinIO")
	}
	b, _, _ := svc.AddAttachment(ctx, andiUser, AttachmentInput{LetterID: testLetterID, FileName: "paparan.pptx", Data: testDocx})
	if e := audit.entries[0]; e.Action != "ATTACHMENT_ADD" || e.IPAddress != "10.0.0.5" || e.Snapshot.(attachmentSnapshot).FileName != "RAB 2027.pdf" {
		t.Errorf("audit tambah salah: %+v", e)
	}

	// Hapus satu: daftar tinggal satu, file di MinIO ikut dihapus
	if _, err := svc.RemoveAttachment(ctx, andiUser, testLetterID, strings.ToUpper(b.ID)); err != nil {
		t.Fatalf("RemoveAttachment: %v", err)
	}
	if len(store.attachments) != 1 || store.attachments[0].ID != a.ID || files.objects[b.ID] != nil {
		t.Errorf("hapus salah: daftar=%+v", store.attachments)
	}
	if e := audit.entries[2]; e.Action != "ATTACHMENT_DELETE" || e.Snapshot.(attachmentSnapshot).AttachmentID != b.ID {
		t.Errorf("audit hapus salah: %+v", e)
	}
	notFound := func(err error) bool { return errors.Is(err, errAttachmentNotFound) }
	if _, err := svc.RemoveAttachment(ctx, andiUser, testLetterID, b.ID); !notFound(err) {
		t.Errorf("hapus dua kali: error = %v", err)
	}
	if _, err := svc.RemoveAttachment(ctx, andiUser, testLetterID, "bukan-uuid"); !notFound(err) {
		t.Errorf("id tidak valid: error = %v", err)
	}

	// Setelah terbit: penerima inbox boleh mengunduh, orang luar tidak, lampiran tidak bisa dihapus
	store.lock.Status, store.detail.Status = StatusSent, StatusSent
	inbox.holders = map[int64]bool{50: true}
	recipient := &auth.CurrentUser{ID: 50, Name: "Sri", Company: auth.UserCompany{Code: "PSP"}}
	f, err := svc.OpenAttachment(ctx, recipient, testLetterID, a.ID)
	if err != nil {
		t.Fatalf("OpenAttachment penerima: %v", err)
	}
	body, _ := io.ReadAll(f.Body)
	if string(body) != string(testPDF) || f.Name != "RAB 2027.pdf" || f.ContentType != PDFContentType {
		t.Errorf("unduhan salah: %q %+v", body, f)
	}
	last := audit.entries[len(audit.entries)-1]
	if snap, ok := last.Snapshot.(attachmentSnapshot); last.Action != "DOWNLOAD" || last.ActorID != 50 || !ok || snap.Type != "ATTACHMENT" {
		t.Errorf("audit unduh salah: %+v", last)
	}
	if _, err := svc.OpenAttachment(ctx, outsiderUser, testLetterID, a.ID); !errors.Is(err, ErrLetterNotFound) {
		t.Errorf("orang luar: error = %v", err)
	}
	if _, err := svc.OpenAttachment(ctx, recipient, testLetterID, b.ID); !notFound(err) {
		t.Errorf("lampiran yang sudah dihapus: error = %v", err)
	}
	var st *apperr.InvalidStateError
	if _, err := svc.RemoveAttachment(ctx, andiUser, testLetterID, a.ID); !errors.As(err, &st) {
		t.Errorf("lampiran surat terbit dihapus: error = %v", err)
	}
}
