// Package upload mengatur antrean unggahan file. Semua unggahan berbagi satu jatah memori: setiap
// unggahan memesan jatah sebesar ukuran request-nya sebelum isinya dibaca. Bila jatah tidak cukup,
// unggahan menunggu giliran (urut kedatangan) sampai unggahan lain selesai; bila terlalu lama,
// ditolak dengan pesan server sibuk. Dengan begitu banyak unggahan bersamaan tidak membuat server
// kehabisan memori.
package upload

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"office.backend/internal/platform/apperr"
)

// ErrBusy: jatah unggahan masih penuh setelah batas waktu menunggu (503).
var ErrBusy = &apperr.UnavailableError{Message: "Server sedang memproses banyak unggahan. Coba lagi beberapa saat."}

// Limiter adalah antrean unggahan dengan jatah memori size byte.
type Limiter struct {
	mu      sync.Mutex
	size    int64
	used    int64
	waiters list.List // *waiter, urut kedatangan
	wait    time.Duration
}

type waiter struct {
	n     int64
	ready chan struct{} // ditutup saat jatahnya diberikan
}

// NewLimiter membuat antrean dengan jatah sizeBytes dan batas menunggu wait.
func NewLimiter(sizeBytes int64, wait time.Duration) *Limiter {
	return &Limiter{size: sizeBytes, wait: wait}
}

// Acquire memesan jatah n byte untuk satu unggahan, menunggu giliran bila jatah tidak cukup.
// Mengembalikan release yang wajib dipanggil saat unggahan selesai (boleh dipanggil lebih dari
// sekali). Gagal dengan ErrBusy setelah menunggu melewati batas, atau ctx.Err() bila pengguna
// membatalkan. Pesanan yang lebih besar dari seluruh jatah dibatasi sebesar jatah (menunggu
// sampai antrean kosong).
func (l *Limiter) Acquire(ctx context.Context, n int64) (release func(), err error) {
	n = max(min(n, l.size), 1)
	l.mu.Lock()
	if l.size-l.used >= n && l.waiters.Len() == 0 {
		l.used += n
		l.mu.Unlock()
		return l.releaser(n), nil
	}
	w := &waiter{n: n, ready: make(chan struct{})}
	elem := l.waiters.PushBack(w)
	log.Printf("Antrean unggahan: menunggu giliran (%s, jatah terpakai %s/%s, %d menunggu)",
		mb(n), mb(l.used), mb(l.size), l.waiters.Len())
	l.mu.Unlock()

	start := time.Now()
	timer := time.NewTimer(l.wait)
	defer timer.Stop()
	select {
	case <-w.ready:
		return l.releaser(n), nil
	case <-timer.C:
		err = ErrBusy
	case <-ctx.Done():
		err = ctx.Err()
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case <-w.ready:
		// Giliran datang bersamaan dengan batas waktu: jatahnya dikembalikan
		l.used -= n
		l.notify()
	default:
		front := l.waiters.Front() == elem
		l.waiters.Remove(elem)
		if front {
			l.notify() // antrean di belakangnya mungkin sudah muat
		}
	}
	if err == ErrBusy {
		log.Printf("Antrean unggahan: ditolak setelah menunggu %s (%s, jatah terpakai %s/%s)",
			time.Since(start).Round(time.Second), mb(n), mb(l.used), mb(l.size))
	}
	return nil, err
}

// releaser mengembalikan jatah n tepat sekali.
func (l *Limiter) releaser(n int64) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			l.mu.Lock()
			l.used -= n
			l.notify()
			l.mu.Unlock()
		})
	}
}

// notify memberi giliran kepada antrean terdepan selama jatahnya muat (urut kedatangan, supaya
// unggahan besar tidak terus didahului unggahan kecil). Dipanggil dengan mu terkunci.
func (l *Limiter) notify() {
	for {
		front := l.waiters.Front()
		if front == nil {
			return
		}
		w := front.Value.(*waiter)
		if l.size-l.used < w.n {
			return
		}
		l.used += w.n
		l.waiters.Remove(front)
		close(w.ready)
	}
}

// mb menulis ukuran byte dalam MB, contoh "25.0 MB".
func mb(n int64) string { return fmt.Sprintf("%.1f MB", float64(n)/(1<<20)) }
