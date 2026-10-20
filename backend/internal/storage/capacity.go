package storage

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/minio/madmin-go/v4"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/config"
)

// capacityCacheTTL adalah lama hasil cek kapasitas disimpan sebelum ditanyakan ulang ke MinIO.
const capacityCacheTTL = 60 * time.Second

// freeSpaceFunc mengambil sisa ruang storage dalam byte.
type freeSpaceFunc func(ctx context.Context) (int64, error)

// Capacity memeriksa sisa ruang storage sebelum memberi izin unggah. Bila info kapasitas tidak
// bisa diambil, unggahan tetap diizinkan (fail-open) dan peringatan dicatat di log.
type Capacity struct {
	freeSpace freeSpaceFunc
	minFree   int64
	now       func() time.Time

	mu       sync.Mutex
	cached   int64
	cachedAt time.Time
	cacheOK  bool
}

// NewCapacity membuat pemeriksa kapasitas memakai Admin API MinIO.
func NewCapacity(cfg *config.Config) (*Capacity, error) {
	admin, err := madmin.New(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioUseSSL)
	if err != nil {
		return nil, fmt.Errorf("membuat client admin MinIO: %w", err)
	}
	free := func(ctx context.Context) (int64, error) {
		info, err := admin.StorageInfo(ctx)
		if err != nil {
			return 0, err
		}
		var total uint64
		for _, d := range info.Disks {
			total += d.AvailableSpace
		}
		return int64(total), nil
	}
	return newCapacity(free, cfg.StorageMinFreeBytes(), time.Now), nil
}

func newCapacity(free freeSpaceFunc, minFree int64, now func() time.Time) *Capacity {
	return &Capacity{freeSpace: free, minFree: minFree, now: now}
}

// HasRoom mengembalikan false bila sisa ruang kurang dari batas aman ditambah ukuran file.
func (c *Capacity) HasRoom(ctx context.Context, size int64) bool {
	free, ok := c.free(ctx)
	if !ok {
		return true
	}
	return free >= c.minFree+size
}

func (c *Capacity) free(ctx context.Context) (int64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cacheOK && c.now().Sub(c.cachedAt) < capacityCacheTTL {
		return c.cached, true
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	free, err := c.freeSpace(ctx)
	if err != nil {
		log.Printf("PERINGATAN: kapasitas storage tidak bisa diperiksa, unggahan tetap diizinkan: %v", err)
		c.cacheOK = false
		return 0, false
	}
	c.cached, c.cachedAt, c.cacheOK = free, c.now(), true
	return free, true
}
