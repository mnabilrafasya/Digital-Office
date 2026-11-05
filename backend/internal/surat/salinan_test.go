package surat

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"office.backend/internal/auth"
	"office.backend/internal/organisasi"
)

// copyLetter menyiapkan surat terbit ke "Seluruh VP" dengan penerima:
//
//	50 Joko  anggota alias (TO)         -> salinan sendiri, "VP Akuntansi"
//	60 Maya  disposisi dari Joko        -> salinan Joko
//	70 Indah tembusan (CC)              -> salinan induk
//	15 Andi  drafter (bukan penerima)   -> salinan induk
func copyLetter(t *testing.T) (*Service, *fakeLetterStore, *fakeInbox, *fakeFiles, *fakeAudit) {
	t.Helper()
	svc, store, _, inbox, audit := sentLetter(true)
	files := svc.repos.Files.(*fakeFiles)
	files.objects["draf-1"] = makeDocx(docWith(`<w:p><w:r><w:t>Kepada Yth. ###*Kepada*###</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>Nomor: ###*NoSurat*###</w:t></w:r></w:p>`))
	store.detail.Recipients = LetterRecipients{To: []LetterRecipient{{Alias: &organisasi.Alias{Label: "Seluruh VP"}}}}
	inbox.holders = map[int64]bool{50: true, 60: true, 70: true}
	inbox.copies = map[int64]fakeCopy{
		50: {owner: 50, typ: RecipientTo, post: "VP Akuntansi"},
		60: {owner: 50, typ: RecipientTo, post: "VP Akuntansi"},
		70: {owner: 70, typ: RecipientCC, post: "VP Anggaran"},
	}
	return svc, store, inbox, files, audit
}

func userID(id int64) *auth.CurrentUser {
	return &auth.CurrentUser{ID: id, Name: "Penerima", Company: auth.UserCompany{Code: "PSP"}}
}

// kepadaOf membaca teks paragraf "Kepada" dari naskah terisi yang dilihat user.
func kepadaOf(t *testing.T, svc *Service, u *auth.CurrentUser) string {
	t.Helper()
	f, err := svc.FilledDocument(context.Background(), u, testLetterID)
	if err != nil {
		t.Fatalf("FilledDocument user %d: %v", u.ID, err)
	}
	data, _ := io.ReadAll(f.Body)
	return paragraphTexts(t, docxFiles(t, data)["word/document.xml"])[0]
}

func TestOfficialCopyPerRecipient(t *testing.T) {
	svc, _, _, _, _ := copyLetter(t)
	ctx := context.Background()

	for _, tc := range []struct {
		user    *auth.CurrentUser
		kepada  string
		copyFor string // "" = salinan induk
	}{
		{userID(50), "Kepada Yth. VP Akuntansi", "VP Akuntansi"},
		{userID(60), "Kepada Yth. VP Akuntansi", "VP Akuntansi"}, // disposisi: salinan Joko
		{userID(70), "Kepada Yth. Para VP", ""},                  // tembusan: salinan induk
		{andiUser, "Kepada Yth. Para VP", ""},                    // drafter: salinan induk
	} {
		if got := kepadaOf(t, svc, tc.user); got != tc.kepada {
			t.Errorf("user %d: Kepada = %q, want %q", tc.user.ID, got, tc.kepada)
		}
		st, err := svc.OfficialStatus(ctx, tc.user, testLetterID)
		if err != nil || st.PDFReady {
			t.Fatalf("user %d: status=%+v err=%v", tc.user.ID, st, err)
		}
		if got := ""; st.CopyFor != nil {
			got = *st.CopyFor
			if got != tc.copyFor {
				t.Errorf("user %d: copyFor = %q, want %q", tc.user.ID, got, tc.copyFor)
			}
		} else if tc.copyFor != "" {
			t.Errorf("user %d: copyFor kosong, want %q", tc.user.ID, tc.copyFor)
		}
	}
	if _, err := svc.OfficialStatus(ctx, outsiderUser, testLetterID); !errors.Is(err, ErrLetterNotFound) {
		t.Errorf("orang luar: error = %v", err)
	}
}

func TestOfficialCopyPDF(t *testing.T) {
	svc, store, inbox, files, audit := copyLetter(t)
	ctx := context.Background()
	joko, maya := userID(50), userID(60)

	// Maya (disposisi) membuka lebih dulu: PDF yang dibuat adalah salinan Joko
	copyPDF := minimalPDF(5)
	if _, created, err := svc.SaveFinalPDF(ctx, maya, FinalPDFInput{LetterID: testLetterID, Data: copyPDF}); err != nil || !created {
		t.Fatalf("salinan Joko: created=%v err=%v", created, err)
	}
	if files.objects["salinan/"+testLetterID+"/50.pdf"] == nil || files.objects["salinan/"+testLetterID+"/50.docx"] == nil {
		t.Errorf("salinan tidak tersimpan dengan nama tetap: %v", len(files.objects))
	}
	if len(store.versions) != 1 {
		t.Errorf("salinan penerima tidak boleh menambah baris versi: %+v", store.versions)
	}
	if snap := audit.entries[0].Snapshot.(finalPDFSnapshot); audit.entries[0].ActorID != 60 || snap.CopyFor != 50 || snap.CopyPosition != "VP Akuntansi" {
		t.Errorf("audit salinan salah: %+v", audit.entries[0])
	}
	if _, created, err := svc.SaveFinalPDF(ctx, joko, FinalPDFInput{LetterID: testLetterID, Data: minimalPDF(6)}); err != nil || created {
		t.Errorf("salinan sudah ada: created=%v err=%v", created, err)
	}
	if st, _ := svc.OfficialStatus(ctx, joko, testLetterID); !st.PDFReady {
		t.Error("status salinan Joko harus siap")
	}

	// Joko dan Maya mengunduh PDF yang sama (salinan Joko); salinan induk belum ada
	for _, u := range []*auth.CurrentUser{joko, maya} {
		f, err := svc.OpenDocument(ctx, u, testLetterID, 0)
		if err != nil {
			t.Fatalf("user %d: %v", u.ID, err)
		}
		got, _ := io.ReadAll(f.Body)
		if !bytes.Equal(got, copyPDF) || f.Name != "015-PSP-X-2026.pdf" {
			t.Errorf("user %d: unduhan bukan salinan Joko", u.ID)
		}
	}
	last := audit.entries[len(audit.entries)-1]
	if snap := last.Snapshot.(downloadSnapshot); last.Action != "DOWNLOAD" || snap.CopyFor != 50 {
		t.Errorf("audit unduhan salah: %+v", last)
	}
	if _, err := svc.OpenDocument(ctx, andiUser, testLetterID, 0); !errors.Is(err, errFinalPDFNotReady) {
		t.Errorf("salinan induk belum dibuat: error = %v", err)
	}

	// Salinan induk dibuat terpisah (baris versi FINAL), tidak tertukar dengan salinan Joko
	masterPDF := minimalPDF(9)
	if _, created, err := svc.SaveFinalPDF(ctx, andiUser, FinalPDFInput{LetterID: testLetterID, Data: masterPDF}); err != nil || !created {
		t.Fatalf("salinan induk: created=%v err=%v", created, err)
	}
	f, _ := svc.OpenDocument(ctx, userID(70), testLetterID, 0)
	if got, _ := io.ReadAll(f.Body); !bytes.Equal(got, masterPDF) || len(store.versions) != 2 || files.objects["salinan/"+testLetterID+"/induk.docx"] == nil {
		t.Error("tembusan harus mengunduh salinan induk")
	}

	// Jabatan Joko berubah setelah salinan dibuat: tampilan tetap sama dengan PDF-nya
	inbox.copies[50] = fakeCopy{owner: 50, typ: RecipientTo, post: "VP Akuntansi & Pajak"}
	if got := kepadaOf(t, svc, joko); !strings.HasSuffix(got, "VP Akuntansi") {
		t.Errorf("salinan berubah setelah dibuat: %q", got)
	}
}
