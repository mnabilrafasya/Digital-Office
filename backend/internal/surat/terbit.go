package surat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"office.backend/internal/audit"
	"office.backend/internal/platform/apperr"
)

// Surat terbit saat approver menyetujui (Decision.Action = Sign): dalam transaksi yang sama surat
// diberi nomor, tanggal (waktu disetujui), dan penandatangan, lalu masuk inbox penerima.

// defaultNumberFormat dipakai bila perusahaan belum mengatur companies.letter_number_format.
const defaultNumberFormat = "{SEQ}/{COMP}/{MM_ROMAN}/{YYYY}"

var romanMonths = [...]string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}

// FormatLetterNumber mengisi format nomor surat. Token: {SEQ} (nomor urut per perusahaan per tahun,
// minimal 3 digit), {COMP} (kode perusahaan), {MM_ROMAN} (bulan romawi), {YYYY} (tahun), dan
// {UNIT} (kode unit pengirim, opsional). Contoh format default: 015/PSP/X/2026.
func FormatLetterNumber(format string, seq int, unitCode, companyCode string, at time.Time) string {
	if format == "" {
		format = defaultNumberFormat
	}
	return strings.NewReplacer(
		"{SEQ}", fmt.Sprintf("%03d", seq),
		"{UNIT}", unitCode,
		"{COMP}", companyCode,
		"{MM_ROMAN}", romanMonths[at.Month()-1],
		"{YYYY}", fmt.Sprint(at.Year()),
	).Replace(format)
}

// Publication adalah data yang ditulis ke surat saat terbit.
type Publication struct {
	Number     string
	Seq        int
	Year       int
	ApprovedAt time.Time
	SignerID   int64
	UnitHeads  map[int64]int64 // unit penerima -> pimpinan yang menerima (letter_recipients.resolved_head_id)
}

// publishTarget adalah surat yang diterbitkan beserta repository di dalam transaksinya.
type publishTarget struct {
	letters     Store
	inbox       InboxStore
	audit       AuditStore
	companyCode string
	letterID    int64
	letterUUID  string
	signerID    int64
}

// deliverSnapshot adalah isi audit_logs.snapshot saat surat terkirim ke inbox penerima.
type deliverSnapshot struct {
	LetterNumber string  `json:"letterNumber"`
	To           []int64 `json:"to"`
	CC           []int64 `json:"cc,omitempty"`
}

// published adalah hasil publish: nomor surat dan penerima di inbox (Kepada dan Tembusan).
type published struct {
	number string
	to, cc []int64
}

// publish menerbitkan surat yang sedang disetujui approver: memastikan penerima masih bisa
// dijangkau, memberi nomor, mengirim ke inbox, lalu mencatat pengiriman (DELIVER, oleh sistem).
// Wajib dipanggil di dalam transaksi, setelah surat dikunci dan sebelum ApplyTransition (CHECK
// ck_letters_sent_complete mewajibkan nomor, tanggal, dan penandatangan sudah terisi saat status
// menjadi SENT).
func (s *Service) publish(ctx context.Context, t publishTarget) (published, error) {
	letters, companyCode, letterID := t.letters, t.companyCode, t.letterID
	d, err := letters.LoadDraft(ctx, letterID)
	if err != nil {
		return published{}, err
	}
	if err := s.checkReachable(ctx, companyCode, d.Recipients); err != nil {
		return published{}, err
	}
	to, cc, heads, err := s.resolveRecipients(ctx, companyCode, d.Recipients)
	if err != nil {
		return published{}, err
	}

	now := time.Now().In(s.loc)
	seq, unitCode, format, err := letters.NextNumber(ctx, letterID, now.Year())
	if err != nil {
		return published{}, err
	}
	number := FormatLetterNumber(format, seq, unitCode, companyCode, now)
	if err := letters.Publish(ctx, letterID, Publication{
		Number: number, Seq: seq, Year: now.Year(), ApprovedAt: now, SignerID: t.signerID, UnitHeads: heads,
	}); err != nil {
		return published{}, err
	}
	if err := t.inbox.DeliverDirect(ctx, letterID, to, cc); err != nil {
		return published{}, err
	}
	if err := t.audit.Record(ctx, audit.Entry{
		CompanyCode: companyCode, EntityType: audit.EntityLetter, EntityID: letterID, EntityUUID: t.letterUUID,
		Action: audit.ActionDeliver, Snapshot: deliverSnapshot{LetterNumber: number, To: to, CC: cc},
	}); err != nil {
		return published{}, err
	}
	return published{number: number, to: to, cc: cc}, nil
}

// resolveRecipients mengubah penerima surat menjadi user: unit -> satu pimpinan aktif, alias ->
// semua anggotanya. Satu orang hanya menerima satu kali; Kepada (TO) menang atas Tembusan (CC).
func (s *Service) resolveRecipients(ctx context.Context, companyCode string, recipients []RecipientRef) (to, cc []int64, heads map[int64]int64, err error) {
	var unitIDs, aliasIDs []int64
	for _, r := range recipients {
		if r.UnitID != 0 {
			unitIDs = append(unitIDs, r.UnitID)
		} else {
			aliasIDs = append(aliasIDs, r.AliasLevelID)
		}
	}
	heads, err = s.repos.Directory.UnitHeadIDs(ctx, companyCode, unitIDs)
	if err != nil {
		return nil, nil, nil, err
	}
	members, err := s.repos.Directory.AliasMemberIDs(ctx, companyCode, aliasIDs)
	if err != nil {
		return nil, nil, nil, err
	}

	types := map[int64]string{}
	var order []int64
	add := func(userID int64, typ string) {
		prev, seen := types[userID]
		if !seen {
			order = append(order, userID)
		}
		if !seen || (prev == RecipientCC && typ == RecipientTo) {
			types[userID] = typ
		}
	}
	for _, r := range recipients {
		if r.UnitID != 0 {
			head, ok := heads[r.UnitID]
			if !ok { // pimpinan dinonaktifkan di sela pengecekan; anggap tidak terjangkau
				return nil, nil, nil, &apperr.InvalidStateError{Message: "Surat belum bisa dikirim karena ada unit penerima tanpa pimpinan aktif."}
			}
			add(head, r.Type)
			continue
		}
		for _, userID := range members[r.AliasLevelID] {
			add(userID, r.Type)
		}
	}
	for _, userID := range order {
		if types[userID] == RecipientTo {
			to = append(to, userID)
		} else {
			cc = append(cc, userID)
		}
	}
	return to, cc, heads, nil
}
