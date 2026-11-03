package surat

import (
	"context"
	"testing"
)

func TestCreateRecordsAudit(t *testing.T) {
	store := letterAt(StatusDraft, 0, 0)
	svc, audit := newTestLetterServiceWithAudit(store, newFakeRefs())
	if _, err := svc.Create(context.Background(), andiUser, completeInput()); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(audit.entries) != 1 {
		t.Fatalf("audit = %d entri, want 1", len(audit.entries))
	}
	if e := audit.entries[0]; e.Action != "CREATE" || e.EntityID != 1 || e.EntityUUID != testLetterID || e.ActorID != 15 {
		t.Errorf("audit CREATE salah: %+v", e)
	}
}

func TestOpeningSecretLetterIsRecorded(t *testing.T) {
	for _, tc := range []struct {
		code string
		want int
	}{{ConfidentialitySecret, 1}, {"TERBATAS", 0}, {"BIASA", 0}} {
		store := letterAt(StatusDraft, 0, 0)
		store.detail.Confidentiality = &LookupRef{Code: tc.code}
		svc, audit := newTestLetterServiceWithAudit(store, newFakeRefs())
		if _, err := svc.Get(context.Background(), andiUser, testLetterID); err != nil {
			t.Fatalf("%s: %v", tc.code, err)
		}
		if len(audit.entries) != tc.want || (tc.want == 1 && audit.entries[0].Action != "VIEW") {
			t.Errorf("%s: audit = %+v, want %d entri VIEW", tc.code, audit.entries, tc.want)
		}
	}
}

func TestDraftDownloadIsRecorded(t *testing.T) {
	store := letterAt(StatusDraft, 0, 0)
	svc, audit := newTestLetterServiceWithAudit(store, newFakeRefs())
	ctx := context.Background()
	if _, _, err := svc.SaveDocument(ctx, andiUser, DocumentInput{LetterID: testLetterID, FileName: "Naskah.docx", Data: testDocx}); err != nil {
		t.Fatal(err)
	}
	f, err := svc.OpenDocument(ctx, andiUser, testLetterID, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.Body.Close()
	last := audit.entries[len(audit.entries)-1]
	snap, _ := last.Snapshot.(downloadSnapshot)
	if last.Action != "DOWNLOAD" || last.ActorID != 15 || snap.Type != VersionDraft || snap.VersionNumber != 1 {
		t.Errorf("audit DOWNLOAD salah: %+v", last)
	}
}
