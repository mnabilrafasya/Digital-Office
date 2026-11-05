package notifikasi

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

func sp(s string) *string { return &s }

func TestMessageText(t *testing.T) {
	l := letterInfo{Number: sp("015/PSP/2026"), Title: sp("Rapat Anggaran"), Confidentiality: "BIASA",
		SenderUnit: "Dep. Keuangan", DrafterName: "Andi"}

	got := messageText(EventReviewRequested, l, "Andi", nil)
	for _, want := range []string{"menunggu review Anda", "Nomor: 015/PSP/2026", "Perihal: Rapat Anggaran", "Dari: Andi, Dep. Keuangan"} {
		if !strings.Contains(got, want) {
			t.Errorf("pesan review tidak memuat %q:\n%s", want, got)
		}
	}

	// Surat Rahasia: perihal disembunyikan; catatan panjang dipotong
	l.Confidentiality, l.Number = "RAHASIA", nil
	got = messageText(EventReturned, l, "Agus", sp(strings.Repeat("x", 500)))
	if strings.Contains(got, "Rapat Anggaran") || strings.Contains(got, "Nomor:") || !strings.Contains(got, "bersifat rahasia") ||
		!strings.Contains(got, "dikembalikan kepada Anda oleh Agus") || strings.Contains(got, strings.Repeat("x", 301)) {
		t.Errorf("pesan rahasia salah:\n%s", got)
	}

	// Batas total panjang pesan
	l.Confidentiality, l.Title = "BIASA", sp(strings.Repeat("é", 3000))
	if n := utf8.RuneCountInString(messageText(EventReceived, l, "", nil)); n != maxMessageRunes {
		t.Errorf("panjang pesan = %d, want %d", n, maxMessageRunes)
	}
}

func TestRecipientIDs(t *testing.T) {
	got := recipientIDs(Event{Recipients: []int64{4, 3, 4, 0, 9}, ActorID: 3})
	if !slices.Equal(got, []int64{4, 9}) {
		t.Errorf("recipientIDs = %v, want [4 9]", got)
	}
}

func TestDisabledNotifierDoesNothing(t *testing.T) {
	n := New(nil, nil)
	n.Notify(Event{Kind: EventPublished, Recipients: []int64{1}})
	if n.Enabled() || len(n.events) != 0 {
		t.Error("Notifier tanpa layanan WhatsApp tetap mengantre")
	}
	var none *Notifier
	none.Notify(Event{}) // nil aman
}

// fakeReader: NIK user 4 = "2003035", user 9 = "2012060"; user lain tidak aktif.
type fakeReader struct{}

func (fakeReader) letterInfo(context.Context, int64) (letterInfo, error) {
	return letterInfo{Title: sp("Rapat"), Confidentiality: "BIASA", SenderUnit: "Dep. Keuangan", DrafterName: "Andi"}, nil
}

func (fakeReader) employeeNumbers(_ context.Context, ids []int64) ([]string, error) {
	nik := map[int64]string{4: "2003035", 9: "2012060"}
	var out []string
	for _, id := range ids {
		if n, ok := nik[id]; ok {
			out = append(out, n)
		}
	}
	return out, nil
}

// fakeSender gagal untuk NIK di failFor.
type fakeSender struct {
	failFor string
	got     []Outgoing
}

func (f *fakeSender) Send(_ context.Context, m Outgoing) error {
	f.got = append(f.got, m)
	if m.EmployeeNumber == f.failFor {
		return errors.New("layanan WhatsApp tidak menjawab")
	}
	return nil
}

func TestComposeAndRetry(t *testing.T) {
	s := &fakeSender{failFor: "2012060"}
	var delays []time.Duration
	n := &Notifier{reader: fakeReader{}, sender: s, events: make(chan Event, 10), messages: make(chan pending, 10),
		after: func(d time.Duration, f func()) { delays = append(delays, d); f() }} // jadwal ulang langsung
	ctx := context.Background()

	n.Notify(Event{Kind: EventDisposed, LetterID: 1, Recipients: []int64{9, 4, 15, 4}, ActorID: 15, ActorName: "Sri"})
	n.compose(ctx, <-n.events)
	if len(n.messages) != 2 { // 15 pelaku, duplikat 4 diabaikan; user tanpa NIK aktif dilewati
		t.Fatalf("pesan diantre = %d, want 2", len(n.messages))
	}
	// Kirim sampai antrean kosong: 2003035 sekali, 2012060 gagal terus (1 + 3 percobaan ulang)
	for len(n.messages) > 0 {
		n.send(ctx, <-n.messages)
	}
	var ok, fail int
	for _, m := range s.got {
		if !strings.Contains(m.Text, "disposisi surat dari Sri") || m.Event != EventDisposed {
			t.Errorf("pesan salah: %+v", m)
		}
		if m.EmployeeNumber == "2012060" {
			fail++
		} else {
			ok++
		}
	}
	if ok != 1 || fail != 1+len(retryDelays) || !slices.Equal(delays, retryDelays) {
		t.Errorf("kirim ok=%d gagal=%d jeda=%v", ok, fail, delays)
	}
}

func TestSendTest(t *testing.T) {
	user := &auth.CurrentUser{ID: 1, Name: "Admin", Company: auth.UserCompany{Code: "PSP"}}
	err := NewHandler(nil, nil).sendTest(context.Background(), user, testRequest{EmployeeNumber: "2015047"})
	var unavailable *apperr.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Message != "Layanan WhatsApp belum terpasang" {
		t.Errorf("tanpa layanan WhatsApp: error = %v, want 503 belum terpasang", err)
	}
	h := NewHandler(nil, &fakeSender{})
	var invalid *apperr.ValidationError
	for _, req := range []testRequest{{EmployeeNumber: "  "}, {EmployeeNumber: "1", Message: strings.Repeat("a", 2001)}} {
		if err := h.sendTest(context.Background(), user, req); !errors.As(err, &invalid) {
			t.Errorf("%+v: error = %v, want validasi", req, err)
		}
	}
}
