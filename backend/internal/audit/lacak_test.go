package audit

import (
	"strings"
	"testing"
	"time"
)

var (
	andi = PersonRef{ID: 8, Name: "Andi"}
	dewi = PersonRef{ID: 7, Name: "Dewi"}
	sri  = PersonRef{ID: 15, Name: "Sri"}
	maya = PersonRef{ID: 16, Name: "Maya"}
	joko = PersonRef{ID: 17, Name: "Joko"}
)

func TestPosition(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	step2 := 2
	approved := now.Add(-time.Hour)
	tests := []struct {
		name    string
		c       letterCore
		label   string
		holder  int64 // 0 = tidak ada
		waiting int   // -1 = tidak dihitung
		stuck   bool
	}{
		{"draf", letterCore{Status: statusDraft, Drafter: andi, Since: now.Add(-100 * time.Hour)},
			"Masih berupa draf", 8, 100, false}, // draf tidak pernah dianggap tertahan
		{"di reviewer 2, belum lama", letterCore{Status: statusInReview, Holder: &dewi, HolderStep: &step2, Since: now.Add(-22 * time.Hour)},
			"Sedang diperiksa Reviewer 2", 7, 22, false},
		{"di reviewer 2, tertahan", letterCore{Status: statusInReview, Holder: &dewi, HolderStep: &step2, Since: now.Add(-48 * time.Hour)},
			"Sedang diperiksa Reviewer 2", 7, 48, true},
		{"dikembalikan approver", letterCore{Status: statusReturnedToReviewer, Holder: &dewi, HolderStep: &step2, Since: now},
			"(dikembalikan approver)", 7, 0, false},
		{"di approver", letterCore{Status: statusInApproval, Holder: &sri, Since: now.Add(-3 * time.Hour)},
			"Menunggu persetujuan approver", 15, 3, false},
		{"dikembalikan ke drafter", letterCore{Status: statusReturnedToDrafter, Holder: &andi, Since: now.Add(-50 * time.Hour)},
			"Dikembalikan ke pembuat surat", 8, 50, true},
		{"terbit", letterCore{Status: statusSent, ApprovedAt: &approved, Since: now.Add(-2 * time.Hour)},
			"Sudah terbit", 0, -1, false},
	}
	for _, tt := range tests {
		p := position(&tt.c, now)
		holder := int64(0)
		if p.Holder != nil {
			holder = p.Holder.ID
		}
		waiting := -1
		if p.WaitingHours != nil {
			waiting = *p.WaitingHours
		}
		if !strings.Contains(p.Label, tt.label) || holder != tt.holder || waiting != tt.waiting || p.Stuck != tt.stuck {
			t.Errorf("%s: %+v (holder %d, waiting %d)", tt.name, p, holder, waiting)
		}
	}
	if p := position(&letterCore{Status: statusSent, ApprovedAt: &approved}, now); !p.Since.Equal(approved) {
		t.Errorf("surat terbit: since = %v, want waktu disetujui", p.Since)
	}
}

func TestTimeline(t *testing.T) {
	created := time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC)
	v2 := 2
	c := &letterCore{Drafter: andi, CreatedAt: created}
	acts := []letterAction{
		{Action: "SUBMIT", ActorRole: "DRAFTER", Actor: andi, Target: &dewi, ToStatus: statusInReview},
		{Action: "READ", ActorRole: "REVIEWER", Actor: dewi, ToStatus: statusInReview},
		{Action: "RETURN", ActorRole: "REVIEWER", Actor: dewi, Target: &andi, ToStatus: statusReturnedToDrafter, Note: ptr("Perbaiki lampiran")},
		{Action: "SAVE", ActorRole: "DRAFTER", Actor: andi, VersionNumber: &v2},
		{Action: "APPROVE", ActorRole: "REVIEWER", Actor: dewi, Target: &sri, ToStatus: statusInApproval},
		{Action: "SIGN", ActorRole: "APPROVER", Actor: sri, ToStatus: statusSent},
	}

	full := timeline(c, acts, ScopeFull)
	var labels []string
	for _, e := range full {
		labels = append(labels, e.Label)
	}
	want := "Draf dibuat|Dikirim untuk diperiksa|Membuka surat|Dikembalikan ke pembuat surat|Menyimpan naskah versi 2|" +
		"Disetujui reviewer, diteruskan ke approver|Disetujui approver, surat terbit"
	if got := strings.Join(labels, "|"); got != want {
		t.Errorf("label =\n%s\nwant\n%s", got, want)
	}
	if !full[0].At.Equal(created) || full[3].Note == nil || *full[3].Note != "Perbaiki lampiran" {
		t.Errorf("isi timeline salah: %+v", full)
	}

	// Penerima inbox hanya melihat kejadian terbit (tanpa catatan review internal)
	recipient := timeline(c, acts, ScopeRecipient)
	if len(recipient) != 1 || recipient[0].Action != "SIGN" {
		t.Errorf("timeline penerima = %+v, want hanya SIGN", recipient)
	}
}

func TestBuildDistribution(t *testing.T) {
	// Sri (TO) mendisposisikan ke Maya dan unit Joko; Maya mengalihkan ke Sri lagi; Joko juga CC langsung.
	at := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	row := func(id, owner, parent int64, p PersonRef, via, status string) inboxRow {
		return inboxRow{OwnerID: owner, ParentID: parent,
			Node: DistributionNode{InboxItemID: id, Recipient: p, Via: via, Status: status, ReceivedAt: at}}
	}
	rows := []inboxRow{
		row(1, 15, 0, sri, "TO", "DISPOSED"),
		row(2, 17, 0, joko, "CC", "UNREAD"),
		row(3, 16, 1, maya, "DISPOSITION", "REDIRECTED"),
		row(4, 17, 1, joko, "DISPOSITION", "READ"),
		row(5, 15, 3, sri, "REDIRECT", "UNREAD"),
	}

	all := buildDistribution(rows, 0)
	if len(all) != 2 || all[0].InboxItemID != 1 || len(all[0].Forwards) != 2 ||
		all[0].Forwards[0].InboxItemID != 3 || all[0].Forwards[0].Forwards[0].InboxItemID != 5 ||
		all[1].InboxItemID != 2 || all[1].Forwards == nil {
		t.Fatalf("pohon lengkap salah: %+v", all)
	}

	// Cabang Maya: hanya itemnya dan turunannya
	if b := buildDistribution(rows, 16); len(b) != 1 || b[0].InboxItemID != 3 || b[0].Forwards[0].InboxItemID != 5 {
		t.Errorf("cabang Maya salah: %+v", b)
	}
	// Cabang Sri: item 5 berada di bawah item 1 miliknya sendiri, jadi tidak menjadi akar kedua
	if b := buildDistribution(rows, 15); len(b) != 1 || b[0].InboxItemID != 1 {
		t.Errorf("cabang Sri salah: %+v", b)
	}
	// Cabang Joko: CC langsung dan disposisi dari Sri, dua akar terpisah
	if b := buildDistribution(rows, 17); len(b) != 2 || b[0].InboxItemID != 2 || b[1].InboxItemID != 4 {
		t.Errorf("cabang Joko salah: %+v", b)
	}
}

func ptr(s string) *string { return &s }
