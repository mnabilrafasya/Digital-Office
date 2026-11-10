package notifikasi

import (
	"strings"
	"unicode/utf8"
)

// letterInfo adalah data surat yang ditulis di pesan.
type letterInfo struct {
	Number          *string
	Title           *string
	Confidentiality string // kode lookup CONFIDENTIALITY
	SenderUnit      string
	DrafterName     string
}

// maxNoteRunes: catatan di pesan dipotong supaya pesan tetap ringkas.
const maxNoteRunes = 300

// messageText menyusun teks pesan WhatsApp untuk satu kejadian. Perihal surat Terbatas dan Rahasia
// tidak ditulis, karena WhatsApp berada di luar sistem.
func messageText(kind string, l letterInfo, actorName string, note *string) string {
	var b strings.Builder
	b.WriteString("*Digital Office PT Pupuk Sriwidjaja Palembang*\n")
	switch kind {
	case EventReviewRequested:
		b.WriteString("Ada surat yang menunggu review Anda.\n")
	case EventApprovalRequested:
		b.WriteString("Ada surat yang menunggu persetujuan dan tanda tangan Anda.\n")
	case EventReturned:
		b.WriteString("Surat dikembalikan kepada Anda oleh " + actorName + ".\n")
	case EventPublished:
		b.WriteString("Surat Anda telah disetujui dan terbit.\n")
	case EventReceived:
		b.WriteString("Ada surat masuk untuk Anda.\n")
	case EventDisposed:
		b.WriteString("Anda menerima disposisi surat dari " + actorName + ".\n")
	case EventRedirected:
		b.WriteString("Surat dialihkan kepada Anda oleh " + actorName + ".\n")
	}
	if l.Number != nil {
		b.WriteString("Nomor: " + *l.Number + "\n")
	}
	b.WriteString("Perihal: " + title(l) + "\n")
	switch kind {
	case EventReviewRequested, EventApprovalRequested:
		b.WriteString("Dari: " + l.DrafterName + ", " + l.SenderUnit + "\n")
	case EventReceived:
		b.WriteString("Dari: " + l.SenderUnit + "\n")
	}
	if note != nil && strings.TrimSpace(*note) != "" {
		b.WriteString("Catatan: " + truncate(strings.TrimSpace(*note), maxNoteRunes) + "\n")
	}
	b.WriteString("Silakan buka aplikasi Digital Office untuk menindaklanjuti.")
	return truncate(b.String(), maxMessageRunes)
}

// title adalah perihal yang boleh ditulis di pesan.
func title(l letterInfo) string {
	switch {
	case l.Confidentiality == "TERBATAS" || l.Confidentiality == "RAHASIA":
		return "(surat bersifat " + strings.ToLower(l.Confidentiality) + ", lihat di aplikasi)"
	case l.Title == nil || strings.TrimSpace(*l.Title) == "":
		return "-"
	}
	return *l.Title
}

// truncate memotong s menjadi maksimal n karakter (rune), diakhiri "..." bila terpotong.
func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-3]) + "..."
}
