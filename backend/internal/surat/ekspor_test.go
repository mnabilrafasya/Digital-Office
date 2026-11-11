package surat

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// sentLetter menyiapkan surat terbit (nomor 015/PSP/X/2026, disetujui Agus) yang naskah
// .docx-nya sudah tersimpan sebagai versi 1, beserta service dengan pembuat PDF palsu.
func sentLetter(withDraft bool) (*Service, *fakeLetterStore, *fakePDF, *fakeInbox, *fakeAudit) {
	store := letterAt(StatusSent, 0, 4)
	store.detail.LetterNumber = ptr("015/PSP/X/2026")
	store.detail.Approval = &Approval{ApprovedAt: time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC),
		ApprovedBy: PersonRef{ID: 3, Name: "Agus", PositionName: "SVP"}}
	files, pdf, inbox, audit := &fakeFiles{}, &fakePDF{}, &fakeInbox{}, &fakeAudit{}
	if withDraft {
		files.objects = map[string][]byte{"draf-1": makeDocx(docWith("<w:p><w:r><w:t>Nomor: ###NoSurat###</w:t></w:r></w:p>"))}
		store.versions = []DocumentVersion{{VersionNumber: 1, Type: VersionDraft, FileName: "Naskah.docx", ObjectKey: "draf-1"}}
	}
	refs := newFakeRefs()
	svc := NewService(fakeTx{}, nil, Repos{
		Letters: func(db.DBTX) Store { return store }, Audit: func(db.DBTX) AuditStore { return audit },
		Inbox: func(db.DBTX) InboxStore { return inbox }, Directory: refs, Lookups: refs, Templates: refs,
		Files: files, PDF: pdf,
	}, time.UTC)
	return svc, store, pdf, inbox, audit
}

// minimalPDF membuat PDF satu halaman yang strukturnya valid (xref benar), isinya sebuah garis
// dengan panjang n, supaya dua PDF bisa dibedakan.
func minimalPDF(n int) []byte {
	stream := fmt.Sprintf("0 0 m %d %d l S", n, n)
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return b.Bytes()
}

func TestFinalPDFFromEditor(t *testing.T) {
	svc, store, _, inbox, audit := sentLetter(true)
	inbox.holders = map[int64]bool{50: true}
	recipient := &auth.CurrentUser{ID: 50, Company: auth.UserCompany{Code: "PSP"}}
	ctx := context.Background()

	// Sebelum editor mengunggah PDF: unduhan menunggu, naskah terisi sudah bisa dibuka
	var is *apperr.InvalidStateError
	if _, err := svc.OpenDocument(ctx, andiUser, testLetterID, 0); !errors.Is(err, errFinalPDFNotReady) {
		t.Errorf("PDF belum ada: error = %v", err)
	}
	f, err := svc.FilledDocument(ctx, recipient, testLetterID)
	if err != nil {
		t.Fatalf("FilledDocument: %v", err)
	}
	filled, _ := io.ReadAll(f.Body)
	doc := docxFiles(t, filled)["word/document.xml"]
	// Naskah uji tanpa penanda QR: QR tidak ditambahkan di mana pun (hanya di posisi penanda)
	if !strings.Contains(doc, "Nomor: 015/PSP/X/2026") || strings.Contains(doc, "<w:drawing>") ||
		f.Name != "015-PSP-X-2026.docx" || f.ContentType != DocxContentType {
		t.Errorf("naskah terisi salah: %+v\n%s", f, doc)
	}

	// PDF pertama dipakai; unggahan berikutnya mengembalikan PDF yang sama
	first := minimalPDF(10)
	v, created, err := svc.SaveFinalPDF(ctx, recipient, FinalPDFInput{LetterID: testLetterID, Data: first, ClientIP: "10.0.0.9"})
	if err != nil || !created || v.Type != VersionFinal || v.FileName != "015-PSP-X-2026.pdf" {
		t.Fatalf("SaveFinalPDF: v=%+v created=%v err=%v", v, created, err)
	}
	if v2, created, err := svc.SaveFinalPDF(ctx, andiUser, FinalPDFInput{LetterID: testLetterID, Data: minimalPDF(20)}); err != nil || created || v2.VersionNumber != v.VersionNumber {
		t.Errorf("unggahan kedua: v=%+v created=%v err=%v", v2, created, err)
	}
	if fv := store.versions[1]; len(store.versions) != 2 || fv.CreatedByRole != RoleApprover || fv.CreatedBy.ID != 3 {
		t.Errorf("versi FINAL salah: %+v", store.versions)
	}

	for _, u := range []*auth.CurrentUser{andiUser, recipient, agusUser} {
		f, err := svc.OpenDocument(ctx, u, testLetterID, 0)
		if err != nil {
			t.Fatalf("user %d: %v", u.ID, err)
		}
		got, _ := io.ReadAll(f.Body)
		if !bytes.Equal(got, first) || f.ContentType != PDFContentType || f.Name != "015-PSP-X-2026.pdf" {
			t.Errorf("user %d: unduhan bukan PDF pertama: %+v", u.ID, f)
		}
	}
	var got []string
	for _, e := range audit.entries {
		got = append(got, fmt.Sprintf("%s:%d", e.Action, e.ActorID))
	}
	if strings.Join(got, ",") != "FINAL_PDF:50,DOWNLOAD:15,DOWNLOAD:50,DOWNLOAD:3" {
		t.Errorf("audit = %v", got)
	}
	if snap := audit.entries[0].Snapshot.(finalPDFSnapshot); snap.Generator != "DEVEXPRESS" || snap.Pages != 1 || audit.entries[0].IPAddress != "10.0.0.9" {
		t.Errorf("audit FINAL_PDF salah: %+v", audit.entries[0])
	}

	for name, call := range map[string]func() error{
		"unduh":         func() error { _, err := svc.OpenDocument(ctx, outsiderUser, testLetterID, 0); return err },
		"naskah terisi": func() error { _, err := svc.FilledDocument(ctx, outsiderUser, testLetterID); return err },
		"unggah PDF": func() error {
			_, _, err := svc.SaveFinalPDF(ctx, outsiderUser, FinalPDFInput{LetterID: testLetterID, Data: first})
			return err
		},
	} {
		if err := call(); !errors.Is(err, ErrLetterNotFound) {
			t.Errorf("orang luar %s: error = %v", name, err)
		}
	}
	if _, err := svc.OpenDocument(ctx, andiUser, testLetterID, 1); !errors.As(err, &is) || !strings.Contains(is.Message, "PDF resmi") {
		t.Errorf(".docx lama setelah terbit: error = %v", err)
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

func TestFinalPDFRejected(t *testing.T) {
	ctx := context.Background()
	var is *apperr.InvalidStateError

	svc, store, _, _, _ := sentLetter(false)
	if _, err := svc.FilledDocument(ctx, andiUser, testLetterID); !errors.As(err, &is) || !strings.Contains(is.Message, "belum diunggah") {
		t.Errorf("tanpa naskah: error = %v", err)
	}

	draft := letterAt(StatusInApproval, 3, 4)
	svcDraft := newTestLetterService(draft, newFakeRefs())
	if _, err := svcDraft.FilledDocument(ctx, andiUser, testLetterID); !errors.As(err, &is) {
		t.Errorf("surat belum terbit (naskah terisi): error = %v", err)
	}
	if _, _, err := svcDraft.SaveFinalPDF(ctx, andiUser, FinalPDFInput{LetterID: testLetterID, Data: minimalPDF(1)}); !errors.As(err, &is) {
		t.Errorf("surat belum terbit (unggah PDF): error = %v", err)
	}

	for name, data := range map[string][]byte{
		"kosong":           nil,
		"bukan PDF":        []byte("PK\x03\x04docx"),
		"struktur rusak":   []byte("%PDF-1.4\nbukan isi pdf"),
		"lebih dari 25 MB": append(minimalPDF(1), make([]byte, MaxFinalPDFBytes)...),
	} {
		var ve *apperr.ValidationError
		if _, _, err := svc.SaveFinalPDF(ctx, andiUser, FinalPDFInput{LetterID: testLetterID, Data: data}); !errors.As(err, &ve) {
			t.Errorf("%s: error = %v, want ValidationError", name, err)
		}
	}
	if len(store.versions) != 0 {
		t.Errorf("PDF ditolak tidak boleh tersimpan: %+v", store.versions)
	}
}
