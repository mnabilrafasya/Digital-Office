// Package notifikasi menyiapkan pemberitahuan WhatsApp untuk kejadian surat, tanpa tabel tambahan:
//
//  1. Setelah aksi surat (ajukan, setujui, kembalikan, terbit, disposisi/alihkan) berhasil disimpan,
//     service memanggil Notifier.Notify. Kejadian masuk antrean di memori; aksi surat tidak
//     menunggu dan tidak pernah gagal karena WhatsApp.
//  2. Worker menyusun pesan (data surat dan NIK penerima dibaca dari tabel yang sudah ada) lalu
//     mengirimnya lewat WhatsAppSender. Gagal kirim dicoba ulang beberapa kali.
//
// Antrean ada di memori: pesan yang belum terkirim hilang bila server dimatikan.
//
// WhatsAppSender adalah "colokan" untuk layanan WhatsApp yang disediakan terpisah. Selama belum
// dipasang (nil), Notify tidak melakukan apa pun dan worker tidak berjalan.
package notifikasi

import (
	"context"
	"time"
)

// Jenis kejadian.
const (
	EventReviewRequested   = "MENUNGGU_REVIEW"      // surat menunggu review penerima pesan
	EventApprovalRequested = "MENUNGGU_PERSETUJUAN" // surat menunggu persetujuan & tanda tangan
	EventReturned          = "DIKEMBALIKAN"         // surat dikembalikan kepada penerima pesan
	EventPublished         = "TERBIT"               // surat milik penerima pesan (drafter) terbit
	EventReceived          = "SURAT_MASUK"          // surat terbit masuk ke inbox penerima pesan
	EventDisposed          = "DISPOSISI"            // penerima pesan mendapat disposisi
	EventRedirected        = "PENGALIHAN"           // surat dialihkan kepada penerima pesan
)

// Event adalah satu kejadian surat yang perlu diberitahukan kepada Recipients.
type Event struct {
	Kind       string
	LetterID   int64
	Recipients []int64 // user penerima pesan; ActorID dan duplikat diabaikan
	ActorID    int64   // pelaku aksi (tidak diberi pesan tentang aksinya sendiri)
	ActorName  string
	Note       *string // catatan pengembalian/disposisi/pengalihan
}

// Outgoing adalah satu pesan WhatsApp yang siap dikirim.
type Outgoing struct {
	EmployeeNumber string // NIK penerima; layanan WhatsApp yang mencari nomor HP-nya
	Event          string
	Text           string
}

// WhatsAppSender mengirim satu pesan WhatsApp. Ini colokan untuk layanan WhatsApp mentor:
// implementasinya cukup memanggil layanan tersebut dengan NIK dan teks pesan, lalu mengembalikan
// error bila gagal (pesan akan dicoba ulang).
type WhatsAppSender interface {
	Send(ctx context.Context, msg Outgoing) error
}

// Pengaturan pengiriman.
const (
	queueSize       = 1000             // kejadian/pesan yang boleh menunggu di antrean
	sendTimeout     = 30 * time.Second // batas waktu satu kali kirim
	maxMessageRunes = 2000             // batas panjang pesan
)

// retryDelays adalah jeda sebelum percobaan ulang ke-1, ke-2, dan ke-3. Setelah itu pesan
// dibuang dan dicatat di log.
var retryDelays = []time.Duration{10 * time.Second, time.Minute, 5 * time.Minute}
