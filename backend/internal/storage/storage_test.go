package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestKeys(t *testing.T) {
	if StagingKey("o1") != "staging/o1" ||
		AttachmentKey("l1", "o1") != "letters/l1/attachments/o1" ||
		VersionKey("l1", "o1") != "letters/l1/versions/o1" {
		t.Error("format object key salah")
	}
}

func TestContentDisposition(t *testing.T) {
	got := ContentDisposition(`Laporan "Q3" Ä.pdf`)
	want := `attachment; filename="Laporan _Q3_ _.pdf"; filename*=UTF-8''Laporan%20%22Q3%22%20%C3%84.pdf`
	if got != want {
		t.Errorf("ContentDisposition()\n got  %s\n want %s", got, want)
	}
}

func TestCapacity(t *testing.T) {
	now := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	calls := 0
	free := int64(2000)
	c := newCapacity(func(context.Context) (int64, error) { calls++; return free, nil }, 1000, func() time.Time { return now })
	ctx := context.Background()

	if !c.HasRoom(ctx, 1000) {
		t.Error("2000 bebas, batas 1000 + file 1000: harus cukup")
	}
	if c.HasRoom(ctx, 1001) {
		t.Error("2000 bebas, batas 1000 + file 1001: harus penuh")
	}
	if calls != 1 {
		t.Errorf("MinIO ditanya %d kali dalam 60 detik, want 1 (cache)", calls)
	}

	now = now.Add(61 * time.Second)
	free = 0
	if c.HasRoom(ctx, 1) || calls != 2 {
		t.Errorf("setelah cache habis harus bertanya ulang: calls=%d", calls)
	}

	// Gagal mengambil info: fail-open
	failing := newCapacity(func(context.Context) (int64, error) { return 0, errors.New("admin API ditolak") }, 1000, time.Now)
	if !failing.HasRoom(ctx, 1<<40) {
		t.Error("info kapasitas gagal diambil: unggahan harus tetap diizinkan")
	}
}
