package surat

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

var testDocx = []byte("PK\x03\x04isi-docx")

func TestParseDocument(t *testing.T) {
	if name, err := parseDocument(`C:\Users\andi\Naskah Rapat.DOCX`, testDocx); err != nil || name != "Naskah Rapat.DOCX" {
		t.Errorf("valid: name=%q err=%v", name, err)
	}
	tests := []struct {
		name, fileName string
		data           []byte
		want           string
	}{
		{"bukan docx", "naskah.pdf", testDocx, "wajib berupa file .docx"},
		{"tanpa nama", "", testDocx, "wajib berupa file .docx"},
		{"kosong", "a.docx", nil, "kosong"},
		{"bukan zip", "a.docx", []byte("bukan zip"), "bukan dokumen .docx"},
		{"terlalu besar", "a.docx", append(append([]byte{}, testDocx...), make([]byte, MaxDocumentBytes)...), "maksimal 10 MB"},
		{"nama terlalu panjang", strings.Repeat("a", 251) + ".docx", testDocx, "Nama file maksimal 255"},
	}
	for _, tt := range tests {
		_, err := parseDocument(tt.fileName, tt.data)
		var ve *apperr.ValidationError
		if !errors.As(err, &ve) || !strings.Contains(ve.Message, tt.want) {
			t.Errorf("%s: error = %v, want %q", tt.name, err, tt.want)
		}
	}
}

func TestSaveDocumentAccess(t *testing.T) {
	save := func(store *fakeLetterStore, u *auth.CurrentUser) error {
		_, _, err := newTestLetterService(store, newFakeRefs()).SaveDocument(context.Background(), u,
			DocumentInput{LetterID: testLetterID, FileName: "Naskah.docx", Data: testDocx})
		return err
	}
	isState := func(err error) bool { var e *apperr.InvalidStateError; return errors.As(err, &e) }

	for _, tc := range []struct {
		name     string
		store    *fakeLetterStore
		user     *auth.CurrentUser
		wantRole string // "" = ditolak
		check    func(error) bool
	}{
		{"drafter saat draf", letterAt(StatusDraft, 0, 0), andiUser, RoleDrafter, nil},
		{"drafter saat dikembalikan", letterAt(StatusReturnedToDrafter, 15, 0), andiUser, RoleDrafter, nil},
		{"reviewer yang memegang", letterAt(StatusInReview, 9, 0), fajarUser, RoleReviewer, nil},
		{"reviewer dikembalikan approver", letterAt(StatusReturnedToReviewer, 4, 4), dewiUser, RoleReviewer, nil},
		{"drafter saat direview", letterAt(StatusInReview, 9, 0), andiUser, "", isState},
		{"reviewer bukan giliran", letterAt(StatusInReview, 9, 0), dewiUser, "", isState},
		{"approver hanya membaca", letterAt(StatusInApproval, 3, 4), agusUser, "", isState},
		{"permission review dicabut", letterAt(StatusInReview, 9, 0), &auth.CurrentUser{ID: 9},
			"", func(err error) bool { return errors.Is(err, apperr.ErrForbidden) }},
		{"orang luar", letterAt(StatusDraft, 0, 0), outsiderUser, "", func(err error) bool { return errors.Is(err, ErrLetterNotFound) }},
	} {
		err := save(tc.store, tc.user)
		if tc.wantRole != "" {
			if err != nil || len(tc.store.versions) != 1 || tc.store.versions[0].CreatedByRole != tc.wantRole {
				t.Errorf("%s: err=%v versions=%+v", tc.name, err, tc.store.versions)
			}
			continue
		}
		if !tc.check(err) || len(tc.store.versions) != 0 {
			t.Errorf("%s: error = %v, versi tersimpan %d", tc.name, err, len(tc.store.versions))
		}
	}
}

func TestSaveAndDownloadDocument(t *testing.T) {
	store := letterAt(StatusDraft, 0, 0)
	store.participants = map[int64]bool{9: true}
	svc, audit, _ := newTestLetterServiceFull(store, newFakeRefs())
	ctx := context.Background()

	var nf *apperr.NotFoundError
	if _, err := svc.OpenDocument(ctx, andiUser, testLetterID, 0); !errors.As(err, &nf) || !strings.Contains(nf.Message, "belum diunggah") {
		t.Errorf("belum ada naskah: error = %v", err)
	}

	v, rowVer, err := svc.SaveDocument(ctx, andiUser, DocumentInput{LetterID: testLetterID, FileName: "Naskah.docx", Data: testDocx, ClientIP: "10.0.0.5"})
	if err != nil || v.VersionNumber != 1 || v.Type != VersionDraft || len(v.Checksum) != 64 || rowVer == nil {
		t.Fatalf("SaveDocument: v=%+v err=%v", v, err)
	}
	if e := audit.entries[0]; e.Action != "SAVE" || e.IPAddress != "10.0.0.5" {
		t.Errorf("audit salah: %+v", e)
	}

	f, err := svc.OpenDocument(ctx, fajarUser, testLetterID, 0) // reviewer boleh mengunduh
	if err != nil {
		t.Fatalf("OpenDocument: %v", err)
	}
	got, _ := io.ReadAll(f.Body)
	if string(got) != string(testDocx) || f.Name != "Naskah.docx" || f.ContentType != DocxContentType {
		t.Errorf("unduhan salah: %q %+v", got, f)
	}
	if _, err := svc.OpenDocument(ctx, andiUser, testLetterID, 2); !errors.As(err, &nf) || !strings.Contains(nf.Message, "Versi naskah") {
		t.Errorf("versi tidak ada: error = %v", err)
	}
	if _, err := svc.OpenDocument(ctx, outsiderUser, testLetterID, 0); !errors.Is(err, ErrLetterNotFound) {
		t.Errorf("orang luar: error = %v", err)
	}

}
