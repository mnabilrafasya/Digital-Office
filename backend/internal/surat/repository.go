package surat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"office.backend/internal/platform/db"
)

// Repository berisi query tabel letters beserta peserta dan penerimanya.
type Repository struct {
	q db.DBTX
}

// NewRepository membuat Repository di atas koneksi atau transaksi.
func NewRepository(q db.DBTX) *Repository {
	return &Repository{q: q}
}

// NewLetter adalah data draf baru: pemilik surat ditambah isi draf.
type NewLetter struct {
	CompanyCode  string
	SenderUnitID int64
	DrafterID    int64
	Draft        *LetterDraft
}

// Create menyimpan draf baru beserta reviewer/approver dan penerimanya, lalu mengembalikan uuid
// surat. Wajib dipanggil di dalam transaksi.
func (r *Repository) Create(ctx context.Context, n NewLetter) (string, error) {
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

// letterLockColumns adalah kolom surat yang dibutuhkan untuk pengecekan akses dan alur persetujuan.
const letterLockColumns = `
SELECT id, drafter_id, ISNULL(current_assignee_id, 0), ISNULL(forwarded_by_id, 0), status, row_ver`

// LockByUUID membaca surat milik companyCode sambil mengunci barisnya (UPDLOCK) sampai transaksi
// selesai, sehingga perubahan lain pada surat yang sama menunggu. Wajib di dalam transaksi.
func (r *Repository) LockByUUID(ctx context.Context, companyCode, uuid string) (*LetterLock, error) {
	return r.findLock(ctx, letterLockColumns+`
FROM letters WITH (UPDLOCK, ROWLOCK)
WHERE uuid = @uuid AND company_code = @companyCode AND deleted_at IS NULL`, companyCode, uuid)
}

// FindByUUID sama dengan LockByUUID tetapi tanpa mengunci (untuk pengecekan akses baca).
func (r *Repository) FindByUUID(ctx context.Context, companyCode, uuid string) (*LetterLock, error) {
	return r.findLock(ctx, letterLockColumns+`
FROM letters
WHERE uuid = @uuid AND company_code = @companyCode AND deleted_at IS NULL`, companyCode, uuid)
}

func (r *Repository) findLock(ctx context.Context, q, companyCode, uuid string) (*LetterLock, error) {
	var l LetterLock
	err := r.q.QueryRowContext(ctx, q, sql.Named("uuid", uuid), sql.Named("companyCode", companyCode)).
		Scan(&l.ID, &l.DrafterID, &l.AssigneeID, &l.ForwardedByID, &l.Status, &l.RowVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("membaca surat: %w", err)
	}
	return &l, nil
}

// UpdateDraft mengganti seluruh isi draf (termasuk peserta dan penerima). updated_at selalu
// diperbarui sehingga ROWVERSION (ETag) pasti berubah. Wajib dipanggil di dalam transaksi.
func (r *Repository) UpdateDraft(ctx context.Context, letterID int64, d *LetterDraft) error {
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

// LoadDraft membaca isi draf yang tersimpan (untuk validasi kelengkapan saat submit).
func (r *Repository) LoadDraft(ctx context.Context, letterID int64) (*LetterDraft, error) {
	const q = `
SELECT letter_type_id, template_id, classification_id, confidentiality_level_id, urgency_level_id,
       title, summary, special_note
FROM letters
WHERE id = @id AND deleted_at IS NULL`

	var d LetterDraft
	err := r.q.QueryRowContext(ctx, q, sql.Named("id", letterID)).Scan(
		&d.LetterTypeID, &d.TemplateID, &d.ClassificationID, &d.ConfidentialityID, &d.UrgencyID,
		&d.Title, &d.Summary, &d.SpecialNote)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("membaca draf surat: %w", err)
	}

	ps, err := r.loadParticipants(ctx, letterID)
	if err != nil {
		return nil, err
	}
	parts := draftParticipants(ps)
	d.ReviewerIDs, d.ApproverID = parts.ReviewerIDs, parts.ApproverID

	rs, err := r.loadRecipients(ctx, letterID)
	if err != nil {
		return nil, err
	}
	for _, rc := range rs {
		d.Recipients = append(d.Recipients, RecipientRef{Type: rc.Type, UnitID: rc.UnitID, AliasLevelID: rc.AliasID})
	}
	return &d, nil
}

func (r *Repository) saveParts(ctx context.Context, letterID int64, d *LetterDraft) error {
	if err := r.saveParticipants(ctx, letterID, d); err != nil {
		return err
	}
	return r.saveRecipients(ctx, letterID, d.Recipients)
}

func draftArgs(d *LetterDraft) []any {
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

// TemplateLetterType mengembalikan jenis surat template aktif milik companyCode
// (nil = template berlaku untuk semua jenis). db.ErrNotFound bila template tidak ada/tidak aktif.
func (r *Repository) TemplateLetterType(ctx context.Context, companyCode string, templateID int64) (*int64, error) {
	const q = `
SELECT letter_type_id FROM letter_templates
WHERE id = @id AND company_code = @companyCode AND is_active = 1 AND deleted_at IS NULL`
	var typeID *int64
	err := r.q.QueryRowContext(ctx, q, sql.Named("id", templateID), sql.Named("companyCode", companyCode)).Scan(&typeID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("memeriksa template: %w", err)
	}
	return typeID, nil
}

// execRows menjalankan query dengan @letterId dan satu parameter JSON berisi banyak baris/ID.
func (r *Repository) execRows(ctx context.Context, q string, letterID int64, param string, rows any) error {
	b, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	_, err = r.q.ExecContext(ctx, q, sql.Named("letterId", letterID), sql.Named(param, string(b)))
	return err
}
