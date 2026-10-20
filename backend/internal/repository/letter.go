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

// LetterRepository berisi query tabel letters. Semua query menyaring company_code dan deleted_at.
type LetterRepository struct {
	q db.DBTX
}

// NewLetterRepository membuat LetterRepository di atas koneksi atau transaksi.
func NewLetterRepository(q db.DBTX) *LetterRepository {
	return &LetterRepository{q: q}
}

const letterSelect = `
SELECT l.id, LOWER(CONVERT(VARCHAR(36), l.uuid)), l.company_code, l.letter_number,
       l.template_id, l.letter_type_id, l.classification_id, l.confidentiality_level_id, l.urgency_level_id,
       l.title, l.summary, l.special_note, l.attachments,
       l.status, l.sender_unit_id, l.sender_unit_name, l.drafter_id, l.current_version_number,
       l.current_assignee_id, l.forwarded_by_id, l.returned_by_reviewer_id, l.returned_by_approver_id,
       l.approved_at, l.row_ver, l.created_at, l.updated_at
FROM letters l`

const letterByUUID = `
WHERE l.company_code = @companyCode AND l.uuid = @uuid AND l.deleted_at IS NULL`

// Get mengambil surat berdasarkan uuid di perusahaan tertentu (tanpa kunci).
func (r *LetterRepository) Get(ctx context.Context, companyCode, uuid string) (*model.Letter, error) {
	return r.get(ctx, letterSelect+letterByUUID, companyCode, uuid)
}

// GetForUpdate mengambil surat sambil mengunci barisnya (UPDLOCK, ROWLOCK) sampai transaksi selesai.
// Wajib dipanggil di dalam transaksi, sebelum kunci lain (misalnya penomoran).
func (r *LetterRepository) GetForUpdate(ctx context.Context, companyCode, uuid string) (*model.Letter, error) {
	return r.get(ctx, letterSelect+" WITH (UPDLOCK, ROWLOCK)"+letterByUUID, companyCode, uuid)
}

func (r *LetterRepository) get(ctx context.Context, q, companyCode, uuid string) (*model.Letter, error) {
	var l model.Letter
	c := &l.Content
	err := r.q.QueryRowContext(ctx, q, sql.Named("companyCode", companyCode), sql.Named("uuid", uuid)).Scan(
		&l.ID, &l.UUID, &l.CompanyCode, &l.LetterNumber,
		&c.TemplateID, &c.LetterTypeID, &c.ClassificationID, &c.ConfidentialityID, &c.UrgencyID,
		&c.Title, &c.Summary, &c.SpecialNote, &l.AttachmentsJSON,
		&l.Status, &l.SenderUnitID, &l.SenderUnitName, &l.DrafterID, &l.CurrentVersion,
		&l.AssigneeID, &l.ForwardedByID, &l.ReturnedByReviewerID, &l.ReturnedByApproverID,
		&l.ApprovedAt, &l.RowVersion, &l.CreatedAt, &l.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("mengambil surat: %w", err)
	}
	return &l, nil
}

// Create menyimpan draf baru dan mengembalikan id serta uuid-nya.
func (r *LetterRepository) Create(ctx context.Context, nl model.NewLetter) (int64, string, error) {
	const q = `
INSERT INTO letters (company_code, sender_unit_id, drafter_id,
                     template_id, letter_type_id, classification_id, confidentiality_level_id, urgency_level_id,
                     title, summary, special_note)
OUTPUT inserted.id, LOWER(CONVERT(VARCHAR(36), inserted.uuid))
VALUES (@companyCode, @senderUnitId, @drafterId,
        @templateId, @letterTypeId, @classificationId, @confidentialityId, @urgencyId,
        @title, @summary, @specialNote)`

	args := append([]any{
		sql.Named("companyCode", nl.CompanyCode),
		sql.Named("senderUnitId", nl.SenderUnitID),
		sql.Named("drafterId", nl.DrafterID),
	}, contentArgs(nl.Content)...)

	var (
		id   int64
		uuid string
	)
	if err := r.q.QueryRowContext(ctx, q, args...).Scan(&id, &uuid); err != nil {
		return 0, "", fmt.Errorf("menyimpan draf: %w", err)
	}
	return id, uuid, nil
}

// UpdateContent mengganti seluruh isian surat yang bisa diedit drafter.
func (r *LetterRepository) UpdateContent(ctx context.Context, id int64, c model.LetterContent) error {
	const q = `
UPDATE letters
SET template_id = @templateId, letter_type_id = @letterTypeId, classification_id = @classificationId,
    confidentiality_level_id = @confidentialityId, urgency_level_id = @urgencyId,
    title = @title, summary = @summary, special_note = @specialNote,
    updated_at = SYSDATETIMEOFFSET()
WHERE id = @id`

	if _, err := r.q.ExecContext(ctx, q, append(contentArgs(c), sql.Named("id", id))...); err != nil {
		return fmt.Errorf("menyimpan isi surat: %w", err)
	}
	return nil
}

func contentArgs(c model.LetterContent) []any {
	return []any{
		sql.Named("templateId", c.TemplateID),
		sql.Named("letterTypeId", c.LetterTypeID),
		sql.Named("classificationId", c.ClassificationID),
		sql.Named("confidentialityId", c.ConfidentialityID),
		sql.Named("urgencyId", c.UrgencyID),
		sql.Named("title", c.Title),
		sql.Named("summary", c.Summary),
		sql.Named("specialNote", c.SpecialNote),
	}
}

// SoftDelete menandai surat sebagai terhapus.
func (r *LetterRepository) SoftDelete(ctx context.Context, id int64) error {
	const q = `
UPDATE letters SET deleted_at = SYSDATETIMEOFFSET(), updated_at = SYSDATETIMEOFFSET()
WHERE id = @id AND deleted_at IS NULL`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", id)); err != nil {
		return fmt.Errorf("menghapus draf: %w", err)
	}
	return nil
}

// UpdateAttachments mengganti isi JSON kolom attachments (nil = NULL).
func (r *LetterRepository) UpdateAttachments(ctx context.Context, id int64, attachmentsJSON *string) error {
	const q = `
UPDATE letters SET attachments = @attachments, updated_at = SYSDATETIMEOFFSET()
WHERE id = @id`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("attachments", attachmentsJSON), sql.Named("id", id)); err != nil {
		return fmt.Errorf("menyimpan lampiran: %w", err)
	}
	return nil
}

// IncrementVersion menaikkan current_version_number dan mengembalikan nomor versi baru.
func (r *LetterRepository) IncrementVersion(ctx context.Context, id int64) (int, error) {
	const q = `
UPDATE letters
SET current_version_number = current_version_number + 1, updated_at = SYSDATETIMEOFFSET()
OUTPUT inserted.current_version_number
WHERE id = @id`

	var n int
	if err := r.q.QueryRowContext(ctx, q, sql.Named("id", id)).Scan(&n); err != nil {
		return 0, fmt.Errorf("menaikkan nomor versi: %w", err)
	}
	return n, nil
}

// ApplyTransition menerapkan perubahan status surat hasil alur persetujuan.
func (r *LetterRepository) ApplyTransition(ctx context.Context, id int64, t model.LetterTransition) error {
	sets := []string{
		"status = @status",
		"current_assignee_id = @assigneeId",
		"updated_at = SYSDATETIMEOFFSET()",
	}
	args := []any{
		sql.Named("id", id),
		sql.Named("status", t.Status),
		sql.Named("assigneeId", t.AssigneeID),
	}
	setID := func(change model.IDChange, column, param string) {
		if change.Set {
			sets = append(sets, column+" = @"+param)
			args = append(args, sql.Named(param, change.Value))
		}
	}
	setID(t.ForwardedBy, "forwarded_by_id", "forwardedBy")
	setID(t.ReturnedByReviewer, "returned_by_reviewer_id", "returnedByReviewer")
	setID(t.ReturnedByApprover, "returned_by_approver_id", "returnedByApprover")

	if t.SenderUnitName != nil {
		sets = append(sets, "sender_unit_name = @senderUnitName")
		args = append(args, sql.Named("senderUnitName", *t.SenderUnitName))
	}
	if s := t.Sign; s != nil {
		sets = append(sets,
			"letter_number = @letterNumber", "number_seq = @numberSeq", "number_year = @numberYear",
			"approved_at = SYSDATETIMEOFFSET()", "signed_by_id = @signedBy")
		args = append(args,
			sql.Named("letterNumber", s.LetterNumber), sql.Named("numberSeq", s.NumberSeq),
			sql.Named("numberYear", s.NumberYear), sql.Named("signedBy", s.SignedByID))
	}

	q := "UPDATE letters SET " + strings.Join(sets, ", ") + " WHERE id = @id"
	if _, err := r.q.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("mengubah status surat: %w", err)
	}
	return nil
}
