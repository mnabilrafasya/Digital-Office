package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// shutdownTimeout adalah batas waktu menunggu request yang sedang berjalan saat server dimatikan.
const shutdownTimeout = 30 * time.Second

// SignalContext mengembalikan context yang selesai saat proses menerima SIGINT (Ctrl+C) atau SIGTERM.
func SignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

// Run menjalankan HTTP server sampai ctx selesai, lalu mematikannya dengan rapi: berhenti menerima
// koneksi baru dan menunggu request yang sedang berjalan maksimal 30 detik.
func Run(ctx context.Context, addr string, handler http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("Server berjalan di http://localhost%s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server berhenti: %w", err)
	case <-ctx.Done():
	}

	log.Println("Mematikan server...")
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("mematikan server: %w", err)
	}
	log.Println("Server berhenti dengan rapi")
	return nil
}
