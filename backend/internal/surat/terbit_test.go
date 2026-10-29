package surat

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

func TestFormatLetterNumber(t *testing.T) {
	oct := time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		format string
		seq    int
		at     time.Time
		want   string
	}{
		{"{SEQ}/{UNIT}/{COMP}/{MM_ROMAN}/{YYYY}", 15, oct, "015/TI/PSP/X/2026"},
		{"", 1, time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC), "001/TI/PSP/I/2027"}, // format default
		{"{COMP}-{YYYY}-{SEQ}", 1234, oct, "PSP-2026-1234"},
	}
	for _, tt := range tests {
		if got := FormatLetterNumber(tt.format, tt.seq, "TI", "PSP", tt.at); got != tt.want {
			t.Errorf("FormatLetterNumber(%q, %d) = %q, want %q", tt.format, tt.seq, got, tt.want)
		}
	}
}

func TestApproverPublishesLetter(t *testing.T) {
	store := letterAt(StatusInApproval, 3, 4)
	refs := newFakeRefs()
	// Kepada: unit 7 dan alias 60; Tembusan: unit 9 (lihat completeInput)
	refs.unitHeads = map[int64]int64{7: 50, 9: 51}
	refs.aliasMembers = map[int64][]int64{60: {52, 50}} // 50 juga pimpinan unit 7: cukup sekali
	svc, audit, inbox := newTestLetterServiceFull(store, refs)

	if _, err := svc.Approve(context.Background(), agusUser, TransitionInput{LetterID: testLetterID}); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if d := store.applied; d == nil || d.Action != Sign || d.To != StatusSent || d.AssigneeID != 0 {
		t.Errorf("transisi salah: %+v", store.applied)
	}
	p := store.published
	if p == nil || p.SignerID != 3 || p.Seq != 15 || p.Year != p.ApprovedAt.Year() ||
		!strings.HasPrefix(p.Number, "015/TI/PSP/") || p.UnitHeads[7] != 50 || p.UnitHeads[9] != 51 {
		t.Errorf("publikasi salah: %+v", p)
	}
	if !slices.Equal(inbox.to, []int64{50, 52}) || !slices.Equal(inbox.cc, []int64{51}) {
		t.Errorf("inbox to=%v cc=%v, want to=[50 52] cc=[51]", inbox.to, inbox.cc)
	}
	e := audit.entries[0]
	snap, _ := e.Snapshot.(transitionSnapshot)
	if e.Action != "SIGN" || snap.LetterNumber == nil || *snap.LetterNumber != p.Number {
		t.Errorf("audit salah: %+v %+v", e, snap)
	}
}

func TestPublishRejectedWhenRecipientUnreachable(t *testing.T) {
	store := letterAt(StatusInApproval, 3, 4)
	refs := newFakeRefs()
	refs.headlessUnits = []string{"Departemen SDM"}
	_, err := newTestLetterService(store, refs).Approve(context.Background(), agusUser, TransitionInput{LetterID: testLetterID})
	var e *apperr.InvalidStateError
	if !errors.As(err, &e) || !strings.Contains(e.Message, "Departemen SDM") {
		t.Errorf("error = %v, want InvalidStateError Departemen SDM", err)
	}
	if store.applied != nil || store.published != nil {
		t.Errorf("surat tidak boleh terbit: applied=%+v published=%+v", store.applied, store.published)
	}
}

func TestRecipientCanOpenSentLetter(t *testing.T) {
	store := letterAt(StatusInApproval, 3, 4)
	svc, _, inbox := newTestLetterServiceFull(store, newFakeRefs())
	inbox.holders = map[int64]bool{50: true}
	recipient := &auth.CurrentUser{ID: 50, Company: auth.UserCompany{Code: "PSP"}}
	ctx := context.Background()

	if _, err := svc.Get(ctx, recipient, testLetterID); !errors.Is(err, ErrLetterNotFound) {
		t.Errorf("surat belum terbit: error = %v, want ErrLetterNotFound", err)
	}
	store.detail.Status = StatusSent
	d, err := svc.Get(ctx, recipient, testLetterID)
	if err != nil || len(d.AllowedActions) != 0 {
		t.Errorf("penerima inbox: err=%v allowedActions=%v", err, d.AllowedActions)
	}
	if _, err := svc.Get(ctx, outsiderUser, testLetterID); !errors.Is(err, ErrLetterNotFound) {
		t.Errorf("bukan penerima: error = %v, want ErrLetterNotFound", err)
	}
}

func TestApprovalQRText(t *testing.T) {
	wib := time.FixedZone("WIB", 7*3600)
	d := &LetterDetail{
		LetterNumber: ptr("015/TI/PSP/X/2026"),
		Approval: &Approval{
			ApprovedAt: time.Date(2026, 10, 29, 14, 5, 12, 0, wib),
			ApprovedBy: PersonRef{Name: "Agus Setiawan", PositionName: "SVP Teknologi Informasi"},
		},
	}
	want := "Persetujuan Surat - Digital Office\nNomor: 015/TI/PSP/X/2026\nDisetujui oleh: Agus Setiawan\n" +
		"Jabatan: SVP Teknologi Informasi\nWaktu: 29-10-2026 14:05:12 WIB"
	if got := approvalQRText(d); got != want {
		t.Errorf("approvalQRText =\n%s\nwant\n%s", got, want)
	}
}

func TestApprovalQRRequiresApproval(t *testing.T) {
	store := letterAt(StatusInApproval, 3, 4)
	svc := newTestLetterService(store, newFakeRefs())
	var e *apperr.InvalidStateError
	if _, err := svc.ApprovalQR(context.Background(), agusUser, testLetterID); !errors.As(err, &e) {
		t.Errorf("belum disetujui: error = %v, want InvalidStateError", err)
	}

	store.detail.Status = StatusSent
	store.detail.LetterNumber = ptr("015/TI/PSP/X/2026")
	store.detail.Approval = &Approval{ApprovedAt: time.Now(), ApprovedBy: PersonRef{ID: 3, Name: "Agus"}}
	png, err := svc.ApprovalQR(context.Background(), andiUser, testLetterID)
	if err != nil || !strings.HasPrefix(string(png), "\x89PNG") {
		t.Errorf("sudah disetujui: err=%v, png %d byte", err, len(png))
	}
	if _, err := svc.ApprovalQR(context.Background(), outsiderUser, testLetterID); !errors.Is(err, ErrLetterNotFound) {
		t.Errorf("orang luar: error = %v, want ErrLetterNotFound", err)
	}
}
