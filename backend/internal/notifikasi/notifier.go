package notifikasi

import (
	"context"
	"log"
	"slices"
	"time"
)

// reader adalah data yang dibaca Notifier (dipenuhi Repository; diganti tiruan di unit test).
type reader interface {
	letterInfo(ctx context.Context, letterID int64) (letterInfo, error)
	employeeNumbers(ctx context.Context, ids []int64) ([]string, error)
}

// senders adalah jumlah pengiriman yang berjalan bersamaan.
const senders = 4

// pending adalah pesan di antrean beserta percobaan ke berapa.
type pending struct {
	msg     Outgoing
	attempt int
}

// Notifier mengantre kejadian surat lalu mengirim pesannya di latar belakang.
type Notifier struct {
	reader   reader
	sender   WhatsAppSender
	events   chan Event
	messages chan pending
	after    func(time.Duration, func()) // penjadwal percobaan ulang (time.AfterFunc)
}

// New membuat Notifier. sender nil = layanan WhatsApp belum dipasang: Notify tidak melakukan apa pun.
func New(r *Repository, sender WhatsAppSender) *Notifier {
	return &Notifier{
		reader: r, sender: sender,
		events: make(chan Event, queueSize), messages: make(chan pending, queueSize),
		after: func(d time.Duration, f func()) { time.AfterFunc(d, f) },
	}
}

// Enabled: layanan WhatsApp sudah dipasang.
func (n *Notifier) Enabled() bool {
	return n != nil && n.sender != nil
}

// Notify mengantre kejadian e tanpa menunggu. Dipanggil setelah aksi surat berhasil disimpan.
func (n *Notifier) Notify(e Event) {
	if !n.Enabled() {
		return
	}
	select {
	case n.events <- e:
	default:
		log.Printf("Pemberitahuan %s surat %d dibuang: antrean penuh", e.Kind, e.LetterID)
	}
}

// Run menyusun dan mengirim pesan sampai ctx dibatalkan.
func (n *Notifier) Run(ctx context.Context) {
	log.Println("Pemberitahuan WhatsApp aktif")
	for range senders {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case p := <-n.messages:
					n.send(ctx, p)
				}
			}
		}()
	}
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-n.events:
			n.compose(ctx, e)
		}
	}
}

// compose menyusun pesan kejadian e untuk setiap penerima aktif lalu mengantrekannya.
func (n *Notifier) compose(ctx context.Context, e Event) {
	ids := recipientIDs(e)
	if len(ids) == 0 {
		return
	}
	info, err := n.reader.letterInfo(ctx, e.LetterID)
	if err != nil {
		log.Printf("Pemberitahuan %s surat %d gagal disusun: %v", e.Kind, e.LetterID, err)
		return
	}
	niks, err := n.reader.employeeNumbers(ctx, ids)
	if err != nil {
		log.Printf("Pemberitahuan %s surat %d gagal disusun: %v", e.Kind, e.LetterID, err)
		return
	}
	text := messageText(e.Kind, info, e.ActorName, e.Note)
	for _, nik := range niks {
		n.enqueue(pending{msg: Outgoing{EmployeeNumber: nik, Event: e.Kind, Text: text}})
	}
}

func (n *Notifier) enqueue(p pending) {
	select {
	case n.messages <- p:
	default:
		log.Printf("Pemberitahuan %s ke NIK %s dibuang: antrean penuh", p.msg.Event, p.msg.EmployeeNumber)
	}
}

// send mengirim satu pesan; gagal dijadwalkan ulang sesuai retryDelays.
func (n *Notifier) send(ctx context.Context, p pending) {
	sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
	err := n.sender.Send(sendCtx, p.msg)
	cancel()
	if err == nil {
		return
	}
	if p.attempt >= len(retryDelays) {
		log.Printf("Pemberitahuan %s ke NIK %s gagal setelah %d percobaan, dibuang: %v", p.msg.Event, p.msg.EmployeeNumber, p.attempt+1, err)
		return
	}
	delay := retryDelays[p.attempt]
	log.Printf("Pemberitahuan %s ke NIK %s gagal (percobaan %d), coba lagi dalam %s: %v", p.msg.Event, p.msg.EmployeeNumber, p.attempt+1, delay, err)
	p.attempt++
	n.after(delay, func() { n.enqueue(p) })
}

// recipientIDs adalah penerima pesan tanpa duplikat, tanpa pelaku aksi.
func recipientIDs(e Event) []int64 {
	var out []int64
	for _, id := range e.Recipients {
		if id > 0 && id != e.ActorID && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}
