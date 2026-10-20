package model

import (
	"encoding/json"
	"time"
)

// Peran peserta dan tipe penerima surat.
const (
	RoleDrafter  = "DRAFTER"
	RoleReviewer = "REVIEWER"
	RoleApprover = "APPROVER"

	RecipientTo = "TO"
	RecipientCC = "CC"
)

// Letter adalah satu baris tabel letters.
type Letter struct {
	ID                   int64
	UUID                 string
	CompanyCode          string
	LetterNumber         *string
	Content              LetterContent
	AttachmentsJSON      *string
	Status               string
	SenderUnitID         int64
	SenderUnitName       *string
	DrafterID            int64
	CurrentVersion       int
	AssigneeID           *int64
	ForwardedByID        *int64
	ReturnedByReviewerID *int64
	ReturnedByApproverID *int64
	ApprovedAt           *time.Time
	RowVersion           []byte
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// LetterContent adalah isian surat yang bisa diedit drafter.
type LetterContent struct {
	TemplateID        *int64
	LetterTypeID      *int64
	ClassificationID  *int64
	ConfidentialityID *int64
	UrgencyID         *int64
	Title             *string
	Summary           *string
	SpecialNote       *string
}

// Participant adalah reviewer atau approver aktif (belum di-soft-delete) di satu surat.
type Participant struct {
	ID        int64
	UserID    int64
	Role      string
	StepOrder int
}

// Recipient adalah penerima aktif surat: unit ATAU alias level jabatan.
type Recipient struct {
	ID             int64
	Type           string
	UnitID         *int64
	AliasLevelID   *int64
	ResolvedHeadID *int64
	SortOrder      int
}

// NewParticipant adalah peserta yang akan disimpan.
type NewParticipant struct {
	UserID    int64
	Role      string
	StepOrder int
}

// NewRecipient adalah penerima yang akan disimpan.
type NewRecipient struct {
	Type         string
	UnitID       *int64
	AliasLevelID *int64
	SortOrder    int
}

// NewLetter adalah draf baru yang akan disimpan.
type NewLetter struct {
	CompanyCode  string
	SenderUnitID int64
	DrafterID    int64
	Content      LetterContent
}

// IDChange menandai kolom ID yang ikut diubah: Set=false berarti tidak diubah,
// Set=true dengan Value nil berarti diisi NULL.
type IDChange struct {
	Set   bool
	Value *int64
}

// LetterTransition adalah perubahan baris letters akibat satu aksi alur persetujuan.
type LetterTransition struct {
	Status             string
	AssigneeID         *int64
	ForwardedBy        IDChange
	ReturnedByReviewer IDChange
	ReturnedByApprover IDChange
	SenderUnitName     *string     // diisi bila tidak nil
	Sign               *LetterSign // diisi saat SIGN
}

// LetterSign adalah data penandatanganan dan penomoran surat.
type LetterSign struct {
	LetterNumber string
	NumberSeq    int
	NumberYear   int
	SignedByID   int64
}

// ActionEntry adalah satu baris letter_actions yang akan disimpan.
type ActionEntry struct {
	LetterID      int64
	ActorID       int64
	TargetUserID  *int64
	ActorRole     string
	Action        string
	FromStatus    string
	ToStatus      string
	Note          *string
	VersionNumber *int
}

// AuditEntry adalah satu baris audit_logs yang akan disimpan.
type AuditEntry struct {
	CompanyCode string
	EntityType  string
	EntityID    int64
	EntityUUID  string
	Action      string
	ActorID     int64
	ActorName   string
	Snapshot    string
	IPAddress   string
}

// InboxEntry adalah satu penerima akhir surat (sudah diterjemahkan dari unit/alias).
type InboxEntry struct {
	UserID        int64  `json:"u"`
	RecipientType string `json:"t"`
}

// LetterListFilter adalah kriteria daftar surat.
type LetterListFilter struct {
	CompanyCode  string
	DrafterID    int64    // scope=mine
	AssigneeID   int64    // scope=tasks
	TaskStatuses []string // status yang dianggap tugas (scope=tasks)
	Statuses     []string // filter status opsional
	Search       string
	Offset       int
	Limit        int
}

// LetterListRow adalah satu baris daftar surat.
type LetterListRow struct {
	UUID            string
	LetterNumber    *string
	Title           *string
	Status          string
	LetterType      *LookupLabel
	Confidentiality *LookupLabel
	Urgency         *LookupLabel
	SenderUnit      UnitLabel
	Assignee        *UserLabel
	StepOrder       *int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// HistoryRow adalah satu baris riwayat dari letter_actions.
type HistoryRow struct {
	Action        string
	ActorRole     string
	Actor         UserLabel
	Target        *UserLabel
	FromStatus    *string
	ToStatus      string
	Note          *string
	VersionNumber *int
	CreatedAt     time.Time
}

// ---- Bentuk JSON untuk klien ----

// LetterDetail adalah detail surat untuk klien.
type LetterDetail struct {
	ID              string           `json:"id"`
	Status          string           `json:"status"`
	LetterNumber    *string          `json:"letterNumber"`
	LetterDate      *string          `json:"letterDate"`
	Title           *string          `json:"title"`
	Summary         *string          `json:"summary"`
	SpecialNote     *string          `json:"specialNote"`
	LetterType      *LookupLabel     `json:"letterType"`
	Classification  *LookupLabel     `json:"classification"`
	Confidentiality *LookupLabel     `json:"confidentiality"`
	Urgency         *LookupLabel     `json:"urgency"`
	Template        *TemplateRef     `json:"template"`
	SenderUnit      UnitLabel        `json:"senderUnit"`
	Drafter         UserLabel        `json:"drafter"`
	Steps           []StepView       `json:"steps"`
	Recipients      RecipientsView   `json:"recipients"`
	Attachments     []AttachmentView `json:"attachments"`
	CurrentVersion  int              `json:"currentVersion"`
	CurrentAssignee *UserLabel       `json:"currentAssignee"`
	LastReturnNote  *string          `json:"lastReturnNote"`
	AllowedActions  []string         `json:"allowedActions"`
	CreatedAt       string           `json:"createdAt"`
	UpdatedAt       string           `json:"updatedAt"`
}

// StepView adalah satu tahap persetujuan (reviewer atau approver).
type StepView struct {
	Role  string    `json:"role"`
	Order int       `json:"order"`
	State string    `json:"state"`
	User  UserLabel `json:"user"`
}

// RecipientsView adalah penerima surat, dikelompokkan Kepada (to) dan Tembusan (cc).
type RecipientsView struct {
	To []RecipientView `json:"to"`
	CC []RecipientView `json:"cc"`
}

// RecipientView adalah satu penerima: unit (dengan pimpinan yang dituju) atau alias.
type RecipientView struct {
	Type         string // UNIT atau ALIAS
	Unit         *UnitLabel
	ResolvedHead *UserLabel
	Alias        *Alias
}

// MarshalJSON mengirim {"type","unit","resolvedHead"} untuk unit dan {"type","alias"} untuk alias.
func (r RecipientView) MarshalJSON() ([]byte, error) {
	if r.Type == "ALIAS" {
		return json.Marshal(struct {
			Type  string `json:"type"`
			Alias *Alias `json:"alias"`
		}{r.Type, r.Alias})
	}
	return json.Marshal(struct {
		Type         string     `json:"type"`
		Unit         *UnitLabel `json:"unit"`
		ResolvedHead *UserLabel `json:"resolvedHead"`
	}{r.Type, r.Unit, r.ResolvedHead})
}

// LetterListItem adalah satu surat di daftar.
type LetterListItem struct {
	ID              string       `json:"id"`
	LetterNumber    *string      `json:"letterNumber"`
	Title           *string      `json:"title"`
	Status          string       `json:"status"`
	LetterType      *LookupLabel `json:"letterType"`
	Confidentiality *LookupLabel `json:"confidentiality"`
	Urgency         *LookupLabel `json:"urgency"`
	SenderUnit      UnitLabel    `json:"senderUnit"`
	CurrentAssignee *UserLabel   `json:"currentAssignee"`
	CurrentStep     *StepRef     `json:"currentStep"`
	CreatedAt       string       `json:"createdAt"`
	UpdatedAt       string       `json:"updatedAt"`
}

// StepRef menunjuk tahap yang sedang berjalan.
type StepRef struct {
	Role  string `json:"role"`
	Order int    `json:"order"`
}

// HistoryItem adalah satu baris riwayat surat untuk klien.
type HistoryItem struct {
	Action        string     `json:"action"`
	ActorRole     string     `json:"actorRole"`
	Actor         UserLabel  `json:"actor"`
	Target        *UserLabel `json:"target"`
	FromStatus    *string    `json:"fromStatus"`
	ToStatus      string     `json:"toStatus"`
	Note          *string    `json:"note"`
	VersionNumber *int       `json:"versionNumber"`
	CreatedAt     string     `json:"createdAt"`
}
