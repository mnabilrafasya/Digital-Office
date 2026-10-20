package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// LetterRepository berisi query tabel letters beserta peserta dan penerimanya.
type LetterRepository struct {
	q db.DBTX
}

// NewLetterRepository membuat LetterRepository di atas koneksi atau transaksi.
func NewLetterRepository(q db.DBTX) *LetterRepository {
	return &LetterRepository{q: q}
}

// NewLetter adalah data draf baru: pemilik surat ditambah isi draf.
type NewLetter struct {
	CompanyCode  string
	SenderUnitID int64
	DrafterID    int64
	Draft        *model.LetterDraft
}

// Create menyimpan draf baru beserta reviewer/approver dan penerimanya, lalu mengembalikan uuid
// surat. Wajib dipanggil di dalam transaksi.
func (r *LetterRepository) Create(ctx context.Context, n NewLetter) (string, error) {
	const q = `
INSERT INTO letters (company_code, sender_unit_id, drafter_id,
    letter_type_id, template_id, classification_id, confidentiality_level_id, urgency_level_id,
    title, summary, special_note)
OUTPUT inserted.id, CONVERT(char(36), inserted.uuid)
VALUES (@companyCode, @senderUnitId, @drafterId,
    @letterTypeId, @templateId, @classificationId, @confidentialityId, @urgencyId,
    @title, @summary, @specialNote)`

	args := append(draftArgs(n.Draft),
		sql.Named("companyCode", n.CompanyCode),
		sql.Named("senderUnitId", n.SenderUnitID),
		sql.Named("drafterId", n.DrafterID),
	)
	var (
		id   int64
		uuid string
	)
	if err := r.q.QueryRowContext(ctx, q, args...).Scan(&id, &uuid); err != nil {
		return "", fmt.Errorf("membuat draf surat: %w", err)
	}
	if err := r.saveParts(ctx, id, n.Draft); err != nil {
		return "", err
	}
	return strings.ToLower(uuid), nil
}

// LockByUUID membaca surat milik companyCode sambil mengunci barisnya (UPDLOCK) sampai transaksi
// selesai, sehingga perubahan lain pada surat yang sama menunggu. Wajib di dalam transaksi.
func (r *LetterRepository) LockByUUID(ctx context.Context, companyCode, uuid string) (*model.LetterLock, error) {
	const q = `
SELECT id, drafter_id, status, row_ver
FROM letters WITH (UPDLOCK, ROWLOCK)
WHERE uuid = @uuid AND company_code = @companyCode AND deleted_at IS NULL`

	var l model.LetterLock
	err := r.q.QueryRowContext(ctx, q, sql.Named("uuid", uuid), sql.Named("companyCode", companyCode)).
		Scan(&l.ID, &l.DrafterID, &l.Status, &l.RowVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("mengunci surat: %w", err)
	}
	return &l, nil
}

// IsParticipant mengecek apakah user adalah reviewer/approver aktif di surat.
func (r *LetterRepository) IsParticipant(ctx context.Context, letterID, userID int64) (bool, error) {
	const q = `
SELECT CASE WHEN EXISTS (
    SELECT 1 FROM letter_participants
    WHERE letter_id = @letterId AND user_id = @userId AND deleted_at IS NULL
) THEN 1 ELSE 0 END`
	var ok bool
	if err := r.q.QueryRowContext(ctx, q, sql.Named("letterId", letterID), sql.Named("userId", userID)).Scan(&ok); err != nil {
		return false, fmt.Errorf("memeriksa peserta surat: %w", err)
	}
	return ok, nil
}

// UpdateDraft mengganti seluruh isi draf (termasuk peserta dan penerima). updated_at selalu
// diperbarui sehingga ROWVERSION (ETag) pasti berubah. Wajib dipanggil di dalam transaksi.
func (r *LetterRepository) UpdateDraft(ctx context.Context, letterID int64, d *model.LetterDraft) error {
	const q = `
UPDATE letters
SET letter_type_id = @letterTypeId, template_id = @templateId,
    classification_id = @classificationId, confidentiality_level_id = @confidentialityId,
    urgency_level_id = @urgencyId, title = @title, summary = @summary, special_note = @specialNote,
    updated_at = SYSDATETIMEOFFSET()
WHERE id = @id`

	if _, err := r.q.ExecContext(ctx, q, append(draftArgs(d), sql.Named("id", letterID))...); err != nil {
		return fmt.Errorf("menyimpan draf surat: %w", err)
	}
	return r.saveParts(ctx, letterID, d)
}

func (r *LetterRepository) saveParts(ctx context.Context, letterID int64, d *model.LetterDraft) error {
	if err := r.saveParticipants(ctx, letterID, d); err != nil {
		return err
	}
	return r.saveRecipients(ctx, letterID, d.Recipients)
}

func draftArgs(d *model.LetterDraft) []any {
	return []any{
		sql.Named("letterTypeId", d.LetterTypeID),
		sql.Named("templateId", d.TemplateID),
		sql.Named("classificationId", d.ClassificationID),
		sql.Named("confidentialityId", d.ConfidentialityID),
		sql.Named("urgencyId", d.UrgencyID),
		sql.Named("title", d.Title),
		sql.Named("summary", d.Summary),
		sql.Named("specialNote", d.SpecialNote),
	}
}
