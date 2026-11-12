package delegasi

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// maxNoteLength adalah panjang maksimal keterangan delegasi (kolom delegations.note).
const maxNoteLength = 200

var (
	errNotFound     = &apperr.NotFoundError{Message: "Delegasi tidak ditemukan"}
	errEnded        = &apperr.InvalidStateError{Message: "Delegasi yang sudah selesai tidak bisa diubah atau dihapus"}
	errSelf         = apperr.Invalid("Pengganti tidak boleh pegawai itu sendiri")
	errSubstitute   = apperr.Invalid("Pengganti tidak ditemukan atau tidak aktif di perusahaan Anda")
	errOwner        = apperr.Invalid("Pegawai tidak ditemukan atau tidak aktif di perusahaan Anda")
	errUntilPast    = apperr.Invalid("Tanggal selesai tidak boleh sebelum hari ini")
	errPeriodOrder  = apperr.Invalid("Tanggal mulai tidak boleh setelah tanggal selesai")
	errStatusFilter = apperr.Invalid("Parameter status wajib ACTIVE, UPCOMING, atau ENDED")
)

// Store adalah query delegasi yang dibutuhkan Service (dipenuhi Repository).
type Store interface {
	List(ctx context.Context, f listFilter) ([]Delegation, int, error)
	Find(ctx context.Context, companyCode string, id int64, today string, lock bool) (*Delegation, error)
	LockCompany(ctx context.Context, companyCode string) error
	FindConflict(ctx context.Context, companyCode string, p period, exceptID int64) (*Conflict, error)
	ActiveUser(ctx context.Context, companyCode string, id int64) (*PersonRef, error)
	Insert(ctx context.Context, companyCode string, p period, createdBy int64) (int64, error)
	Update(ctx context.Context, id int64, p period) error
	Delete(ctx context.Context, id int64) error
}

// AuditStore mencatat jejak audit.
type AuditStore interface {
	Record(ctx context.Context, e audit.Entry) error
}

// Service mengelola jadwal delegasi: pegawai untuk dirinya sendiri (/me/delegations) dan admin
// perusahaan (USER_MANAGE) untuk pegawai mana pun (/admin/delegations).
type Service struct {
	tx    db.TxRunner
	store func(q db.DBTX) Store
	audit func(q db.DBTX) AuditStore
	read  Store
	loc   *time.Location
	now   func() time.Time
}

// NewService membuat Service. conn dipakai untuk query baca; loc menentukan "hari ini".
func NewService(tx db.TxRunner, conn db.DBTX, store func(q db.DBTX) Store, audit func(q db.DBTX) AuditStore,
	loc *time.Location) *Service {
	return &Service{tx: tx, store: store, audit: audit, read: store(conn), loc: loc, now: time.Now}
}

// today adalah tanggal hari ini (YYYY-MM-DD) di zona waktu aplikasi.
func (s *Service) today() string {
	return s.now().In(s.loc).Format(time.DateOnly)
}

// --- Pegawai -------------------------------------------------------------------------------

// Mine mengambil delegasi milik user (berlaku dan akan datang) dan jadwal yang ia wakili.
func (s *Service) Mine(ctx context.Context, user *auth.CurrentUser) (*Mine, error) {
	open := []string{StatusActive, StatusUpcoming}
	mine, _, err := s.read.List(ctx, listFilter{CompanyCode: user.Company.Code, Today: s.today(), OwnerID: user.ID, Statuses: open})
	if err != nil {
		return nil, err
	}
	representing, _, err := s.read.List(ctx, listFilter{CompanyCode: user.Company.Code, Today: s.today(), DelegateID: user.ID, Statuses: open})
	if err != nil {
		return nil, err
	}
	s.localize(mine)
	s.localize(representing)
	return &Mine{Delegations: mine, Representing: representing}, nil
}

// CreateMine menambah jadwal delegasi untuk user sendiri.
func (s *Service) CreateMine(ctx context.Context, user *auth.CurrentUser, in Input) (*Delegation, error) {
	return s.create(ctx, user, user.ID, in)
}

// UpdateMine mengubah jadwal delegasi milik user sendiri.
func (s *Service) UpdateMine(ctx context.Context, user *auth.CurrentUser, rawID string, in Input) (*Delegation, error) {
	return s.update(ctx, user, rawID, in, true)
}

// DeleteMine menghapus jadwal delegasi milik user sendiri.
func (s *Service) DeleteMine(ctx context.Context, user *auth.CurrentUser, rawID string) error {
	return s.delete(ctx, user, rawID, true)
}

// --- Admin perusahaan ------------------------------------------------------------------------

// List mengambil jadwal delegasi perusahaan admin beserta jumlah totalnya.
func (s *Service) List(ctx context.Context, user *auth.CurrentUser, lq ListQuery) ([]Delegation, int, error) {
	if !user.HasPermission(auth.UserManage) {
		return nil, 0, apperr.ErrForbidden
	}
	f := listFilter{CompanyCode: user.Company.Code, Today: s.today(), Offset: lq.Offset, Limit: lq.Limit}
	if st := strings.ToUpper(strings.TrimSpace(lq.Status)); st != "" {
		if st != StatusActive && st != StatusUpcoming && st != StatusEnded {
			return nil, 0, errStatusFilter
		}
		f.Statuses = []string{st}
	}
	if strings.TrimSpace(lq.UserID) != "" {
		id, err := parseID(lq.UserID)
		if err != nil {
			return nil, 0, apperr.Invalid("Parameter userId tidak valid")
		}
		f.PersonID = id
	}
	items, total, err := s.read.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	s.localize(items)
	return items, total, nil
}

// Create (admin) menambah jadwal delegasi untuk pegawai in.UserID.
func (s *Service) Create(ctx context.Context, user *auth.CurrentUser, in Input) (*Delegation, error) {
	if !user.HasPermission(auth.UserManage) {
		return nil, apperr.ErrForbidden
	}
	ownerID, err := parseID(in.UserID)
	if err != nil {
		return nil, errOwner
	}
	return s.create(ctx, user, ownerID, in)
}

// Update (admin) mengubah jadwal delegasi pegawai mana pun di perusahaannya.
func (s *Service) Update(ctx context.Context, user *auth.CurrentUser, rawID string, in Input) (*Delegation, error) {
	if !user.HasPermission(auth.UserManage) {
		return nil, apperr.ErrForbidden
	}
	return s.update(ctx, user, rawID, in, false)
}

// Delete (admin) menghapus jadwal delegasi pegawai mana pun di perusahaannya.
func (s *Service) Delete(ctx context.Context, user *auth.CurrentUser, rawID string) error {
	if !user.HasPermission(auth.UserManage) {
		return apperr.ErrForbidden
	}
	return s.delete(ctx, user, rawID, false)
}

// --- Inti ------------------------------------------------------------------------------------

func (s *Service) create(ctx context.Context, user *auth.CurrentUser, ownerID int64, in Input) (*Delegation, error) {
	p, err := s.parse(ownerID, in)
	if err != nil {
		return nil, err
	}
	var id int64
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		st := s.store(q)
		if err := st.LockCompany(ctx, user.Company.Code); err != nil {
			return err
		}
		if err := s.checkPeople(ctx, st, user.Company.Code, p); err != nil {
			return err
		}
		if err := checkConflict(ctx, st, user.Company.Code, p, 0); err != nil {
			return err
		}
		if id, err = st.Insert(ctx, user.Company.Code, p, user.ID); err != nil {
			return err
		}
		after, err := st.Find(ctx, user.Company.Code, id, s.today(), false)
		if err != nil {
			return err
		}
		return s.record(ctx, q, user, id, audit.ActionDelegationCreate, nil, after)
	})
	if err != nil {
		return nil, err
	}
	return s.find(ctx, user.Company.Code, id)
}

func (s *Service) update(ctx context.Context, user *auth.CurrentUser, rawID string, in Input, ownOnly bool) (*Delegation, error) {
	id, err := parseID(rawID)
	if err != nil {
		return nil, errNotFound
	}
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		st := s.store(q)
		if err := st.LockCompany(ctx, user.Company.Code); err != nil {
			return err
		}
		old, err := s.editable(ctx, st, user, id, ownOnly)
		if err != nil {
			return err
		}
		p, err := s.parse(old.User.ID, in)
		if err != nil {
			return err
		}
		if err := s.checkPeople(ctx, st, user.Company.Code, p); err != nil {
			return err
		}
		if err := checkConflict(ctx, st, user.Company.Code, p, id); err != nil {
			return err
		}
		if err := st.Update(ctx, id, p); err != nil {
			return err
		}
		after, err := st.Find(ctx, user.Company.Code, id, s.today(), false)
		if err != nil {
			return err
		}
		return s.record(ctx, q, user, id, audit.ActionDelegationUpdate, old, after)
	})
	if err != nil {
		return nil, err
	}
	return s.find(ctx, user.Company.Code, id)
}

func (s *Service) delete(ctx context.Context, user *auth.CurrentUser, rawID string, ownOnly bool) error {
	id, err := parseID(rawID)
	if err != nil {
		return errNotFound
	}
	return s.tx.InTx(ctx, func(q db.DBTX) error {
		st := s.store(q)
		old, err := s.editable(ctx, st, user, id, ownOnly)
		if err != nil {
			return err
		}
		if err := st.Delete(ctx, id); err != nil {
			return err
		}
		return s.record(ctx, q, user, id, audit.ActionDelegationDelete, old, nil)
	})
}

// editable mengunci jadwal yang boleh diubah user: milik perusahaannya, milik dirinya sendiri bila
// ownOnly, dan belum selesai.
func (s *Service) editable(ctx context.Context, st Store, user *auth.CurrentUser, id int64, ownOnly bool) (*Delegation, error) {
	d, err := st.Find(ctx, user.Company.Code, id, s.today(), true)
	if errors.Is(err, db.ErrNotFound) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	if ownOnly && d.User.ID != user.ID {
		return nil, errNotFound
	}
	if d.Status == StatusEnded {
		return nil, errEnded
	}
	return d, nil
}

// parse merapikan dan memvalidasi isian delegasi milik ownerID.
func (s *Service) parse(ownerID int64, in Input) (period, error) {
	p := period{UserID: ownerID}
	id, err := parseID(in.SubstituteID)
	if err != nil {
		return p, apperr.Invalid("Pengganti wajib dipilih")
	}
	p.DelegateID = id
	if p.DelegateID == p.UserID {
		return p, errSelf
	}
	from, err := time.Parse(time.DateOnly, strings.TrimSpace(in.From))
	if err != nil {
		return p, apperr.Invalid("Tanggal mulai wajib diisi dengan format YYYY-MM-DD")
	}
	until, err := time.Parse(time.DateOnly, strings.TrimSpace(in.Until))
	if err != nil {
		return p, apperr.Invalid("Tanggal selesai wajib diisi dengan format YYYY-MM-DD")
	}
	if from.After(until) {
		return p, errPeriodOrder
	}
	p.From, p.Until = from.Format(time.DateOnly), until.Format(time.DateOnly)
	if p.Until < s.today() {
		return p, errUntilPast
	}
	if in.Note != nil {
		note := strings.TrimSpace(*in.Note)
		if utf8.RuneCountInString(note) > maxNoteLength {
			return p, apperr.Invalid(fmt.Sprintf("Keterangan maksimal %d karakter", maxNoteLength))
		}
		if note != "" {
			p.Note = &note
		}
	}
	return p, nil
}

// checkPeople memastikan pegawai dan penggantinya user aktif di perusahaan yang sama.
func (s *Service) checkPeople(ctx context.Context, st Store, companyCode string, p period) error {
	if _, err := st.ActiveUser(ctx, companyCode, p.UserID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return errOwner
		}
		return err
	}
	if _, err := st.ActiveUser(ctx, companyCode, p.DelegateID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return errSubstitute
		}
		return err
	}
	return nil
}

// checkConflict menolak jadwal yang tumpang-tindih dengan jadwal pegawai lainnya, atau yang
// membuat delegasi berantai (pengganti sedang mendelegasikan, atau pegawai sedang menjadi pengganti).
func checkConflict(ctx context.Context, st Store, companyCode string, p period, exceptID int64) error {
	c, err := st.FindConflict(ctx, companyCode, p, exceptID)
	if err != nil || c == nil {
		return err
	}
	when := fmt.Sprintf("%s s.d. %s", c.From, c.Until)
	switch c.Kind {
	case "OWN":
		return &apperr.InvalidStateError{Message: fmt.Sprintf(
			"%s sudah punya jadwal delegasi yang beririsan (%s, pengganti %s). Ubah jadwal itu atau pilih tanggal lain.",
			c.OwnerName, when, c.TargetName)}
	case "DELEGATE_AWAY":
		return &apperr.InvalidStateError{Message: fmt.Sprintf(
			"%s sedang mendelegasikan tugasnya ke %s pada %s, jadi tidak bisa menjadi pengganti di periode itu.",
			c.OwnerName, c.TargetName, when)}
	default:
		return &apperr.InvalidStateError{Message: fmt.Sprintf(
			"%s sedang menjadi pengganti %s pada %s. Ubah delegasi %s lebih dulu supaya delegasi tidak berantai.",
			c.TargetName, c.OwnerName, when, c.OwnerName)}
	}
}

func (s *Service) find(ctx context.Context, companyCode string, id int64) (*Delegation, error) {
	d, err := s.read.Find(ctx, companyCode, id, s.today(), false)
	if errors.Is(err, db.ErrNotFound) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	s.localizeOne(d)
	return d, nil
}

// delegationSnapshot adalah isi audit_logs.snapshot untuk perubahan delegasi.
type delegationSnapshot struct {
	UserID     int64   `json:"userId"`
	DelegateID int64   `json:"delegateId"`
	From       string  `json:"from"`
	Until      string  `json:"until"`
	Note       *string `json:"note,omitempty"`
}

func snapshotOf(d *Delegation) *delegationSnapshot {
	if d == nil {
		return nil
	}
	return &delegationSnapshot{UserID: d.User.ID, DelegateID: d.Delegate.ID, From: d.From, Until: d.Until, Note: d.Note}
}

func (s *Service) record(ctx context.Context, q db.DBTX, user *auth.CurrentUser, id int64, action string, before, after *Delegation) error {
	return s.audit(q).Record(ctx, audit.Entry{
		CompanyCode: user.Company.Code,
		EntityType:  audit.EntityDelegation,
		EntityID:    id,
		Action:      action,
		ActorID:     user.ID,
		ActorName:   user.Name,
		Snapshot: struct {
			Before *delegationSnapshot `json:"before,omitempty"`
			After  *delegationSnapshot `json:"after,omitempty"`
		}{snapshotOf(before), snapshotOf(after)},
	})
}

func (s *Service) localize(items []Delegation) {
	for i := range items {
		s.localizeOne(&items[i])
	}
}

func (s *Service) localizeOne(d *Delegation) {
	d.CreatedAt = d.CreatedAt.In(s.loc)
	d.UpdatedAt = d.UpdatedAt.In(s.loc)
}

// parseID membaca ID positif dari string.
func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("ID tidak valid")
	}
	return id, nil
}
