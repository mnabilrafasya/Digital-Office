package db

import (
	"context"
	"errors"
	"fmt"
	"testing"

	mssql "github.com/microsoft/go-mssqldb"
)

func TestIsDeadlock(t *testing.T) {
	deadlock := fmt.Errorf("query: %w", mssql.Error{Number: 1205})
	if !IsDeadlock(deadlock) {
		t.Error("IsDeadlock(1205 terbungkus) = false, want true")
	}
	if IsDeadlock(mssql.Error{Number: 2627}) || IsDeadlock(errors.New("lain")) {
		t.Error("IsDeadlock untuk error lain = true, want false")
	}
	if !IsUniqueViolation(mssql.Error{Number: 2601}) || !IsUniqueViolation(mssql.Error{Number: 2627}) {
		t.Error("IsUniqueViolation(2601/2627) = false, want true")
	}
}

func TestWithRetryOnDeadlock(t *testing.T) {
	ctx := context.Background()

	t.Run("berhasil setelah deadlock", func(t *testing.T) {
		calls := 0
		err := WithRetryOnDeadlock(ctx, 3, func() error {
			calls++
			if calls < 3 {
				return mssql.Error{Number: 1205}
			}
			return nil
		})
		if err != nil || calls != 3 {
			t.Fatalf("err = %v, calls = %d; want nil, 3", err, calls)
		}
	})

	t.Run("menyerah setelah batas", func(t *testing.T) {
		calls := 0
		err := WithRetryOnDeadlock(ctx, 3, func() error {
			calls++
			return mssql.Error{Number: 1205}
		})
		if !IsDeadlock(err) || calls != 3 {
			t.Fatalf("err = %v, calls = %d; want deadlock, 3", err, calls)
		}
	})

	t.Run("error lain tidak diulang", func(t *testing.T) {
		calls := 0
		want := errors.New("validasi")
		err := WithRetryOnDeadlock(ctx, 3, func() error {
			calls++
			return want
		})
		if !errors.Is(err, want) || calls != 1 {
			t.Fatalf("err = %v, calls = %d; want validasi, 1", err, calls)
		}
	})
}
