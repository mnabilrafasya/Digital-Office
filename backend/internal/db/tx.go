package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// DBTX adalah kemampuan query yang dimiliki *sql.DB maupun *sql.Tx, sehingga repository
// bisa dipakai di dalam atau di luar transaksi.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// WithTx menjalankan fn di dalam satu transaksi. Transaksi di-commit bila fn berhasil dan
// di-rollback bila fn mengembalikan error atau panic (panic diteruskan setelah rollback).
func WithTx(ctx context.Context, conn *sql.DB, fn func(tx DBTX) error) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("memulai transaksi: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback: %w", rbErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaksi: %w", err)
	}
	return nil
}

// TxRunner menjalankan fungsi di dalam transaksi database. Service bergantung pada interface
// ini (bukan *sql.DB) supaya bisa diuji tanpa database.
type TxRunner interface {
	InTx(ctx context.Context, fn func(q DBTX) error) error
}

// Runner adalah TxRunner di atas pool *sql.DB, dengan pengulangan otomatis saat deadlock.
type Runner struct {
	conn     *sql.DB
	attempts int
}

// NewRunner membuat Runner. attempts adalah jumlah percobaan maksimal saat terjadi deadlock.
func NewRunner(conn *sql.DB, attempts int) *Runner {
	return &Runner{conn: conn, attempts: attempts}
}

// InTx menjalankan fn dalam transaksi; seluruh transaksi diulang bila SQL Server memilihnya
// sebagai korban deadlock.
func (r *Runner) InTx(ctx context.Context, fn func(q DBTX) error) error {
	return WithRetryOnDeadlock(ctx, r.attempts, func() error {
		return WithTx(ctx, r.conn, fn)
	})
}
