package model

import (
	"encoding/json"
	"time"
)

// Status surat (sama dengan CHECK ck_letters_status).
const (
	StatusDraft              = "DRAFT"
	StatusInReview           = "IN_REVIEW"
	StatusReturnedToDrafter  = "RETURNED_TO_DRAFTER"
	StatusInApproval         = "IN_APPROVAL"
	StatusReturnedToReviewer = "RETURNED_TO_REVIEWER"
	StatusSent               = "SENT"
	StatusCancelled          = "CANCELLED"
)

// Peran dalam surat.
const (
	RoleDrafter  = "DRAFTER"
	RoleReviewer = "REVIEWER"
	RoleApprover = "APPROVER"
)

// Jenis penerima surat.
const (
	RecipientTo = "TO" // Kepada
	RecipientCC = "CC" // Tembusan
)

// Keadaan satu tahap reviewer/approver di detail surat.
const (
	StepDone    = "DONE"
	StepActive  = "ACTIVE"
	StepPending = "PENDING"
)

// LetterDraft adalah isi surat yang bisa diedit drafter, sudah divalidasi formatnya.
// nil berarti kosong.
type LetterDraft struct {
	LetterTypeID      *int64
	TemplateID        *int64
	ClassificationID  *int64
	ConfidentialityID *int64
	UrgencyID         *int64
	Title             *string
	Summary           *string
	SpecialNote       *string
	ReviewerIDs       []int64 // urutan = step_order
	ApproverID        *int64
	Recipients        []RecipientRef
}

// RecipientRef adalah satu penerima surat: tepat satu dari UnitID atau AliasLevelID terisi.
type RecipientRef struct {
	Type         string // RecipientTo atau RecipientCC
	UnitID       int64
	AliasLevelID int64
}

// LetterLock adalah data surat yang dibaca sambil mengunci barisnya untuk diubah.
type LetterLock struct {
	ID         int64
	DrafterID  int64
	Status     string
	RowVersion []byte
}

// LetterDetail adalah detail surat untuk pembuat dan peserta (Lampiran A).
type LetterDetail struct {
	ID              string           `json:"id"`
	InternalID      int64            `json:"-"`
	RowVersion      []byte           `json:"-"`
	Status          string           `json:"status"`
	LetterNumber    *string          `json:"letterNumber"`
	LetterDate      *time.Time       `json:"letterDate"`
	Title           *string          `json:"title"`
	Summary         *string          `json:"summary"`
	SpecialNote     *string          `json:"specialNote"`
	LetterType      *LookupRef       `json:"letterType"`
	Classification  *LookupRef       `json:"classification"`
	Confidentiality *LookupRef       `json:"confidentiality"`
	Urgency         *LookupRef       `json:"urgency"`
	Template        *TemplateRef     `json:"template"`
	SenderUnit      UnitRef          `json:"senderUnit"`
	Drafter         PersonRef        `json:"drafter"`
	Steps           []LetterStep     `json:"steps"`
	Recipients      LetterRecipients `json:"recipients"`
	Attachments     []Attachment     `json:"attachments"`
	CurrentVersion  int              `json:"currentVersion"`
	CurrentAssignee *PersonRef       `json:"currentAssignee"`
	LastReturnNote  *string          `json:"lastReturnNote"`
	AllowedActions  []string         `json:"allowedActions"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
}

// IsParticipant mengecek apakah user adalah drafter, reviewer, atau approver surat ini.
func (d *LetterDetail) IsParticipant(userID int64) bool {
	if d.Drafter.ID == userID {
		return true
	}
	for _, s := range d.Steps {
		if s.User.ID == userID {
			return true
		}
	}
	return false
}

// LookupRef adalah ringkasan lookup di detail surat.
type LookupRef struct {
	ID   int64  `json:"id,string"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// TemplateRef adalah ringkasan template surat.
type TemplateRef struct {
	ID   int64  `json:"id,string"`
	Name string `json:"name"`
}

// UnitRef adalah ringkasan unit.
type UnitRef struct {
	ID   int64  `json:"id,string"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// PersonRef adalah ringkasan user beserta jabatannya.
type PersonRef struct {
	ID           int64  `json:"id,string"`
	Name         string `json:"name"`
	PositionName string `json:"positionName"`
}

// StepUser adalah reviewer/approver di satu tahap.
type StepUser struct {
	ID           int64  `json:"id,string"`
	Name         string `json:"name"`
	PositionName string `json:"positionName"`
	UnitName     string `json:"unitName"`
}

// LetterStep adalah satu tahap reviewer/approver.
type LetterStep struct {
	Role  string   `json:"role"`
	Order int      `json:"order"`
	State string   `json:"state"`
	User  StepUser `json:"user"`
}

// LetterRecipients adalah penerima surat, dikelompokkan Kepada dan Tembusan.
type LetterRecipients struct {
	To []LetterRecipient `json:"to"`
	CC []LetterRecipient `json:"cc"`
}

// LetterRecipient adalah penerima berupa unit (Unit terisi) atau alias jabatan (Alias terisi).
type LetterRecipient struct {
	Unit         *UnitRef
	ResolvedHead *PersonRef // pimpinan unit yang menerima; diisi saat surat ditandatangani
	Alias        *Alias
}

// MarshalJSON menulis {"type":"UNIT","unit","resolvedHead"} atau {"type":"ALIAS","alias"}.
func (r LetterRecipient) MarshalJSON() ([]byte, error) {
	if r.Alias != nil {
		return json.Marshal(struct {
			Type  string `json:"type"`
			Alias *Alias `json:"alias"`
		}{"ALIAS", r.Alias})
	}
	return json.Marshal(struct {
		Type         string     `json:"type"`
		Unit         *UnitRef   `json:"unit"`
		ResolvedHead *PersonRef `json:"resolvedHead"`
	}{"UNIT", r.Unit, r.ResolvedHead})
}

// Alias adalah alias penerima berdasarkan level jabatan, contoh "Seluruh VP".
type Alias struct {
	LevelID     int64  `json:"levelId,string"`
	Code        string `json:"code"`
	Label       string `json:"label"`
	MemberCount int    `json:"memberCount"`
}

// Attachment adalah lampiran surat (diisi mulai Tahap 5).
type Attachment struct {
	ID          string    `json:"id"`
	FileName    string    `json:"fileName"`
	Size        int64     `json:"size"`
	ContentType string    `json:"contentType"`
	UploadedAt  time.Time `json:"uploadedAt"`
	UploadedBy  struct {
		ID   int64  `json:"id,string"`
		Name string `json:"name"`
	} `json:"uploadedBy"`
}
