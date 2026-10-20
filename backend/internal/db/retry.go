package db

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	mssql "github.com/microsoft/go-mssqldb"
)

// Nomor error SQL Server yang perlu dikenali aplikasi
const (
	errDeadlock          = 1205 // transaksi dipilih sebagai korban deadlock
	errUniqueIndex       = 2601 // duplikat pada unique index
	errUniqueConstraint  = 2627 // duplikat pada UNIQUE/PRIMARY KEY constraint
	maxDeadlockBackoffMs = 120
)

// IsDeadlock mengembalikan true bila err berasal dari deadlock SQL Server (error 1205).
func IsDeadlock(err error) bool {
	return sqlErrorNumber(err) == errDeadlock
}

// IsUniqueViolation mengembalikan true bila err berasal dari pelanggaran unique index/constraint.
func IsUniqueViolation(err error) bool {
	n := sqlErrorNumber(err)
	return n == errUniqueIndex || n == errUniqueConstraint
}

func sqlErrorNumber(err error) int32 {
	var e mssql.Error
	if errors.As(err, &e) {
		return e.Number
	}
	return 0
}

// WithRetryOnDeadlock menjalankan fn dan mengulanginya (maksimal attempts kali) bila gagal
// karena deadlock, dengan jeda acak kecil yang makin panjang. Error lain langsung dikembalikan.
func WithRetryOnDeadlock(ctx context.Context, attempts int, fn func() error) error {
	if attempts < 1 {
		attempts = 1
	}
	var err error
	for i := 1; i <= attempts; i++ {
		if err = fn(); err == nil || !IsDeadlock(err) || i == attempts {
			return err
		}
		delay := time.Duration(10*i+rand.IntN(maxDeadlockBackoffMs)) * time.Millisecond
		select {
		case <-ctx.Done():
			return errors.Join(err, ctx.Err())
		case <-time.After(delay):
		}
	}
	return err
}
