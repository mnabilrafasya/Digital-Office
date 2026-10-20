package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// VersionRepository berisi query tabel letter_versions (riwayat dokumen surat).
type VersionRepository struct {
	q db.DBTX
}

// NewVersionRepository membuat VersionRepository di atas koneksi atau transaksi.
func NewVersionRepository(q db.DBTX) *VersionRepository {
	return &VersionRepository{q: q}
}

const versionSelect = `
SELECT v.id, v.letter_id, v.version_number, v.version_type, LOWER(CONVERT(VARCHAR(36), v.object_key)),
       v.file_name, v.file_extension, v.file_size, v.checksum,
       u.id, u.name, v.created_by_role, v.created_at
FROM letter_versions v
JOIN users u ON u.id = v.created_by`

// Insert menyimpan satu versi dokumen (version_type DRAFT).
func (r *VersionRepository) Insert(ctx context.Context, v model.NewVersion) error {
	const q = `
INSERT INTO letter_versions (letter_id, version_number, version_type, object_key, file_name,
                             file_extension, file_size, checksum, created_by, created_by_role)
VALUES (@letterId, @versionNumber, 'DRAFT', @objectKey, @fileName,
        @fileExtension, @fileSize, @checksum, @createdBy, @createdByRole)`

	_, err := r.q.ExecContext(ctx, q,
		sql.Named("letterId", v.LetterID), sql.Named("versionNumber", v.VersionNumber),
		sql.Named("objectKey", v.ObjectKey), sql.Named("fileName", v.FileName),
		sql.Named("fileExtension", v.FileExtension), sql.Named("fileSize", v.FileSize),
		sql.Named("checksum", v.Checksum), sql.Named("createdBy", v.CreatedBy),
		sql.Named("createdByRole", v.CreatedByRole))
	if err != nil {
		return fmt.Errorf("menyimpan versi dokumen: %w", err)
	}
	return nil
}

// List mengambil semua versi dokumen surat, urut dari yang terbaru.
func (r *VersionRepository) List(ctx context.Context, letterID int64) ([]model.Version, error) {
	rows, err := r.q.QueryContext(ctx, versionSelect+`
WHERE v.letter_id = @letterId AND v.deleted_at IS NULL
ORDER BY v.version_number DESC`, sql.Named("letterId", letterID))
	if err != nil {
		return nil, fmt.Errorf("mengambil versi dokumen: %w", err)
	}
	defer rows.Close()

	out := []model.Version{}
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca versi dokumen: %w", err)
	}
	return out, nil
}

// GetByNumber mengambil satu versi berdasarkan nomornya.
func (r *VersionRepository) GetByNumber(ctx context.Context, letterID int64, number int) (*model.Version, error) {
	return r.getOne(ctx, versionSelect+`
WHERE v.letter_id = @letterId AND v.version_number = @n AND v.deleted_at IS NULL`,
		sql.Named("letterId", letterID), sql.Named("n", number))
}

// GetByObjectKey mengambil versi berdasarkan uuid object (untuk konfirmasi unggah yang idempoten).
func (r *VersionRepository) GetByObjectKey(ctx context.Context, letterID int64, objectKey string) (*model.Version, error) {
	return r.getOne(ctx, versionSelect+`
WHERE v.letter_id = @letterId AND v.object_key = @objectKey AND v.deleted_at IS NULL`,
		sql.Named("letterId", letterID), sql.Named("objectKey", objectKey))
}

// ObjectKeys mengambil uuid object semua versi dokumen surat.
func (r *VersionRepository) ObjectKeys(ctx context.Context, letterID int64) ([]string, error) {
	const q = `
SELECT LOWER(CONVERT(VARCHAR(36), object_key)) FROM letter_versions
WHERE letter_id = @letterId AND deleted_at IS NULL`
	rows, err := r.q.QueryContext(ctx, q, sql.Named("letterId", letterID))
	if err != nil {
		return nil, fmt.Errorf("mengambil object versi: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("membaca object versi: %w", err)
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *VersionRepository) getOne(ctx context.Context, q string, args ...any) (*model.Version, error) {
	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil versi dokumen: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("mengambil versi dokumen: %w", err)
		}
		return nil, ErrNotFound
	}
	return scanVersion(rows)
}

func scanVersion(rows *sql.Rows) (*model.Version, error) {
	var v model.Version
	if err := rows.Scan(&v.ID, &v.LetterID, &v.VersionNumber, &v.VersionType, &v.ObjectKey,
		&v.FileName, &v.FileExtension, &v.FileSize, &v.Checksum,
		&v.CreatedBy.ID, &v.CreatedBy.Name, &v.CreatedByRole, &v.CreatedAt); err != nil {
		return nil, fmt.Errorf("membaca versi dokumen: %w", err)
	}
	return &v, nil
}
