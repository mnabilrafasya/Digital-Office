package upload

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestAcquireWithinBudget(t *testing.T) {
	l := NewLimiter(100, time.Second)
	r1, err1 := l.Acquire(context.Background(), 60)
	r2, err2 := l.Acquire(context.Background(), 40)
	if err1 != nil || err2 != nil || l.used != 100 {
		t.Fatalf("dua unggahan dalam jatah harus langsung jalan: %v %v used=%d", err1, err2, l.used)
	}
	r1()
	r1() // aman dipanggil dua kali
	r2()
	if l.used != 0 {
		t.Errorf("jatah tidak kembali: used=%d", l.used)
	}
}

func TestAcquireWaitsForTurn(t *testing.T) {
	l := NewLimiter(100, 2*time.Second)
	r1, _ := l.Acquire(context.Background(), 80)

	got := make(chan error, 1)
	go func() {
		r, err := l.Acquire(context.Background(), 50)
		if err == nil {
			defer r()
		}
		got <- err
	}()
	select {
	case <-got:
		t.Fatal("unggahan kedua harus menunggu selama jatah penuh")
	case <-time.After(100 * time.Millisecond):
	}
	r1() // unggahan pertama selesai: giliran berpindah
	select {
	case err := <-got:
		if err != nil {
			t.Fatalf("setelah jatah kembali: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("unggahan kedua tidak mendapat giliran")
	}
}

func TestAcquireTimeoutAndCancel(t *testing.T) {
	l := NewLimiter(100, 150*time.Millisecond)
	r1, _ := l.Acquire(context.Background(), 100)
	defer r1()

	start := time.Now()
	if _, err := l.Acquire(context.Background(), 10); !errors.Is(err, ErrBusy) || time.Since(start) < 150*time.Millisecond {
		t.Errorf("lewat batas tunggu: err=%v setelah %v, want ErrBusy", err, time.Since(start))
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if _, err := l.Acquire(ctx, 10); !errors.Is(err, context.Canceled) {
		t.Errorf("pengguna membatalkan: err=%v", err)
	}
	if l.waiters.Len() != 0 || l.used != 100 {
		t.Errorf("antrean harus bersih: %d menunggu, used=%d", l.waiters.Len(), l.used)
	}
}

func TestFIFOAndOversize(t *testing.T) {
	l := NewLimiter(100, 2*time.Second)
	r1, _ := l.Acquire(context.Background(), 70)

	order := make(chan int64, 3)
	var wg sync.WaitGroup
	start := func(n int64) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := l.Acquire(context.Background(), n)
			if err != nil {
				t.Errorf("unggahan %d: %v", n, err)
				return
			}
			order <- n
			time.Sleep(20 * time.Millisecond)
			r()
		}()
		time.Sleep(30 * time.Millisecond) // urutan kedatangan pasti
	}
	start(500) // lebih besar dari jatah: dibatasi 100, menunggu antrean kosong
	start(10)  // datang belakangan: tidak boleh mendahului walaupun muat
	r1()
	wg.Wait()
	close(order)
	var got []int64
	for n := range order {
		got = append(got, n)
	}
	if len(got) != 2 || got[0] != 500 || got[1] != 10 {
		t.Errorf("urutan giliran = %v, want [500 10] (urut kedatangan)", got)
	}
	if l.used != 0 || l.waiters.Len() != 0 {
		t.Errorf("jatah tidak kembali penuh: used=%d menunggu=%d", l.used, l.waiters.Len())
	}
}
