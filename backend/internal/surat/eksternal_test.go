package surat

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"office.backend/internal/auth"
	"office.backend/internal/masterdata"
	"office.backend/internal/platform/apperr"
)

// externalLetter mengubah surat uji menjadi surat eksternal keluar ke satu instansi.
func externalLetter(store *fakeLetterStore) *fakeLetterStore {
	store.lock.TypeCode = masterdata.TypeExternalOut
	store.detail.Direction = masterdata.DirectionOutgoing
	store.detail.External = &ExternalInfo{Recipient: &ExternalParty{Name: "PT Pupuk Kaltim"}}
	store.draft.TypeCode = masterdata.TypeExternalOut
	store.draft.Recipients = nil
	store.draft.ExternalRecipient = &ExternalParty{Name: "PT Pupuk Kaltim"}
	return store
}

func TestExternalDraftRules(t *testing.T) {
	refs := newFakeRefs()
	refs.codes = map[int64]string{12: masterdata.TypeExternalOut, 13: masterdata.TypeExternalIn}
	refs.lookups[13] = "LETTER_TYPE"
	svc := newTestLetterService(letterAt(StatusDraft, 0), refs)
	ctx := context.Background()
	external := func(mutate func(*LetterInput)) error {
		in := completeInput()
		in.Recipients.To, in.Recipients.CC = nil, nil
		in.ExternalRecipient = &ExternalPartyInput{Name: ptr(" PT Pupuk Kaltim "), Address: ptr("Bontang"), Attention: ptr("Direktur SDM")}
		mutate(&in)
		d, err := parseDraft(in)
		if err != nil {
			return err
		}
		if err := svc.checkRefs(ctx, "PSP", d); err != nil {
			return err
		}
		return checkComplete(d)
	}
	if err := external(func(*LetterInput) {}); err != nil {
		t.Errorf("surat eksternal lengkap: %v", err)
	}
	var invalid *apperr.ValidationError
	cases := map[string]func(*LetterInput){
		"eksternal dengan penerima internal": func(in *LetterInput) { in.Recipients.To = []RecipientInput{{UnitID: ptr("7")}} },
		"eksternal tanpa tujuan":             func(in *LetterInput) { in.ExternalRecipient = nil },
		"tujuan tanpa nama":                  func(in *LetterInput) { in.ExternalRecipient.Name = ptr(" ") },
		"surat internal dengan tujuan luar": func(in *LetterInput) {
			in.LetterTypeID = nil
			in.Recipients.To = []RecipientInput{{UnitID: ptr("7")}}
		},
		"jenis surat masuk eksternal dipakai sebagai draf": func(in *LetterInput) { in.LetterTypeID = ptr("13") },
	}
	for name, mutate := range cases {
		if err := external(mutate); !errors.As(err, &invalid) {
			t.Errorf("%s: error = %v, want validasi", name, err)
		}
	}
	// Approver bertingkat: maksimal 10, tidak boleh dobel
	in := completeInput()
	in.ApproverIDs = []string{"3", "3"}
	if _, err := parseDraft(in); err == nil || !strings.Contains(err.Error(), "Approver yang sama") {
		t.Errorf("approver dobel: %v", err)
	}
}

// Surat eksternal: approver terakhir menyetujui → APPROVED (nomor terbit, tanpa inbox penerima,
// penanda QR tidak wajib).
func TestExternalSignAndUploadSigned(t *testing.T) {
	store := externalLetter(letterAt(StatusInApproval, 3))
	store.draft.Recipients = nil
	svc, audit, inbox := newTestLetterServiceFull(store, newFakeRefs())
	files := svc.repos.Files.(*fakeFiles)
	files.objects = map[string][]byte{"naskah": makeDocx(docWith(`<w:p><w:r><w:t>Tanpa penanda QR</w:t></w:r></w:p>`))}
	store.versions = []DocumentVersion{{VersionNumber: 1, Type: VersionDraft, ObjectKey: "naskah"}}
	ctx := context.Background()

	if _, err := svc.Approve(ctx, agusUser, TransitionInput{LetterID: testLetterID}); err != nil {
		t.Fatalf("approve eksternal tanpa penanda QR: %v", err)
	}
	if d := store.applied; d.To != StatusApproved || d.Action != Sign || d.AssigneeID != 3 || store.published == nil ||
		len(inbox.to) != 0 {
		t.Errorf("approve eksternal: %+v published=%v inbox=%v", d, store.published, inbox.to)
	}

	// Unggah scan: hanya approver terakhir, status APPROVED
	store.lock.Status, store.lock.AssigneeID = StatusApproved, 3
	store.detail.Status, store.detail.LetterNumber = StatusApproved, ptr("001/PSP/XI/2026")
	if _, err := svc.UploadSigned(ctx, fajarUser, FinalPDFInput{LetterID: testLetterID, Data: minimalPDF(5)}); !errors.Is(err, errNotSignable) {
		t.Errorf("reviewer mengunggah scan: %v", err)
	}
	if _, err := svc.UploadSigned(ctx, agusUser, FinalPDFInput{LetterID: testLetterID, Data: []byte("bukan pdf")}); err == nil {
		t.Error("scan bukan PDF diterima")
	}
	scan := minimalPDF(7)
	if _, err := svc.UploadSigned(ctx, agusUser, FinalPDFInput{LetterID: testLetterID, Data: scan}); err != nil {
		t.Fatalf("UploadSigned: %v", err)
	}
	if d := store.applied; d.Action != UploadSigned || d.From != StatusApproved || d.To != StatusSent {
		t.Errorf("transisi unggah scan salah: %+v", d)
	}
	final := store.versions[len(store.versions)-1]
	if final.Type != VersionFinal || files.objects[final.ObjectKey] == nil || string(files.objects[final.ObjectKey]) != string(scan) {
		t.Error("scan harus disimpan apa adanya (tanpa watermark tambahan) sebagai versi FINAL")
	}
	if !slices.Equal(inbox.to, []int64{15, 9, 4, 3}) {
		t.Errorf("arsip ke inbox drafter, reviewer, approver: %v", inbox.to)
	}
	if last := audit.entries[len(audit.entries)-1]; last.Action != "UPLOAD_SIGNED" {
		t.Errorf("audit unggah scan: %+v", last)
	}

	// Surat internal tidak bisa diunggah scan
	internal := letterAt(StatusSent, 0)
	internal.detail.LetterNumber = ptr("002/PSP/XI/2026")
	if _, err := newTestLetterService(internal, newFakeRefs()).UploadSigned(ctx, agusUser,
		FinalPDFInput{LetterID: testLetterID, Data: scan}); !errors.Is(err, errNotSignable) {
		t.Errorf("surat internal: %v", err)
	}
}

// Naskah resmi surat eksternal: penanda QR dikosongkan dan QR tidak tersedia.
func TestExternalOfficialDocumentWithoutQR(t *testing.T) {
	svc, store, _, _, _ := sentLetter(true)
	externalLetter(store)
	store.detail.Status = StatusApproved
	svc.repos.Files.(*fakeFiles).objects["draf-1"] = makeDocx(docWith(
		`<w:p><w:r><w:t>Nomor: ###*NoSurat*### TTD ###*QRCode*###</w:t></w:r></w:p>`))
	ctx := context.Background()
	f, err := svc.FilledDocument(ctx, andiUser, testLetterID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f.Body)
	doc := docxFiles(t, data)["word/document.xml"]
	if strings.Contains(doc, "<w:drawing>") || strings.Contains(doc, "QRCode") || !strings.Contains(doc, "015/PSP/X/2026") {
		t.Errorf("naskah eksternal: %s", doc)
	}
	var state *apperr.InvalidStateError
	if _, err := svc.ApprovalQR(ctx, andiUser, testLetterID); !errors.As(err, &state) {
		t.Errorf("QR surat eksternal: %v", err)
	}
	// PDF cetak (APPROVED) tersimpan; setelah SENT, PDF resmi dari editor ditolak (scan yang resmi)
	if _, created, err := svc.SaveFinalPDF(ctx, andiUser, FinalPDFInput{LetterID: testLetterID, Data: minimalPDF(3)}); err != nil || !created {
		t.Fatalf("PDF cetak: created=%v err=%v", created, err)
	}
	store.detail.Status = StatusSent
	if _, _, err := svc.SaveFinalPDF(ctx, andiUser, FinalPDFInput{LetterID: testLetterID, Data: minimalPDF(4)}); !errors.As(err, &state) {
		t.Errorf("PDF editor setelah scan: %v", err)
	}
}

func TestCreateIncoming(t *testing.T) {
	refs := newFakeRefs()
	refs.codes = map[int64]string{70: masterdata.TypeExternalIn}
	refs.aliasMembers = map[int64][]int64{60: {52, 15}} // 15 = pengunggah sendiri, tidak dikirimi
	store := letterAt(StatusReceived, 0)
	svc, audit, inbox := newTestLetterServiceFull(store, refs)
	uploader := &auth.CurrentUser{ID: 15, Name: "Andi", Unit: auth.UserUnit{ID: 8},
		Position: auth.UserPosition{Name: "Staf"}, Company: auth.UserCompany{Code: "PSP"}}
	ctx := context.Background()
	valid := func() IncomingInput {
		return IncomingInput{Title: " Undangan Kerja Sama ", Sender: "PT Pupuk Kaltim", ReferenceNumber: "123/PKT/XI/2026",
			LetterDate: "2026-11-02", UserIDs: []string{"9", ""}, AliasLevelIDs: []string{"60"}, Note: ptr("Mohon dipelajari"),
			Data: minimalPDF(5)}
	}
	if _, err := svc.CreateIncoming(ctx, uploader, valid()); err != nil {
		t.Fatalf("CreateIncoming: %v", err)
	}
	n := store.incoming
	if n == nil || n.Title != "Undangan Kerja Sama" || n.LetterTypeID != 70 || n.UploaderID != 15 ||
		n.External.Sender != "PT Pupuk Kaltim" || n.External.ReferenceNumber != "123/PKT/XI/2026" || n.External.LetterDate != "2026-11-02" {
		t.Errorf("surat masuk tersimpan salah: %+v", n)
	}
	if in := inbox.incoming; in == nil || in.from != 15 || !slices.Equal(in.recipients, []int64{9, 52}) || *in.note != "Mohon dipelajari" {
		t.Errorf("disposisi surat masuk salah: %+v", inbox.incoming)
	}
	if v := store.versions[len(store.versions)-1]; v.Type != VersionFinal || !strings.HasPrefix(v.FileName, "Surat Masuk 123-PKT") {
		t.Errorf("scan surat masuk: %+v", v)
	}
	if last := audit.entries[len(audit.entries)-1]; last.Action != "INCOMING_RECEIVE" {
		t.Errorf("audit surat masuk: %+v", last)
	}

	var invalid *apperr.ValidationError
	bad := map[string]func(*IncomingInput){
		"tanpa perihal":       func(in *IncomingInput) { in.Title = " " },
		"tanpa pengirim":      func(in *IncomingInput) { in.Sender = "" },
		"tanpa nomor asal":    func(in *IncomingInput) { in.ReferenceNumber = "" },
		"tanggal salah":       func(in *IncomingInput) { in.LetterDate = "02-11-2026" },
		"tanpa tujuan":        func(in *IncomingInput) { in.UserIDs, in.AliasLevelIDs = nil, nil },
		"hanya diri sendiri":  func(in *IncomingInput) { in.UserIDs, in.AliasLevelIDs = []string{"15"}, nil },
		"alias tanpa anggota": func(in *IncomingInput) { in.AliasLevelIDs = []string{"61"}; refs.lookups[61] = "POSITION_LEVEL" },
		"alias bukan jabatan": func(in *IncomingInput) { in.AliasLevelIDs = []string{"40"} },
		"orang tidak aktif":   func(in *IncomingInput) { in.UserIDs = []string{"99"}; refs.inactiveUsers = map[int64]bool{99: true} },
		"file bukan PDF":      func(in *IncomingInput) { in.Data = []byte("bukan pdf") },
	}
	for name, mutate := range bad {
		in := valid()
		mutate(&in)
		if _, err := svc.CreateIncoming(ctx, uploader, in); !errors.As(err, &invalid) {
			t.Errorf("%s: error = %v, want validasi", name, err)
		}
	}
}

// Penanda QR dikosongkan bila gambar QR tidak diberikan.
func TestFillPlaceholdersWithoutQR(t *testing.T) {
	out, err := fillPlaceholders(makeDocx(docWith(`<w:p><w:r><w:t>TTD: ###*QRCode*### selesai</w:t></w:r></w:p>`)),
		map[string]string{phNoSurat: "1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	files := docxFiles(t, out)
	if got := paragraphTexts(t, files["word/document.xml"]); len(got) != 1 || got[0] != "TTD:  selesai" ||
		files["word/media/qr-persetujuan.png"] != "" {
		t.Errorf("penanda QR harus dikosongkan tanpa gambar: %q", got)
	}
}
