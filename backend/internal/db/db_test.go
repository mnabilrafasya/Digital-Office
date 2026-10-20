package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	mssql "github.com/microsoft/go-mssqldb"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/config"
)

func TestBuildDSN(t *testing.T) {
	cfg := &config.Config{DBUser: "sa", DBPassword: "p#ss", DBHost: "localhost", DBPort: "1433", DBTrustServerCert: false}
	dsn := BuildDSN(cfg, "master")
	for _, want := range []string{"sqlserver://sa:p%23ss@localhost:1433", "database=master", "TrustServerCertificate=false", "encrypt=true"} {
		if !strings.Contains(dsn, want) {
			t.Errorf("DSN tidak berisi %q", want)
		}
	}
}

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
	tests := []struct {
		name      string
		fails     int   // jumlah kegagalan sebelum berhasil
		failErr   error // error yang dikembalikan saat gagal
		wantCalls int
		wantErr   bool
	}{
		{"berhasil setelah 2 deadlock", 2, mssql.Error{Number: 1205}, 3, false},
		{"menyerah setelah 3 percobaan", 99, mssql.Error{Number: 1205}, 3, true},
		{"error lain tidak diulang", 99, errors.New("validasi"), 1, true},
	}
	for _, tt := range tests {
		calls := 0
		err := WithRetryOnDeadlock(ctx, 3, func() error {
			calls++
			if calls <= tt.fails {
				return tt.failErr
			}
			return nil
		})
		if (err != nil) != tt.wantErr || calls != tt.wantCalls {
			t.Errorf("%s: err = %v, calls = %d; want err=%v, calls=%d", tt.name, err, calls, tt.wantErr, tt.wantCalls)
		}
	}
}
