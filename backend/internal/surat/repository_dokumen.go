package surat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"office.backend/internal/platform/db"
)

// AddVersion menyimpan versi naskah baru: nomor versi surat dinaikkan, baris letter_versions
// ditambahkan, dan untuk versi DRAFT aksi SAVE dicatat di riwayat (PDF FINAL dibuat sistem, bukan
// disimpan pengguna). Mengembalikan nomor versi dan ROWVERSION surat terbaru. Wajib dipanggil di
// dalam transaksi, setelah LockByUUID.
func (r *Repository) AddVersion(ctx context.Context, v NewVersion) (int, []byte, error) {
	const bump = `
UPDATE letters
SET current_version_number = current_version_number + 1, updated_at = SYSDATETIMEOFFSET()
OUTPUT inserted.current_version_number, inserted.row_ver
WHERE id = @id AND deleted_at IS NULL`
	var (
		number int
		rowVer []byte
	)
	if err := r.q.QueryRowContext(ctx, bump, sql.Named("id", v.LetterID)).Scan(&number, &rowVer); err != nil {
		return 0, nil, fmt.Errorf("menaikkan versi naskah: %w", err)
	}

	const insert = `
INSERT INTO letter_versions (letter_id, version_number, version_type, object_key, file_name,
                             file_extension, file_size, checksum, created_by, created_by_role)
VALUES (@letterId, @number, @type, @objectKey, @fileName, @ext, @size, @checksum, @createdBy, @role)`
	if _, err := r.q.ExecContext(ctx, insert,
		sql.Named("letterId", v.LetterID),
		sql.Named("number", number),
		sql.Named("type", v.Type),
		sql.Named("objectKey", v.ObjectKey),
		sql.Named("fileName", v.FileName),
		sql.Named("ext", v.FileExtension),
		sql.Named("size", v.FileSize),
		sql.Named("checksum", v.Checksum),
		sql.Named("createdBy", v.CreatedBy),
		sql.Named("role", v.CreatedByRole),
	); err != nil {
		return 0, nil, fmt.Errorf("menyimpan versi naskah: %w", err)
	}
	if v.Type != VersionDraft {
		return number, rowVer, nil
	}

	const action = `
INSERT INTO letter_actions (letter_id, actor_id, actor_role, action, from_status, to_status, version_number)
VALUES (@letterId, @actorId, @role, 'SAVE', @status, @status, @number)`
	if _, err := r.q.ExecContext(ctx, action,
		sql.Named("letterId", v.LetterID),
		sql.Named("actorId", v.CreatedBy),
		sql.Named("role", v.CreatedByRole),
		sql.Named("status", v.Status),
		sql.Named("number", number),
	); err != nil {
		return 0, nil, fmt.Errorf("mencatat riwayat simpan naskah: %w", err)
	}
	return number, rowVer, nil
}

const versionColumns = `
SELECT v.version_number, v.version_type, v.file_name, v.file_extension, v.file_size, v.checksum,
       u.id, u.name, p.name, v.created_by_role, v.created_at, CONVERT(char(36), v.object_key)
FROM letter_versions v
JOIN users u     ON u.id = v.created_by
JOIN positions p ON p.id = u.position_id`

// Versions mengambil semua versi naskah surat, terbaru lebih dulu.
func (r *Repository) Versions(ctx context.Context, letterID int64) ([]DocumentVersion, error) {
	rows, err := r.q.QueryContext(ctx, versionColumns+`
WHERE v.letter_id = @letterId AND v.deleted_at IS NULL
ORDER BY v.version_number DESC`, sql.Named("letterId", letterID))
	if err != nil {
		return nil, fmt.Errorf("mengambil versi naskah: %w", err)
	}
	defer rows.Close()

	out := []DocumentVersion{}
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca versi naskah: %w", err)
	}
	return out, nil
}

// FindVersion mengambil satu versi naskah berdasarkan nomornya.
func (r *Repository) FindVersion(ctx context.Context, letterID int64, number int) (*DocumentVersion, error) {
	return r.findVersion(ctx, versionColumns+`
WHERE v.letter_id = @letterId AND v.version_number = @number AND v.deleted_at IS NULL`,
		sql.Named("letterId", letterID), sql.Named("number", number))
}

// LatestVersion mengambil versi terbaru dengan jenis tertentu (DRAFT atau FINAL).
func (r *Repository) LatestVersion(ctx context.Context, letterID int64, versionType string) (*DocumentVersion, error) {
	return r.findVersion(ctx, versionColumns+`
WHERE v.letter_id = @letterId AND v.version_type = @type AND v.deleted_at IS NULL
ORDER BY v.version_number DESC
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY`,
		sql.Named("letterId", letterID), sql.Named("type", versionType))
}

func (r *Repository) findVersion(ctx context.Context, q string, args ...any) (*DocumentVersion, error) {
	v, err := scanVersion(r.q.QueryRowContext(ctx, q, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	return v, err
}

func scanVersion(row interface{ Scan(...any) error }) (*DocumentVersion, error) {
	var v DocumentVersion
	err := row.Scan(&v.VersionNumber, &v.Type, &v.FileName, &v.FileExtension, &v.FileSize, &v.Checksum,
		&v.CreatedBy.ID, &v.CreatedBy.Name, &v.CreatedBy.PositionName, &v.CreatedByRole, &v.CreatedAt, &v.ObjectKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("membaca versi naskah: %w", err)
	}
	// SQL Server menulis uuid huruf besar; nama objek di MinIO memakai huruf kecil
	v.ObjectKey = strings.ToLower(v.ObjectKey)
	return &v, nil
}
