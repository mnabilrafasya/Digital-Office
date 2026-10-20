package service

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/workflow"
)

// maxNoteLength adalah panjang maksimal catatan aksi (karakter).
const maxNoteLength = 1000

// TransitionInput adalah satu aksi alur persetujuan dari klien.
type TransitionInput struct {
	Action   workflow.Action
	Note     *string
	ClientIP string
}

// Transition menjalankan aksi SUBMIT, APPROVE, RETURN, atau SIGN. Seluruh langkah berjalan dalam
// satu transaksi (diulang otomatis bila deadlock): kunci baris surat, hitung transisi lewat
// workflow.Decide, terapkan perubahan, lalu tulis letter_actions dan audit_logs.
func (s *LetterService) Transition(ctx context.Context, user *model.CurrentUser, uuid string, in TransitionInput) (*LetterResult, error) {
	note, err := normalizeNote(in.Action, in.Note)
	if err != nil {
		return nil, err
	}

	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		r := s.repos(q)
		lc, err := loadVisible(ctx, r, user, uuid, true)
		if err != nil {
			return err
		}
		d, err := workflow.Decide(lc.state(), in.Action, user.ID)
		if err != nil {
			return mapWorkflowError(err)
		}
		if !hasRolePermission(user, d.ActorRole) {
			return ErrForbidden
		}
		if d.Action == workflow.ActionSubmit {
			if err := s.validateSubmit(ctx, r, user, lc); err != nil {
				return err
			}
		}
		if err := ensureAssigneeActive(ctx, r, user.Company.Code, d.NextAssigneeID); err != nil {
			return err
		}
		return s.applyDecision(ctx, r, user, lc, d, note, in.ClientIP)
	})
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, user, uuid, false)
}

func normalizeNote(action workflow.Action, note *string) (*string, error) {
	n := cleanText(note)
	switch action {
	case workflow.ActionReturn:
		if n == nil {
			return nil, invalid("Catatan wajib diisi saat mengembalikan surat")
		}
	case workflow.ActionApprove:
	default:
		return nil, nil // SUBMIT dan SIGN tidak membawa catatan
	}
	if n != nil && utf8.RuneCountInString(*n) > maxNoteLength {
		return nil, invalid("Catatan maksimal %d karakter", maxNoteLength)
	}
	return n, nil
}

// mapWorkflowError mengubah error mesin status menjadi error service (409 INVALID_STATE).
func mapWorkflowError(err error) error {
	switch {
	case errors.Is(err, workflow.ErrInvalidState):
		return conflict(msgAlreadyProcessed)
	case errors.Is(err, workflow.ErrNotAllowed):
		return conflict(msgNotYourTurn)
	case errors.Is(err, workflow.ErrIncomplete):
		return invalid("Minimal 1 reviewer dan tepat 1 approver wajib dipilih sebelum surat dikirim")
	default:
		return err
	}
}

// validateSubmit memastikan draf lengkap, semua referensi masih valid, dan semua penerima bisa
// diterjemahkan menjadi orang (unit punya pimpinan aktif, alias punya anggota).
func (s *LetterService) validateSubmit(ctx context.Context, r LetterRepos, user *model.CurrentUser, lc *letterCtx) error {
	recips, err := r.Recipients.List(ctx, lc.letter.ID)
	if err != nil {
		return err
	}
	if err := validateDraft(ctx, r, user, draftFromStored(lc.letter, lc.parts, recips), lc.letter.DrafterID, true); err != nil {
		return err
	}
	_, _, err = resolveRecipients(ctx, r, user.Company.Code, recips)
	return err
}

// ensureAssigneeActive memastikan orang yang akan menerima tugas berikutnya masih aktif.
func ensureAssigneeActive(ctx context.Context, r LetterRepos, company string, userID int64) error {
	if userID == 0 {
		return nil
	}
	users, err := r.Users.FindParticipantUsers(ctx, company, []int64{userID}, permission.LetterReview, permission.LetterApprove)
	if err != nil {
		return err
	}
	if u, ok := users[userID]; !ok || !u.Eligible {
		name := "Penerima tugas berikutnya"
		if ok {
			name = u.Name
		}
		return conflict("Surat tidak dapat diteruskan karena %s sudah tidak aktif. Hubungi administrator.", name)
	}
	return nil
}

// resolveRecipients menerjemahkan penerima menjadi user: pimpinan aktif untuk penerima unit dan
// anggota untuk penerima alias. Penerima yang tidak bisa diterjemahkan menghasilkan 409.
func resolveRecipients(ctx context.Context, r LetterRepos, company string, recips []model.Recipient) (map[int64]int64, map[int64][]int64, error) {
	var unitIDs, aliasIDs []int64
	for _, rc := range recips {
		if rc.UnitID != nil {
			unitIDs = append(unitIDs, *rc.UnitID)
		}
		if rc.AliasLevelID != nil {
			aliasIDs = append(aliasIDs, *rc.AliasLevelID)
		}
	}

	heads, err := r.Recipients.ResolveUnitHeads(ctx, company, unitIDs)
	if err != nil {
		return nil, nil, err
	}
	members, err := r.Recipients.AliasMembers(ctx, company, aliasIDs)
	if err != nil {
		return nil, nil, err
	}

	var missingUnits, missingAliases []int64
	for _, id := range unitIDs {
		if _, ok := heads[id]; !ok {
			missingUnits = append(missingUnits, id)
		}
	}
	for _, id := range aliasIDs {
		if len(members[id]) == 0 {
			missingAliases = append(missingAliases, id)
		}
	}
	if len(missingUnits) == 0 && len(missingAliases) == 0 {
		return heads, members, nil
	}

	var msgs []string
	if len(missingUnits) > 0 {
		units, err := r.Units.FindRefs(ctx, company, missingUnits)
		if err != nil {
			return nil, nil, err
		}
		var names []string
		for _, id := range missingUnits {
			names = append(names, units[id].Name)
		}
		msgs = append(msgs, "unit tujuan belum punya pimpinan aktif: "+strings.Join(names, ", "))
	}
	if len(missingAliases) > 0 {
		aliases, err := r.Lookups.FindAliases(ctx, company, missingAliases)
		if err != nil {
			return nil, nil, err
		}
		var names []string
		for _, id := range missingAliases {
			names = append(names, "Seluruh "+aliases[id].Name)
		}
		msgs = append(msgs, "alias penerima tidak punya anggota: "+strings.Join(names, ", "))
	}
	return nil, nil, conflict("Surat tidak dapat diproses karena %s.", strings.Join(msgs, "; "))
}

// inboxEntries menghitung penerima akhir, satu baris per user. Bila seseorang terkena TO dan CC,
// yang dipakai TO.
func inboxEntries(recips []model.Recipient, heads map[int64]int64, members map[int64][]int64) []model.InboxEntry {
	types := map[int64]string{}
	add := func(userID int64, typ string) {
		if cur, ok := types[userID]; !ok || (cur == model.RecipientCC && typ == model.RecipientTo) {
			types[userID] = typ
		}
	}
	for _, rc := range recips {
		if rc.UnitID != nil {
			add(heads[*rc.UnitID], rc.Type)
		} else if rc.AliasLevelID != nil {
			for _, u := range members[*rc.AliasLevelID] {
				add(u, rc.Type)
			}
		}
	}
	out := make([]model.InboxEntry, 0, len(types))
	for u, t := range types {
		out = append(out, model.InboxEntry{UserID: u, RecipientType: t})
	}
	slices.SortFunc(out, func(a, b model.InboxEntry) int { return cmp.Compare(a.UserID, b.UserID) })
	return out
}

func (s *LetterService) applyDecision(ctx context.Context, r LetterRepos, user *model.CurrentUser, lc *letterCtx, d workflow.Decision, note *string, ip string) error {
	l := lc.letter
	t := model.LetterTransition{
		Status:             string(d.To),
		AssigneeID:         idPtr(d.NextAssigneeID),
		ForwardedBy:        idChange(d.ForwardedBy),
		ReturnedByReviewer: idChange(d.ReturnedByReviewer),
		ReturnedByApprover: idChange(d.ReturnedByApprover),
	}
	if d.SnapshotSenderUnit {
		units, err := r.Units.FindRefs(ctx, user.Company.Code, []int64{l.SenderUnitID})
		if err != nil {
			return err
		}
		name := units[l.SenderUnitID].Name
		t.SenderUnitName = &name
	}
	if d.Sign {
		sign, err := s.signLetter(ctx, r, user, l)
		if err != nil {
			return err
		}
		t.Sign = sign
	}

	if err := r.Letters.ApplyTransition(ctx, l.ID, t); err != nil {
		return err
	}
	version := l.CurrentVersion
	if err := r.Actions.Insert(ctx, model.ActionEntry{
		LetterID: l.ID, ActorID: user.ID, TargetUserID: t.AssigneeID, ActorRole: string(d.ActorRole),
		Action: string(d.Action), FromStatus: string(d.From), ToStatus: string(d.To),
		Note: note, VersionNumber: &version,
	}); err != nil {
		return err
	}

	snapshot := map[string]any{"from": d.From, "to": d.To, "assigneeId": nil, "letterNumber": nil}
	if t.AssigneeID != nil {
		snapshot["assigneeId"] = strconv.FormatInt(*t.AssigneeID, 10)
	}
	if t.Sign != nil {
		snapshot["letterNumber"] = t.Sign.LetterNumber
	}
	snap, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("menyusun snapshot audit: %w", err)
	}
	return r.Audit.Insert(ctx, model.AuditEntry{
		CompanyCode: l.CompanyCode, EntityType: "LETTER", EntityID: l.ID, EntityUUID: l.UUID,
		Action: string(d.Action), ActorID: user.ID, ActorName: user.Name, Snapshot: string(snap), IPAddress: ip,
	})
}

// signLetter memberi nomor surat dan mengirim surat ke inbox penerima. Kunci penomoran
// (UPDLOCK, HOLDLOCK) diambil setelah baris surat terkunci, sesuai urutan kunci yang disepakati.
func (s *LetterService) signLetter(ctx context.Context, r LetterRepos, user *model.CurrentUser, l *model.Letter) (*model.LetterSign, error) {
	company := user.Company.Code
	recips, err := r.Recipients.List(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	heads, members, err := resolveRecipients(ctx, r, company, recips)
	if err != nil {
		return nil, err
	}

	format, err := r.Numbering.LetterNumberFormat(ctx, company)
	if err != nil {
		return nil, err
	}
	units, err := r.Units.FindRefs(ctx, company, []int64{l.SenderUnitID})
	if err != nil {
		return nil, err
	}
	now := s.now().In(s.cfg.Location)
	seq, err := r.Numbering.NextSequence(ctx, l.SenderUnitID, now.Year())
	if err != nil {
		return nil, err
	}
	number, err := workflow.FormatLetterNumber(format, workflow.NumberParts{
		Seq: seq, UnitCode: units[l.SenderUnitID].Code, CompanyCode: company, Month: now.Month(), Year: now.Year(),
	})
	if err != nil {
		return nil, fmt.Errorf("format nomor surat perusahaan %s: %w", company, err)
	}

	resolved := map[int64]int64{}
	for _, rc := range recips {
		if rc.UnitID != nil {
			resolved[rc.ID] = heads[*rc.UnitID]
		}
	}
	if err := r.Recipients.SetResolvedHeads(ctx, resolved); err != nil {
		return nil, err
	}
	if err := r.Inbox.InsertDirect(ctx, l.ID, inboxEntries(recips, heads, members)); err != nil {
		return nil, err
	}
	return &model.LetterSign{LetterNumber: number, NumberSeq: seq, NumberYear: now.Year(), SignedByID: user.ID}, nil
}

func idPtr(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}

func idChange(u workflow.IDUpdate) model.IDChange {
	return model.IDChange{Set: u.Set, Value: idPtr(u.ID)}
}
