package auth

// Pembatas percobaan login di memori (per instance server).
// Bila API dijalankan lebih dari satu instance, batas ini perlu dipindah ke penyimpanan bersama.

import (
	"context"
	"sync"
	"time"
)

// Window adalah penghitung kegagalan dengan jendela waktu tetap: jendela dimulai pada kegagalan
// pertama dan berakhir setelah period. Setelah limit kegagalan, kunci diblokir sampai jendela habis.
type Window struct {
	mu      sync.Mutex
	limit   int
	period  time.Duration
	now     func() time.Time
	entries map[string]*entry
}

type entry struct {
	failures int
	resetAt  time.Time
}

// NewWindow membuat penghitung dengan batas limit kegagalan per period. now boleh nil (jam sistem).
func NewWindow(limit int, period time.Duration, now func() time.Time) *Window {
	if now == nil {
		now = time.Now
	}
	return &Window{limit: limit, period: period, now: now, entries: map[string]*entry{}}
}

// Blocked mengembalikan sisa waktu blokir untuk key, atau 0 bila key masih boleh mencoba.
func (w *Window) Blocked(key string) time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()

	e, ok := w.entries[key]
	if !ok {
		return 0
	}
	now := w.now()
	if !now.Before(e.resetAt) {
		delete(w.entries, key)
		return 0
	}
	if e.failures >= w.limit {
		return e.resetAt.Sub(now)
	}
	return 0
}

// Fail mencatat satu kegagalan untuk key.
func (w *Window) Fail(key string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := w.now()
	e, ok := w.entries[key]
	if !ok || !now.Before(e.resetAt) {
		w.entries[key] = &entry{failures: 1, resetAt: now.Add(w.period)}
		return
	}
	e.failures++
}

// Reset menghapus catatan kegagalan key.
func (w *Window) Reset(key string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.entries, key)
}

// Sweep membuang entri yang jendelanya sudah habis dan mengembalikan jumlah yang dibuang.
func (w *Window) Sweep() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := w.now()
	removed := 0
	for k, e := range w.entries {
		if !now.Before(e.resetAt) {
			delete(w.entries, k)
			removed++
		}
	}
	return removed
}

// Len mengembalikan jumlah entri yang tersimpan.
func (w *Window) Len() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.entries)
}

// Aturan pembatasan login.
const (
	PerAccountLimit = 5  // kegagalan per IP+NIK
	PerIPLimit      = 30 // kegagalan per IP
	LoginPeriod     = 15 * time.Minute
	JanitorInterval = 5 * time.Minute
)

// LoginLimiter membatasi percobaan login gagal per IP+NIK dan per IP.
type LoginLimiter struct {
	account *Window
	ip      *Window
}

// NewLoginLimiter membuat LoginLimiter dengan aturan bawaan. now boleh nil (jam sistem).
func NewLoginLimiter(now func() time.Time) *LoginLimiter {
	return &LoginLimiter{
		account: NewWindow(PerAccountLimit, LoginPeriod, now),
		ip:      NewWindow(PerIPLimit, LoginPeriod, now),
	}
}

func accountKey(ip, employeeNumber string) string { return ip + "|" + employeeNumber }

// Check mengembalikan sisa waktu blokir (0 = boleh mencoba). Dipanggil sebelum bcrypt.
func (l *LoginLimiter) Check(ip, employeeNumber string) time.Duration {
	return max(l.account.Blocked(accountKey(ip, employeeNumber)), l.ip.Blocked(ip))
}

// Failure mencatat login gagal.
func (l *LoginLimiter) Failure(ip, employeeNumber string) {
	l.account.Fail(accountKey(ip, employeeNumber))
	l.ip.Fail(ip)
}

// Success mereset hitungan IP+NIK setelah login berhasil.
func (l *LoginLimiter) Success(ip, employeeNumber string) {
	l.account.Reset(accountKey(ip, employeeNumber))
}

// Sweep membersihkan entri kedaluwarsa.
func (l *LoginLimiter) Sweep() {
	l.account.Sweep()
	l.ip.Sweep()
}

// RunJanitor membersihkan entri kedaluwarsa secara berkala sampai ctx selesai (server shutdown).
func (l *LoginLimiter) RunJanitor(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.Sweep()
		}
	}
}
