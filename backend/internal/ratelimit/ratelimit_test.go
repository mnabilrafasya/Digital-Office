package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// fakeClock adalah jam palsu yang bisa dimajukan manual.
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time          { return c.t }
func (c *fakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

func newClock() *fakeClock { return &fakeClock{t: time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)} }

func TestLoginLimiterPerAccount(t *testing.T) {
	clock := newClock()
	l := NewLoginLimiter(clock.Now)

	for i := 1; i <= PerAccountLimit; i++ {
		if d := l.Check("10.0.0.1", "2015047"); d != 0 {
			t.Fatalf("percobaan ke-%d diblokir %v, want boleh", i, d)
		}
		l.Failure("10.0.0.1", "2015047")
		clock.Advance(time.Minute)
	}

	// Percobaan ke-6 diblokir sampai 15 menit sejak kegagalan pertama
	if d := l.Check("10.0.0.1", "2015047"); d != 10*time.Minute {
		t.Fatalf("blokir = %v, want 10m (15m - 5m berlalu)", d)
	}
	// NIK lain dari IP yang sama dan IP lain untuk NIK yang sama tidak terpengaruh
	if l.Check("10.0.0.1", "2003035") != 0 || l.Check("10.0.0.2", "2015047") != 0 {
		t.Fatal("kunci lain ikut diblokir")
	}

	clock.Advance(10 * time.Minute)
	if d := l.Check("10.0.0.1", "2015047"); d != 0 {
		t.Fatalf("setelah jendela habis masih diblokir %v", d)
	}
}

func TestLoginLimiterSuccessResetsAccount(t *testing.T) {
	l := NewLoginLimiter(newClock().Now)
	for i := 0; i < PerAccountLimit-1; i++ {
		l.Failure("10.0.0.1", "2015047")
	}
	l.Success("10.0.0.1", "2015047")
	for i := 0; i < PerAccountLimit-1; i++ {
		l.Failure("10.0.0.1", "2015047")
	}
	if d := l.Check("10.0.0.1", "2015047"); d != 0 {
		t.Fatalf("setelah login sukses hitungan tidak direset: blokir %v", d)
	}
}

func TestLoginLimiterPerIP(t *testing.T) {
	l := NewLoginLimiter(newClock().Now)
	for i := 0; i < PerIPLimit; i++ {
		l.Failure("10.0.0.1", fmt.Sprintf("nik-%d", i)) // menyebar ke banyak NIK
	}
	if d := l.Check("10.0.0.1", "nik-baru"); d != LoginPeriod {
		t.Fatalf("blokir per IP = %v, want %v", d, LoginPeriod)
	}
	if l.Check("10.0.0.2", "nik-baru") != 0 {
		t.Fatal("IP lain ikut diblokir")
	}
}

func TestSweepAndJanitor(t *testing.T) {
	clock := newClock()
	w := NewWindow(5, time.Minute, clock.Now)
	w.Fail("a")
	w.Fail("b")
	clock.Advance(2 * time.Minute)
	w.Fail("c")
	if n := w.Sweep(); n != 2 || w.Len() != 1 {
		t.Fatalf("Sweep() = %d, sisa %d; want 2, 1", n, w.Len())
	}

	// Janitor berhenti saat context dibatalkan (server shutdown)
	l := NewLoginLimiter(nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { l.RunJanitor(ctx, time.Millisecond); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("janitor tidak berhenti setelah context dibatalkan")
	}
}
