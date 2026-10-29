package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// Request yang sedang berjalan saat server dimatikan harus tetap selesai (graceful shutdown).
func TestRunGracefulShutdown(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	started := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		time.Sleep(300 * time.Millisecond) // pekerjaan yang masih berjalan saat shutdown
		_, _ = io.WriteString(w, "selesai")
	})

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- Run(ctx, addr, handler) }()

	// Tunggu server siap menerima koneksi
	var resp *http.Response
	reqErr := make(chan error, 1)
	go func() {
		for i := 0; i < 50; i++ {
			resp, err = http.Get("http://" + addr)
			if err == nil {
				reqErr <- nil
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		reqErr <- err
	}()

	<-started
	cancel() // matikan server saat request masih diproses

	if err := <-reqErr; err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "selesai" {
		t.Fatalf("response = %d %q, want 200 \"selesai\"", resp.StatusCode, body)
	}
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run() = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run tidak berhenti setelah context dibatalkan")
	}
}
