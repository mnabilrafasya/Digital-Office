package service

import (
	"context"
	"time"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/auth"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/storage"
)

// Interface repository yang dibutuhkan service surat (didefinisikan di sisi pemakai).

// LetterStore adalah query tabel letters.
type LetterStore interface {
	Get(ctx context.Context, companyCode, uuid string) (*model.Letter, error)
	GetForUpdate(ctx context.Context, companyCode, uuid string) (*model.Letter, error)
	Create(ctx context.Context, nl model.NewLetter) (int64, string, error)
	UpdateContent(ctx context.Context, id int64, c model.LetterContent) error
	SoftDelete(ctx context.Context, id int64) error
	UpdateAttachments(ctx context.Context, id int64, attachmentsJSON *string) error
	IncrementVersion(ctx context.Context, id int64) (int, error)
	ApplyTransition(ctx context.Context, id int64, t model.LetterTransition) error
	List(ctx context.Context, f model.LetterListFilter) ([]model.LetterListRow, int, error)
}

// ParticipantStore adalah query tabel letter_participants.
type ParticipantStore interface {
	List(ctx context.Context, letterID int64) ([]model.Participant, error)
	SoftDelete(ctx context.Context, ids []int64) error
	Insert(ctx context.Context, letterID int64, items []model.NewParticipant) error
}

// RecipientStore adalah query tabel letter_recipients dan penerjemahan penerima menjadi user.
type RecipientStore interface {
	List(ctx context.Context, letterID int64) ([]model.Recipient, error)
	SoftDelete(ctx context.Context, ids []int64) error
	Insert(ctx context.Context, letterID int64, items []model.NewRecipient) error
	SetResolvedHeads(ctx context.Context, heads map[int64]int64) error
	ResolveUnitHeads(ctx context.Context, companyCode string, unitIDs []int64) (map[int64]int64, error)
	AliasMembers(ctx context.Context, companyCode string, levelIDs []int64) (map[int64][]int64, error)
}

// ActionStore adalah query tabel letter_actions.
type ActionStore interface {
	Insert(ctx context.Context, e model.ActionEntry) error
	RecordReadOnce(ctx context.Context, letterID, userID int64, role, status string, transitions []string) error
	Exists(ctx context.Context, letterID int64, action string) (bool, error)
	LastNote(ctx context.Context, letterID int64, action string) (*string, error)
	History(ctx context.Context, letterID int64) ([]model.HistoryRow, error)
}

// VersionStore adalah query tabel letter_versions.
type VersionStore interface {
	Insert(ctx context.Context, v model.NewVersion) error
	List(ctx context.Context, letterID int64) ([]model.Version, error)
	GetByNumber(ctx context.Context, letterID int64, number int) (*model.Version, error)
	GetByObjectKey(ctx context.Context, letterID int64, objectKey string) (*model.Version, error)
	ObjectKeys(ctx context.Context, letterID int64) ([]string, error)
}

// InboxStore adalah query tabel inbox_items.
type InboxStore interface {
	InsertDirect(ctx context.Context, letterID int64, entries []model.InboxEntry) error
}

// AuditStore adalah query tabel audit_logs.
type AuditStore interface {
	Insert(ctx context.Context, e model.AuditEntry) error
}

// NumberingStore adalah query penomoran surat.
type NumberingStore interface {
	LetterNumberFormat(ctx context.Context, companyCode string) (string, error)
	NextSequence(ctx context.Context, unitID int64, year int) (int, error)
}

// LookupRefStore adalah query referensi lookup dan alias.
type LookupRefStore interface {
	FindRefs(ctx context.Context, companyCode string, ids []int64) (map[int64]model.LookupRef, error)
	FindAliases(ctx context.Context, companyCode string, ids []int64) (map[int64]repository.AliasRow, error)
}

// UnitRefStore adalah query referensi unit dan template.
type UnitRefStore interface {
	FindRefs(ctx context.Context, companyCode string, ids []int64) (map[int64]model.UnitRef, error)
	FindTemplate(ctx context.Context, companyCode string, id int64) (*model.TemplateRef, error)
}

// UserRefStore adalah query referensi user.
type UserRefStore interface {
	FindParticipantUsers(ctx context.Context, companyCode string, ids []int64, reviewPerm, approvePerm string) (map[int64]model.ParticipantUser, error)
	FindLabels(ctx context.Context, companyCode string, ids []int64) (map[int64]model.UserLabel, error)
}

// LetterRepos adalah kumpulan repository surat yang terikat ke satu koneksi atau transaksi.
type LetterRepos struct {
	Letters      LetterStore
	Participants ParticipantStore
	Recipients   RecipientStore
	Actions      ActionStore
	Versions     VersionStore
	Inbox        InboxStore
	Audit        AuditStore
	Numbering    NumberingStore
	Lookups      LookupRefStore
	Units        UnitRefStore
	Users        UserRefStore
}

// LetterReposFactory membuat LetterRepos di atas koneksi atau transaksi tertentu.
type LetterReposFactory func(q db.DBTX) LetterRepos

// FileStore adalah operasi storage yang dibutuhkan service file.
type FileStore interface {
	PresignUpload(ctx context.Context, key string, maxSize int64, ttl time.Duration) (storage.PresignedUpload, error)
	PresignDownload(ctx context.Context, key, fileName string, ttl time.Duration) (string, time.Time, error)
	Stat(ctx context.Context, key string) (model.StoredObject, error)
	Copy(ctx context.Context, srcKey, dstKey string) error
	Checksum(ctx context.Context, key string) (string, error)
	Remove(ctx context.Context, key string) error
}

// CapacityChecker memeriksa sisa ruang storage.
type CapacityChecker interface {
	HasRoom(ctx context.Context, size int64) bool
}

// UploadTokenService membuat dan memeriksa upload token.
type UploadTokenService interface {
	Issue(c auth.UploadClaims) (string, time.Time, error)
	Verify(token string, userID int64, letterUUID, kind string) (auth.UploadClaims, error)
}

// LetterConfig adalah pengaturan service surat.
type LetterConfig struct {
	Location       *time.Location // zona waktu bisnis (nomor surat, tampilan waktu)
	MaxUploadBytes int64
	UploadURLTTL   time.Duration
	DownloadURLTTL time.Duration
}

// LetterDeps adalah dependency LetterService.
type LetterDeps struct {
	Tx       db.TxRunner
	Repos    LetterReposFactory
	Read     LetterRepos // terikat ke pool koneksi, untuk bacaan di luar transaksi
	Files    FileStore
	Capacity CapacityChecker
	Uploads  UploadTokenService
	Config   LetterConfig
	Now      func() time.Time
}
