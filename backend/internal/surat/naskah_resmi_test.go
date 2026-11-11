package surat

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

func userID(id int64) *auth.CurrentUser {
	return &auth.CurrentUser{ID: id, Name: "Penerima", Company: auth.UserCompany{Code: "PSP"}}
}

// officialLetter menyiapkan surat terbit yang naskahnya memuat penanda sistem dan penanda isi.
// Penerima: 50 (Kepada), 60 (disposisi), 70 (tembusan); 15 Andi drafter.
func officialLetter() (*Service, *fakeLetterStore, *fakeFiles, *fakeAudit) {
	svc, store, _, inbox, audit := sentLetter(true)
	files := svc.repos.Files.(*fakeFiles)
	files.objects["draf-1"] = makeDocx(docWith(`<w:p><w:r><w:t>Kepada Yth. ###*Kepada*###</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>Nomor: ###*NoSurat*###</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>###*QRCode*###</w:t></w:r></w:p>`))
	inbox.holders = map[int64]bool{50: true, 60: true, 70: true}
	return svc, store, files, audit
}

// Satu surat = satu naskah resmi dan satu PDF resmi untuk semua orang.
func TestOfficialDocumentSameForEveryone(t *testing.T) {
	svc, store, files, audit := officialLetter()
	ctx := context.Background()
	everyone := []*auth.CurrentUser{userID(50), userID(60), userID(70), andiUser}

	for _, u := range everyone {
		f, err := svc.FilledDocument(ctx, u, testLetterID)
		if err != nil {
			t.Fatalf("user %d: %v", u.ID, err)
		}
		data, _ := io.ReadAll(f.Body)
		got := paragraphTexts(t, docxFiles(t, data)["word/document.xml"])
		if got[0] != "Kepada Yth. ###*Kepada*###" || got[1] != "Nomor: 015/PSP/X/2026" {
			t.Errorf("user %d: naskah terisi = %q (Kepada tidak diisi otomatis, nomor diisi)", u.ID, got)
		}
		st, err := svc.OfficialStatus(ctx, u, testLetterID)
		if err != nil || st.PDFReady || st.CopyFor != nil {
			t.Errorf("user %d: status = %+v, err = %v", u.ID, st, err)
		}
	}
	if _, err := svc.OfficialStatus(ctx, outsiderUser, testLetterID); !errors.Is(err, ErrLetterNotFound) {
		t.Errorf("orang luar: error = %v", err)
	}

	// Penerima disposisi membuka lebih dulu: PDF yang dibuat adalah PDF resmi surat (versi FINAL)
	pdf := minimalPDF(5)
	if _, created, err := svc.SaveFinalPDF(ctx, userID(60), FinalPDFInput{LetterID: testLetterID, Data: pdf}); err != nil || !created {
		t.Fatalf("SaveFinalPDF: created=%v err=%v", created, err)
	}
	if len(store.versions) != 2 || store.versions[1].Type != VersionFinal || files.objects[filledObjectKey(testLetterID)] == nil {
		t.Errorf("PDF resmi harus menjadi versi FINAL + naskah terisi tersimpan: %+v", store.versions)
	}
	if _, created, err := svc.SaveFinalPDF(ctx, andiUser, FinalPDFInput{LetterID: testLetterID, Data: minimalPDF(6)}); err != nil || created {
		t.Errorf("PDF sudah ada: created=%v err=%v", created, err)
	}
	for _, u := range everyone {
		if st, _ := svc.OfficialStatus(ctx, u, testLetterID); !st.PDFReady {
			t.Errorf("user %d: PDF harus siap", u.ID)
		}
		f, err := svc.OpenDocument(ctx, u, testLetterID, 0)
		if err != nil {
			t.Fatalf("user %d: %v", u.ID, err)
		}
		if got, _ := io.ReadAll(f.Body); !bytes.Equal(got, pdf) {
			t.Errorf("user %d: unduhan bukan PDF resmi yang sama", u.ID)
		}
	}
	last := audit.entries[len(audit.entries)-1]
	if snap, ok := last.Snapshot.(downloadSnapshot); !ok || last.Action != "DOWNLOAD" || snap.Type != VersionFinal {
		t.Errorf("audit unduhan salah: %+v", last)
	}
}

// Penanda QR wajib ada di naskah: diperiksa saat diajukan dan saat approver menyetujui.
func TestQRPlaceholderRequired(t *testing.T) {
	ctx := context.Background()
	withQR := makeDocx(docWith(`<w:p><w:r><w:t>TTD ###*QRCode*###</w:t></w:r></w:p>`))
	noQR := makeDocx(docWith(`<w:p><w:r><w:t>TTD tanpa QR</w:t></w:r></w:p>`))
	letterWith := func(status string, assignee, forwardedBy int64, docx []byte) (*Service, *fakeLetterStore) {
		store := letterAt(status, assignee, forwardedBy)
		refs := newFakeRefs()
		refs.unitHeads = map[int64]int64{7: 50, 9: 51} // penerima surat uji punya pimpinan (syarat terbit)
		refs.aliasMembers = map[int64][]int64{60: {52}}
		svc := newTestLetterService(store, refs)
		if docx != nil {
			svc.repos.Files.(*fakeFiles).objects = map[string][]byte{"naskah": docx}
			store.versions = []DocumentVersion{{VersionNumber: 1, Type: VersionDraft, FileName: "Naskah.docx", ObjectKey: "naskah"}}
		}
		return svc, store
	}
	in := TransitionInput{LetterID: testLetterID}

	var invalid *apperr.ValidationError
	svc, store := letterWith(StatusDraft, 0, 0, noQR)
	if _, err := svc.Submit(ctx, andiUser, in); !errors.As(err, &invalid) || store.applied != nil {
		t.Errorf("ajukan tanpa penanda QR: err=%v applied=%v", err, store.applied)
	}
	svc, store = letterWith(StatusReturnedToDrafter, 15, 0, noQR)
	if _, err := svc.Submit(ctx, andiUser, in); !errors.As(err, &invalid) || store.applied != nil {
		t.Errorf("ajukan ulang tanpa penanda QR: err=%v", err)
	}
	svc, store = letterWith(StatusDraft, 0, 0, withQR)
	if _, err := svc.Submit(ctx, andiUser, in); err != nil || store.applied == nil {
		t.Errorf("ajukan dengan penanda QR: %v", err)
	}
	svc, store = letterWith(StatusDraft, 0, 0, nil)
	if _, err := svc.Submit(ctx, andiUser, in); err != nil || store.applied == nil {
		t.Errorf("ajukan tanpa naskah tetap boleh: %v", err)
	}

	// Approver menyetujui: naskah diperiksa ulang sebelum nomor terbit
	var state *apperr.InvalidStateError
	svc, store = letterWith(StatusInApproval, 3, 4, noQR)
	if _, err := svc.Approve(ctx, agusUser, in); !errors.As(err, &state) || store.published != nil || store.applied != nil {
		t.Errorf("approve tanpa penanda QR: err=%v published=%v", err, store.published)
	}
	svc, store = letterWith(StatusInApproval, 3, 4, withQR)
	if _, err := svc.Approve(ctx, agusUser, in); err != nil || store.published == nil {
		t.Errorf("approve dengan penanda QR: %v", err)
	}
	// Reviewer menyetujui (bukan terbit): tidak diperiksa
	svc, store = letterWith(StatusInReview, 9, 0, noQR)
	if _, err := svc.Approve(ctx, fajarUser, in); err != nil || store.applied == nil {
		t.Errorf("reviewer setujui: %v", err)
	}
}
