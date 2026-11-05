package surat

import (
	"slices"

	"office.backend/internal/auth"
	"office.backend/internal/notifikasi"
)

// transitionEvents menentukan pemberitahuan WhatsApp untuk satu aksi alur:
//   - Ajukan/Setujui (reviewer): penerima tugas berikutnya, "menunggu review" atau "menunggu
//     persetujuan" sesuai perannya.
//   - Kembalikan: penerima tugas berikutnya (drafter atau reviewer), dengan catatannya.
//   - Tanda tangan (approver): drafter ("surat terbit") dan semua penerima inbox ("surat masuk").
func transitionEvents(user *auth.CurrentUser, letterID, drafterID int64, dec Decision, note *string, pub published) []notifikasi.Event {
	event := func(kind string, recipients []int64, note *string) notifikasi.Event {
		return notifikasi.Event{Kind: kind, LetterID: letterID, Recipients: recipients,
			ActorID: user.ID, ActorName: user.Name, Note: note}
	}
	switch dec.Action {
	case Submit, Approve:
		kind := notifikasi.EventReviewRequested
		if dec.AssigneeRole == RoleApprover {
			kind = notifikasi.EventApprovalRequested
		}
		return []notifikasi.Event{event(kind, []int64{dec.AssigneeID}, nil)}
	case Return:
		return []notifikasi.Event{event(notifikasi.EventReturned, []int64{dec.AssigneeID}, note)}
	case Sign:
		// Drafter yang juga penerima cukup mendapat pesan "terbit"
		received := slices.DeleteFunc(append(slices.Clone(pub.to), pub.cc...), func(id int64) bool { return id == drafterID })
		return []notifikasi.Event{
			event(notifikasi.EventPublished, []int64{drafterID}, nil),
			event(notifikasi.EventReceived, received, nil),
		}
	}
	return nil
}
