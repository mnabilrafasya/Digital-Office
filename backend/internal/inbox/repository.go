package inbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"office.backend/internal/platform/db"
)

// Repository membaca dan menulis inbox (tabel inbox_items) beserta penerusannya (letter_forwards).
type Repository struct {
	q db.DBTX
}

// NewRepository membuat Repository di atas koneksi atau transaksi.
func NewRepository(q db.DBTX) *Repository {
	return &Repository{q: q}
}

// Filter adalah kriteria inbox satu user. Statuses kosong berarti semua status.
type Filter struct {
	CompanyCode string
	UserID      int64
	Statuses    []string
	Search      string // dicari di perihal dan nomor surat
	Offset      int
	Limit       int
}

// where menyusun klausa WHERE dari potongan SQL tetap; nilai filter selalu lewat parameter.
// withStatus=false dipakai untuk menghitung jumlah per tab (semua status).
func (f Filter) where(withStatus bool) (string, []any) {
	var sb strings.Builder
	sb.WriteString(`
WHERE i.user_id = @userId AND i.deleted_at IS NULL
  AND l.company_code = @companyCode AND l.deleted_at IS NULL`)
	args := []any{sql.Named("userId", f.UserID), sql.Named("companyCode", f.CompanyCode)}
	if withStatus && len(f.Statuses) > 0 {
		statuses, _ := json.Marshal(f.Statuses) // []string selalu berhasil di-marshal
		sb.WriteString("\n  AND i.status IN (SELECT [value] FROM OPENJSON(@statuses))")
		args = append(args, sql.Named("statuses", string(statuses)))
	}
	if f.Search != "" {
		sb.WriteString("\n  AND (l.title LIKE @pattern ESCAPE '\\' OR l.letter_number LIKE @pattern ESCAPE '\\')")
		args = append(args, sql.Named("pattern", db.ContainsPattern(f.Search)))
	}
	return sb.String(), args
}

// itemColumns adalah kolom satu item inbox beserta ringkasan surat dan asal penerusannya.
const itemColumns = `
SELECT i.id, i.status, i.created_at, i.read_at,
       i.source_type, i.recipient_type, fw.type, fw.from_user_id, fw.from_user_name, fw.from_position, fw.note,
       CONVERT(char(36), l.uuid), l.letter_number, l.title, l.approved_at,
       lt.id, lt.code, lt.name, cf.id, cf.code, cf.name, ug.id, ug.code, ug.name,
       su.id, su.code, COALESCE(l.sender_unit_name, su.name),
       sb.id, sb.name, sp.name
FROM inbox_items i
JOIN letters l               ON l.id = i.letter_id
JOIN units su                ON su.id = l.sender_unit_id
LEFT JOIN letter_forwards fw ON fw.id = i.forward_id
LEFT JOIN lookups lt         ON lt.id = l.letter_type_id
LEFT JOIN lookups cf         ON cf.id = l.confidentiality_level_id
LEFT JOIN lookups ug         ON ug.id = l.urgency_level_id
LEFT JOIN users sb           ON sb.id = l.signed_by_id
LEFT JOIN positions sp       ON sp.id = sb.position_id`

// List mengambil satu halaman inbox, terbaru masuk lebih dulu.
func (r *Repository) List(ctx context.Context, f Filter) ([]Item, error) {
	where, args := f.where(true)
	q := itemColumns + where + `
ORDER BY i.created_at DESC, i.id DESC
OFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY`
	args = append(args, sql.Named("offset", f.Offset), sql.Named("limit", f.Limit))

	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil inbox: %w", err)
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca inbox: %w", err)
	}
	return items, nil
}

// FindItem mengambil satu item inbox milik userID (db.ErrNotFound bila tidak ada atau milik orang lain).
func (r *Repository) FindItem(ctx context.Context, companyCode string, userID, itemID int64) (*Item, error) {
	f := Filter{CompanyCode: companyCode, UserID: userID}
	where, args := f.where(false)
	row := r.q.QueryRowContext(ctx, itemColumns+where+"\n  AND i.id = @itemId", append(args, sql.Named("itemId", itemID))...)
	it, err := scanItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	return it, err
}

func scanItem(row interface{ Scan(...any) error }) (*Item, error) {
	var (
		it                        Item
		readAt, letterDate        sql.NullTime
		recipientType, fwType     sql.NullString
		fwUserID, signerID        sql.NullInt64
		fwName, fwPos, signerName sql.NullString
		signerPos                 sql.NullString
		letterType, conf, urgency nullRef
	)
	err := row.Scan(&it.ID, &it.Status, &it.ReceivedAt, &readAt,
		&it.Source.Type, &recipientType, &fwType, &fwUserID, &fwName, &fwPos, &it.Source.Note,
		&it.Letter.ID, &it.Letter.LetterNumber, &it.Letter.Title, &letterDate,
		&letterType.ID, &letterType.Code, &letterType.Name, &conf.ID, &conf.Code, &conf.Name,
		&urgency.ID, &urgency.Code, &urgency.Name,
		&it.Letter.SenderUnit.ID, &it.Letter.SenderUnit.Code, &it.Letter.SenderUnit.Name,
		&signerID, &signerName, &signerPos,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("membaca inbox: %w", err)
	}
	it.ReadAt = timePtr(readAt)
	it.Source.RecipientType = stringPtr(recipientType)
	it.Source.ForwardType = stringPtr(fwType)
	if fwUserID.Valid {
		it.Source.ForwardedBy = &PersonRef{ID: fwUserID.Int64, Name: fwName.String, PositionName: fwPos.String}
	}
	it.Letter.ID = strings.ToLower(it.Letter.ID)
	it.Letter.LetterDate = timePtr(letterDate)
	it.Letter.LetterType, it.Letter.Confidentiality, it.Letter.Urgency = letterType.lookup(), conf.lookup(), urgency.lookup()
	if signerID.Valid {
		it.Letter.Approver = &PersonRef{ID: signerID.Int64, Name: signerName.String, PositionName: signerPos.String}
	}
	return &it, nil
}

// Counts menghitung item inbox per status dengan filter yang sama, kecuali status.
func (r *Repository) Counts(ctx context.Context, f Filter) (Counts, error) {
	where, args := f.where(false)
	q := `
SELECT i.status, COUNT(*)
FROM inbox_items i
JOIN letters l ON l.id = i.letter_id` + where + `
GROUP BY i.status`
	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("menghitung inbox: %w", err)
	}
	defer rows.Close()

	counts := Counts{}
	for rows.Next() {
		var (
			status string
			n      int
		)
		if err := rows.Scan(&status, &n); err != nil {
			return nil, fmt.Errorf("membaca jumlah inbox: %w", err)
		}
		counts[status] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca jumlah inbox: %w", err)
	}
	return counts, nil
}

// HasItem mengecek apakah surat ada di inbox user (langsung atau lewat penerusan).
func (r *Repository) HasItem(ctx context.Context, letterID, userID int64) (bool, error) {
	const q = `
SELECT CASE WHEN EXISTS (
    SELECT 1 FROM inbox_items
    WHERE letter_id = @letterId AND user_id = @userId AND deleted_at IS NULL) THEN 1 ELSE 0 END`
	var found bool
	if err := r.q.QueryRowContext(ctx, q, sql.Named("letterId", letterID), sql.Named("userId", userID)).Scan(&found); err != nil {
		return false, fmt.Errorf("memeriksa inbox: %w", err)
	}
	return found, nil
}

// nullRef menampung lookup hasil LEFT JOIN yang bisa kosong.
type nullRef struct {
	ID   sql.NullInt64
	Code sql.NullString
	Name sql.NullString
}

func (n nullRef) lookup() *LookupRef {
	if !n.ID.Valid {
		return nil
	}
	return &LookupRef{ID: n.ID.Int64, Code: n.Code.String, Name: n.Name.String}
}

func stringPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func timePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
