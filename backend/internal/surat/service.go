// Package surat berisi siklus hidup surat: draf, detail, daftar, riwayat, dan alur persetujuan.
// Aturan perpindahan status ada di alur.go sebagai fungsi murni; file lain mengikuti tanggung
// jawabnya (draf.go, persetujuan.go, repository_*.go, handler*.go).
package surat

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/db"
)

// ActionEdit adalah tombol simpan/ubah draf di allowedActions (aksi alur lain didefinisikan di alur.go).
const ActionEdit = "EDIT"

// Store adalah query surat yang dibutuhkan Service.
type Store interface {
	// Draf
	Create(ctx context.Context, n NewLetter) (string, error)
	UpdateDraft(ctx context.Context, letterID int64, d *LetterDraft) error
	LoadDraft(ctx context.Context, letterID int64) (*LetterDraft, error)

	// Akses dan alur persetujuan
	LockByUUID(ctx context.Context, companyCode, uuid string) (*LetterLock, error)
	FindByUUID(ctx context.Context, companyCode, uuid string) (*LetterLock, error)
	IsParticipant(ctx context.Context, letterID, userID int64) (bool, error)
	Participants(ctx context.Context, letterID int64) (reviewers []int64, approverID int64, err error)
	ApplyTransition(ctx context.Context, letterID, actorID int64, d Decision, note *string) error
	RecordRead(ctx context.Context, letterID, userID int64, role string) error

	// Naskah
	AddVersion(ctx context.Context, v NewVersion) (version int, rowVersion []byte, err error)
	Versions(ctx context.Context, letterID int64) ([]DocumentVersion, error)
	FindVersion(ctx context.Context, letterID int64, number int) (*DocumentVersion, error)
	LatestVersion(ctx context.Context, letterID int64, versionType string) (*DocumentVersion, error)

	// Terbit
	NextNumber(ctx context.Context, letterID int64, year int) (seq int, unitCode, format string, err error)
	Publish(ctx context.Context, letterID int64, p Publication) error

	// Tampilan
	FindDetail(ctx context.Context, companyCode, uuid string) (*LetterDetail, error)
	List(ctx context.Context, f LetterFilter) ([]LetterListItem, int, error)
	Actions(ctx context.Context, letterID int64) ([]LetterAction, error)
}

// Directory adalah data organisasi yang dirujuk surat: unit, user (reviewer/approver), dan
// penerima (dipenuhi repository organisasi).
type Directory interface {
	ActiveUnitIDs(ctx context.Context, companyCode string, ids []int64) (map[int64]bool, error)
	ActiveUserIDs(ctx context.Context, companyCode string, ids []int64) (map[int64]bool, error)
	EligibleUserIDs(ctx context.Context, companyCode, permissionCode string, ids []int64) (map[int64]bool, error)
	UnreachableRecipients(ctx context.Context, companyCode string, unitIDs, aliasIDs []int64) (units, aliases []string, err error)
	UnitHeadIDs(ctx context.Context, companyCode string, unitIDs []int64) (map[int64]int64, error)
	AliasMemberIDs(ctx context.Context, companyCode string, aliasIDs []int64) (map[int64][]int64, error)
}

// InboxStore memasukkan surat terbit ke inbox penerima, dan mengecek apakah surat ada di inbox
// seseorang (penerima boleh membuka surat terbit). Dipenuhi repository inbox.
type InboxStore interface {
	DeliverDirect(ctx context.Context, letterID int64, to, cc []int64) error
	HasItem(ctx context.Context, letterID, userID int64) (bool, error)
}

// Lookups memeriksa data acuan yang dirujuk surat (dipenuhi repository masterdata).
type Lookups interface {
	LookupCategories(ctx context.Context, companyCode string, ids []int64) (map[int64]string, error)
}

// Templates memeriksa template surat yang dirujuk draf.
type Templates interface {
	TemplateLetterType(ctx context.Context, companyCode string, templateID int64) (*int64, error)
}

// DocumentFiles menyimpan dan membaca file naskah surat (dipenuhi platform/storage).
type DocumentFiles interface {
	PutDocument(ctx context.Context, objectKey string, data []byte, contentType string) error
	OpenDocument(ctx context.Context, objectKey string) (io.ReadCloser, int64, error)
}

// PDFMaker membuat PDF resmi: .docx diubah ke PDF (LibreOffice) lalu QR ditempel di halaman
// terakhir. Dipenuhi platform/pdf.
type PDFMaker interface {
	FromOffice(ctx context.Context, fileName string, data []byte) ([]byte, error)
	StampLastPage(ctx context.Context, pdf, png []byte) ([]byte, error)
}

// AuditStore mencatat jejak audit.
type AuditStore interface {
	Record(ctx context.Context, e audit.Entry) error
}

// Repos adalah repository yang dibutuhkan Service. Letters, Audit, dan Inbox dibuat di atas
// koneksi atau transaksi; Directory, Lookups, dan Templates hanya membaca di luar transaksi.
// Files adalah penyimpanan file naskah (MinIO); PDF adalah pembuat PDF resmi (Gotenberg).
type Repos struct {
	Letters   func(q db.DBTX) Store
	Audit     func(q db.DBTX) AuditStore
	Inbox     func(q db.DBTX) InboxStore
	Directory Directory
	Lookups   Lookups
	Templates Templates
	Files     DocumentFiles
	PDF       PDFMaker
}

// Service mengelola surat: draf, alur persetujuan, daftar, dan riwayat. Semua surat
// otomatis disaring ke perusahaan user yang login.
type Service struct {
	tx     db.TxRunner
	repos  Repos
	reader Store      // repository surat di luar transaksi (baca saja)
	inbox  InboxStore // repository inbox di luar transaksi (baca saja)
	loc    *time.Location
}

// NewService membuat Service. conn dipakai untuk query baca di luar transaksi;
// loc adalah zona waktu yang dikirim ke klien.
func NewService(tx db.TxRunner, conn db.DBTX, repos Repos, loc *time.Location) *Service {
	return &Service{tx: tx, repos: repos, reader: repos.Letters(conn), inbox: repos.Inbox(conn), loc: loc}
}

// rolePermission adalah permission yang wajib dimiliki untuk beraksi sebagai peran tertentu.
// Drafter tidak butuh permission tambahan (cukup sebagai pembuat surat).
func rolePermission(role string) string {
	switch role {
	case RoleReviewer:
		return auth.LetterReview
	case RoleApprover:
		return auth.LetterApprove
	}
	return ""
}

// isEditable: isi surat hanya bisa diubah drafter saat masih draf atau dikembalikan kepadanya.
func isEditable(status string) bool {
	return status == StatusDraft || status == StatusReturnedToDrafter
}

// notFoundAs mengubah db.ErrNotFound menjadi ErrLetterNotFound; error lain diteruskan.
func notFoundAs(err error) error {
	if errors.Is(err, db.ErrNotFound) {
		return ErrLetterNotFound
	}
	return err
}

// parseLetterID memvalidasi uuid surat dari URL dan mengubahnya ke bentuk baku (huruf kecil).
func parseLetterID(raw string) (string, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", false
	}
	return id.String(), true
}
