package audit

import (
	"time"

	"github.com/google/uuid"

	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// StuckAfterHours adalah batas surat dianggap tertahan di satu orang (Lacak Surat dan Pantauan).
const StuckAfterHours = 48

// Status surat yang dipakai pelacakan (sama dengan CHECK ck_letters_status).
const (
	statusDraft              = "DRAFT"
	statusInReview           = "IN_REVIEW"
	statusReturnedToDrafter  = "RETURNED_TO_DRAFTER"
	statusInApproval         = "IN_APPROVAL"
	statusReturnedToReviewer = "RETURNED_TO_REVIEWER"
	statusSent               = "SENT"
	statusCancelled          = "CANCELLED"
)

// inProcessStatuses adalah status surat yang sedang menunggu tindakan seseorang.
func inProcessStatuses() []string {
	return []string{statusInReview, statusReturnedToDrafter, statusInApproval, statusReturnedToReviewer}
}

// Error yang aman ditampilkan ke pengguna.
var (
	ErrLetterNotFound error = &apperr.NotFoundError{Message: "Surat tidak ditemukan"}
	ErrLogNotFound    error = &apperr.NotFoundError{Message: "Catatan audit tidak ditemukan"}
)

// Service menyediakan pelacakan surat, audit log, dan pantauan surat. Semua data otomatis
// disaring ke perusahaan user yang login.
type Service struct {
	repo *Repository
	loc  *time.Location
	now  func() time.Time
}

// NewService membuat Service. conn dipakai untuk semua query; loc adalah zona waktu tampilan.
func NewService(conn db.DBTX, loc *time.Location) *Service {
	return &Service{repo: NewRepository(conn), loc: loc, now: time.Now}
}

// parseLetterID memvalidasi uuid surat dari URL dan mengubahnya ke bentuk baku (huruf kecil).
func parseLetterID(raw string) (string, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", false
	}
	return id.String(), true
}

// waitingHours adalah lama menunggu sejak since, dibulatkan ke bawah (minimal 0).
func waitingHours(since, now time.Time) int {
	return max(int(now.Sub(since).Hours()), 0)
}
