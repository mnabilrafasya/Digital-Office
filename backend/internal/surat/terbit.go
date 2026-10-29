package surat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"office.backend/internal/platform/apperr"
)

// Surat terbit saat approver menyetujui (Decision.Action = Sign): dalam transaksi yang sama surat
// diberi nomor, tanggal (waktu disetujui), dan penandatangan, lalu masuk inbox penerima.

// defaultNumberFormat dipakai bila perusahaan belum mengatur companies.letter_number_format.
const defaultNumberFormat = "{SEQ}/{UNIT}/{COMP}/{MM_ROMAN}/{YYYY}"

var romanMonths = [...]string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}

// FormatLetterNumber mengisi format nomor surat. Token: {SEQ} (nomor urut per unit per tahun,
// minimal 3 digit), {UNIT} (kode unit pengirim), {COMP} (kode perusahaan), {MM_ROMAN} (bulan
// romawi), {YYYY} (tahun). Contoh: 015/TI/PSP/X/2026.
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

// publish menerbitkan surat yang sedang disetujui approver: memastikan penerima masih bisa
// dijangkau, memberi nomor, lalu mengirim ke inbox. Wajib dipanggil di dalam transaksi, setelah
// surat dikunci dan sebelum ApplyTransition (CHECK ck_letters_sent_complete mewajibkan nomor,
// tanggal, dan penandatangan sudah terisi saat status menjadi SENT).
func (s *Service) publish(ctx context.Context, letters Store, inbox InboxStore, companyCode string, letterID, signerID int64) (string, error) {
	d, err := letters.LoadDraft(ctx, letterID)
	if err != nil {
		return "", err
	}
	if err := s.checkReachable(ctx, companyCode, d.Recipients); err != nil {
		return "", err
	}
	to, cc, heads, err := s.resolveRecipients(ctx, companyCode, d.Recipients)
	if err != nil {
		return "", err
	}

	now := time.Now().In(s.loc)
	seq, unitCode, format, err := letters.NextNumber(ctx, letterID, now.Year())
	if err != nil {
		return "", err
	}
	number := FormatLetterNumber(format, seq, unitCode, companyCode, now)
	if err := letters.Publish(ctx, letterID, Publication{
		Number: number, Seq: seq, Year: now.Year(), ApprovedAt: now, SignerID: signerID, UnitHeads: heads,
	}); err != nil {
		return "", err
	}
	if err := inbox.DeliverDirect(ctx, letterID, to, cc); err != nil {
		return "", err
	}
	return number, nil
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
