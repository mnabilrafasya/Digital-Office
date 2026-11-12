package surat

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"office.backend/internal/audit"
	"office.backend/internal/platform/apperr"
)

// Surat terbit saat approver menyetujui (Decision.Action = Sign): dalam transaksi yang sama surat
// diberi nomor, tanggal (waktu disetujui), dan penandatangan, lalu masuk inbox penerima.

// DefaultNumberFormat dipakai bila perusahaan belum mengatur companies.letter_number_format.
const DefaultNumberFormat = "{SEQ}/{COMP}/{MM_ROMAN}/{YYYY}"

var romanMonths = [...]string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}

// FormatLetterNumber mengisi format nomor surat. Token: {SEQ} (nomor urut per perusahaan per bulan,
// minimal 3 digit), {COMP} (kode perusahaan), {MM_ROMAN} (bulan romawi), {YYYY} (tahun), dan
// {UNIT} (kode unit pengirim, opsional). Contoh format default: 015/PSP/X/2026.
func FormatLetterNumber(format string, seq int, unitCode, companyCode string, at time.Time) string {
	if format == "" {
		format = DefaultNumberFormat
	}
	return strings.NewReplacer(
		"{SEQ}", fmt.Sprintf("%03d", seq),
		"{UNIT}", unitCode,
		"{COMP}", companyCode,
		"{MM_ROMAN}", romanMonths[at.Month()-1],
		"{YYYY}", fmt.Sprint(at.Year()),
	).Replace(format)
}

// numberTokens adalah token yang boleh dipakai admin di format nomor surat. {UNIT} tetap dikenali
// FormatLetterNumber demi format lama, tetapi tidak ditawarkan: kode unit adalah kode internal HC.
var numberTokens = []string{"{SEQ}", "{COMP}", "{MM_ROMAN}", "{YYYY}"}

// requiredNumberTokens wajib ada supaya nomor surat unik: urutan, perusahaan (nomor unik di seluruh
// sistem), bulan, dan tahun (urutan mulai lagi setiap bulan).
var requiredNumberTokens = []string{"{SEQ}", "{COMP}", "{MM_ROMAN}", "{YYYY}"}

// maxLetterNumberLength adalah panjang kolom letters.letter_number dan companies.letter_number_format.
const maxLetterNumberLength = 100

// tokenPattern menemukan semua token {...} di format nomor.
var tokenPattern = regexp.MustCompile(`\{[^{}]*\}`)

// ValidateNumberFormat memeriksa format nomor surat dari admin perusahaan: hanya karakter ASCII
// yang terlihat, token dikenal, token wajib ada, dan nomor hasilnya muat di kolom.
func ValidateNumberFormat(format string) error {
	if format == "" {
		return apperr.Invalid("Format nomor surat wajib diisi")
	}
	if len(format) > maxLetterNumberLength {
		return apperr.Invalid(fmt.Sprintf("Format nomor surat maksimal %d karakter", maxLetterNumberLength))
	}
	for _, r := range format {
		if r < 0x20 || r > 0x7e {
			return apperr.Invalid("Format nomor surat hanya boleh berisi huruf, angka, dan tanda baca biasa")
		}
	}
	for _, tok := range tokenPattern.FindAllString(format, -1) {
		if !slices.Contains(numberTokens, tok) {
			return apperr.Invalid("Token " + tok + " tidak dikenal. Gunakan {SEQ}, {COMP}, {MM_ROMAN}, dan {YYYY}")
		}
	}
	if strings.ContainsAny(tokenPattern.ReplaceAllString(format, ""), "{}") {
		return apperr.Invalid("Tanda { dan } hanya untuk token, contoh {SEQ}")
	}
	for _, tok := range requiredNumberTokens {
		if !strings.Contains(format, tok) {
			return apperr.Invalid("Format nomor surat wajib memuat {SEQ}, {COMP}, {MM_ROMAN}, dan {YYYY}")
		}
	}
	// Nomor terpanjang: urutan 5 digit, kode perusahaan 20 karakter, bulan VIII
	longest := FormatLetterNumber(format, 99999, "", strings.Repeat("X", 20), time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	if len(longest) > maxLetterNumberLength {
		return apperr.Invalid(fmt.Sprintf("Nomor surat hasil format ini bisa lebih dari %d karakter", maxLetterNumberLength))
	}
	return nil
}

// EffectiveNumberFormat adalah format nomor yang benar-benar dipakai: format tersimpan perusahaan,
// atau DefaultNumberFormat bila kosong atau tidak lagi memenuhi aturan (mis. format lama tanpa
// {MM_ROMAN}, yang bisa menghasilkan nomor kembar setelah urutan direset bulanan).
func EffectiveNumberFormat(format string) string {
	if ValidateNumberFormat(format) != nil {
		return DefaultNumberFormat
	}
	return format
}

// NumberPeriod adalah periode nomor urut (YYYYMM, kolom letters.number_period): urutan mulai lagi
// dari 1 setiap bulan.
func NumberPeriod(at time.Time) int {
	return at.Year()*100 + int(at.Month())
}

// NumberPreview adalah contoh nomor surat ke-1 dengan format dan perusahaan tertentu.
func NumberPreview(format, companyCode string, at time.Time) string {
	return FormatLetterNumber(format, 1, "", companyCode, at)
}

// Publication adalah data yang ditulis ke surat saat terbit.
type Publication struct {
	Number     string
	Seq        int
	Period     int // YYYYMM (NumberPeriod)
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
	external    bool // surat eksternal keluar: hanya diberi nomor, tidak dikirim ke inbox penerima
}

// deliverSnapshot adalah isi audit_logs.snapshot saat surat terkirim ke inbox penerima.
type deliverSnapshot struct {
	LetterNumber string  `json:"letterNumber"`
	To           []int64 `json:"to"`
	CC           []int64 `json:"cc,omitempty"`
}

// publish menerbitkan surat yang disetujui approver terakhir: memastikan penerima masih bisa
// dijangkau, memberi nomor, mengirim ke inbox, lalu mencatat pengiriman (DELIVER, oleh sistem).
// Surat eksternal keluar hanya diberi nomor (penerimanya di luar sistem; lihat eksternal.go).
// Wajib dipanggil di dalam transaksi, setelah surat dikunci dan sebelum ApplyTransition (CHECK
// ck_letters_sent_complete mewajibkan nomor, tanggal, dan penandatangan sudah terisi saat status
// menjadi SENT).
func (s *Service) publish(ctx context.Context, t publishTarget) (string, error) {
	if t.external {
		return s.assignNumber(ctx, t, nil) // surat eksternal: tidak ada penerima internal
	}
	d, err := t.letters.LoadDraft(ctx, t.letterID)
	if err != nil {
		return "", err
	}
	if err := s.checkReachable(ctx, t.companyCode, d.Recipients); err != nil {
		return "", err
	}
	to, cc, heads, err := s.resolveRecipients(ctx, t.companyCode, d.Recipients)
	if err != nil {
		return "", err
	}
	number, err := s.assignNumber(ctx, t, heads)
	if err != nil {
		return "", err
	}
	if err := t.inbox.DeliverDirect(ctx, t.letterID, to, cc); err != nil {
		return "", err
	}
	if err := t.audit.Record(ctx, audit.Entry{
		CompanyCode: t.companyCode, EntityType: audit.EntityLetter, EntityID: t.letterID, EntityUUID: t.letterUUID,
		Action: audit.ActionDeliver, Snapshot: deliverSnapshot{LetterNumber: number, To: to, CC: cc},
	}); err != nil {
		return "", err
	}
	return number, nil
}

// assignNumber memberi nomor, tanggal, dan penandatangan surat (heads: pimpinan penerima tiap unit).
func (s *Service) assignNumber(ctx context.Context, t publishTarget, heads map[int64]int64) (string, error) {
	now := time.Now().In(s.loc)
	period := NumberPeriod(now)
	seq, unitCode, format, err := t.letters.NextNumber(ctx, t.letterID, period)
	if err != nil {
		return "", err
	}
	number := FormatLetterNumber(EffectiveNumberFormat(format), seq, unitCode, t.companyCode, now)
	if err := t.letters.Publish(ctx, t.letterID, Publication{
		Number: number, Seq: seq, Period: period, ApprovedAt: now, SignerID: t.signerID, UnitHeads: heads,
	}); err != nil {
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
