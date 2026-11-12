package surat

import (
	"context"
	"fmt"
	"time"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

// Delegasi tugas (tabel delegations, jadwalnya dikelola paket delegasi). Selama periode delegasi
// pegawai A ke B:
//   - tugas review/approval A hanya bisa dikerjakan B, atas nama dan dengan kewenangan A; A tetap
//     tercatat sebagai peserta (giliran tidak berubah) tetapi tidak bisa bertindak;
//   - riwayat mencatat B sebagai pelaku. "Atas nama" tidak disimpan terpisah: penerima tugas
//     sebelum suatu aksi adalah target aksi alur sebelumnya, jadi pelaku yang berbeda berarti
//     pengganti (markOnBehalf);
//   - surat yang masuk ke inbox A juga ditembuskan ke B (InboxStore.CopyToDelegates).
// Setelah periode selesai tugas kembali ke A; B tetap bisa membuka surat yang pernah ia kerjakan.

// delegable: status yang tugasnya dikerjakan reviewer/approver, sehingga bisa didelegasikan.
func delegable(status string) bool {
	role := AssigneeRole(status)
	return role == RoleReviewer || role == RoleApprover
}

// today adalah tanggal hari ini (YYYY-MM-DD) di zona waktu aplikasi.
func (s *Service) today() string {
	return time.Now().In(s.loc).Format(time.DateOnly)
}

// assigneeDelegation adalah delegasi yang sedang berlaku untuk penerima tugas surat berstatus
// status, atau nil bila tugasnya tidak sedang didelegasikan.
func (s *Service) assigneeDelegation(ctx context.Context, letters Store, companyCode, status string, assigneeID int64) (*Delegation, error) {
	if assigneeID == 0 || !delegable(status) {
		return nil, nil
	}
	return letters.ActiveDelegation(ctx, companyCode, assigneeID, s.today())
}

// actingFor menentukan atas nama siapa user bertindak pada surat l: dirinya sendiri, atau penerima
// tugas yang sedang mendelegasikan tugasnya kepada user (onBehalf). Penerima tugas yang sedang
// mendelegasikan tidak boleh bertindak.
func (s *Service) actingFor(ctx context.Context, letters Store, user *auth.CurrentUser, l *LetterLock) (actorID int64, onBehalf bool, err error) {
	del, err := s.assigneeDelegation(ctx, letters, user.Company.Code, l.Status, l.AssigneeID)
	if err != nil || del == nil {
		return user.ID, false, err
	}
	switch user.ID {
	case del.Delegate.ID:
		return l.AssigneeID, true, nil
	case l.AssigneeID:
		return 0, false, delegatedAway(del)
	}
	return user.ID, false, nil
}

// delegatedAway adalah penolakan untuk penerima tugas yang sedang mendelegasikan tugasnya.
func delegatedAway(d *Delegation) error {
	until := d.Until
	if t, err := time.Parse(time.DateOnly, d.Until); err == nil {
		until = tanggalIndonesia(t)
	}
	return &apperr.InvalidStateError{Message: fmt.Sprintf(
		"Tugas Anda sedang didelegasikan ke %s sampai %s, jadi surat ini dikerjakan pengganti Anda.", d.Delegate.Name, until)}
}

// checkRolePermission memastikan pelaku berwenang untuk perannya. Pengganti memakai kewenangan
// orang yang diwakilinya (actorID), jadi siapa pun boleh menjadi pengganti.
func (s *Service) checkRolePermission(ctx context.Context, user *auth.CurrentUser, actorID int64, onBehalf bool, role string) error {
	perm := rolePermission(role)
	if perm == "" {
		return nil
	}
	if !onBehalf {
		if !user.HasPermission(perm) {
			return apperr.ErrForbidden
		}
		return nil
	}
	ok, err := s.repos.Directory.EligibleUserIDs(ctx, user.Company.Code, perm, []int64{actorID})
	if err != nil {
		return err
	}
	if !ok[actorID] {
		return &apperr.InvalidStateError{Message: "Pegawai yang Anda wakili tidak lagi berwenang memproses surat ini. Hubungi admin."}
	}
	return nil
}

// visibleToNonParticipant: user yang bukan drafter/peserta tetap boleh membuka surat bila ia
// pengganti penerima tugas saat ini, atau pernah bertindak atas surat itu (sebagai pengganti).
func (s *Service) visibleToNonParticipant(ctx context.Context, companyCode string, letterID int64, status string, assigneeID, userID int64) (bool, error) {
	del, err := s.assigneeDelegation(ctx, s.reader, companyCode, status, assigneeID)
	if err != nil {
		return false, err
	}
	if del != nil && del.Delegate.ID == userID {
		return true, nil
	}
	return s.reader.HasActed(ctx, letterID, userID)
}

// markOnBehalf mengisi OnBehalfOf pada riwayat (urut dari yang paling lama): aksi oleh orang selain
// penerima tugas saat itu dilakukan sebagai pengganti penerima tugas tersebut. Penerima tugas
// adalah target aksi alur sebelumnya (READ dan SAVE tidak memindahkan surat).
func markOnBehalf(actions []LetterAction) {
	var holder *PersonRef
	for i := range actions {
		a := &actions[i]
		if holder != nil && a.Action != "READ" && a.Action != string(Submit) && a.Actor.ID != holder.ID {
			h := *holder
			a.OnBehalfOf = &h
		}
		if a.Action != "READ" && a.Action != "SAVE" {
			holder = a.Target
		}
	}
}
