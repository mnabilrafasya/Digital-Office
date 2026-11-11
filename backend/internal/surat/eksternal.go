package surat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/masterdata"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Surat eksternal.
//
// Keluar (Pusri → luar, jenis SURAT_EKSTERNAL): dibuat dan diproses seperti surat internal, tetapi
// tujuannya satu perusahaan/instansi (external_info) dan tidak memakai QR. Setelah approver terakhir
// setuju statusnya APPROVED: nomor terbit, editor membuat PDF cetak ber-watermark (resmi.go), surat
// dicetak dan ditandatangani basah, lalu approver terakhir mengunggah scan-nya (UploadSigned). Status
// menjadi SENT, scan menjadi PDF resmi, dan surat masuk inbox (tab eksternal) drafter, reviewer, dan
// approver sebagai arsip.
//
// Masuk (luar → Pusri, jenis SURAT_MASUK_EKSTERNAL): siapa pun mengunggah scan surat beserta data
// asalnya dan tujuan disposisi (orang dan/atau alias jabatan) dalam satu langkah (CreateIncoming).
// Surat tercatat dengan status RECEIVED, masuk inbox pengunggah (sudah didisposisikan) dan inbox
// semua tujuan (UNREAD), lalu mengikuti alur inbox biasa.

// Batas isian surat masuk eksternal.
const (
	maxIncomingSender    = 200
	maxIncomingReference = 100
	maxIncomingTargets   = 50
)

// signedSnapshot adalah isi audit_logs.snapshot saat scan bertanda tangan diunggah.
type signedSnapshot struct {
	VersionNumber int    `json:"versionNumber"`
	FileSize      int64  `json:"fileSize"`
	Pages         int    `json:"pages"`
	Checksum      string `json:"checksum"`
}

// UploadSigned menyimpan scan surat eksternal bertanda tangan basah (oleh approver terakhir, saat
// status APPROVED) sebagai PDF resmi, lalu surat menjadi SENT dan masuk inbox pesertanya sebagai
// arsip. Scan tidak diberi watermark lagi (watermark sudah tercetak di PDF cetaknya).
func (s *Service) UploadSigned(ctx context.Context, user *auth.CurrentUser, in FinalPDFInput) (*LetterDetail, error) {
	id, ok := parseLetterID(in.LetterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	d, err := s.detail(ctx, user, id)
	if err != nil {
		return nil, err
	}
	if d.Direction != masterdata.DirectionOutgoing || d.LetterNumber == nil {
		return nil, errNotSignable
	}
	pages, err := parseFinalPDF(ctx, in.Data)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(in.Data)
	snap := signedSnapshot{FileSize: int64(len(in.Data)), Pages: pages, Checksum: hex.EncodeToString(sum[:])}

	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		letters := s.repos.Letters(q)
		l, err := letters.LockByUUID(ctx, user.Company.Code, id)
		if err != nil {
			return notFoundAs(err)
		}
		reviewers, approvers, err := letters.Participants(ctx, l.ID)
		if err != nil {
			return err
		}
		state := State{Status: l.Status, DrafterID: l.DrafterID, AssigneeID: l.AssigneeID,
			Reviewers: reviewers, Approvers: approvers, External: l.TypeCode == masterdata.TypeExternalOut}
		if !CanUploadSigned(state, user.ID) {
			return errNotSignable
		}
		if !user.HasPermission(auth.LetterApprove) {
			return apperr.ErrForbidden
		}
		key := uuid.NewString()
		if err := s.repos.Files.PutDocument(ctx, key, in.Data, PDFContentType); err != nil {
			return err
		}
		if snap.VersionNumber, _, err = letters.AddVersion(ctx, NewVersion{
			LetterID: l.ID, Status: l.Status, Type: VersionFinal, ObjectKey: key,
			FileName: finalFileName(*d.LetterNumber), FileExtension: "pdf",
			FileSize: snap.FileSize, Checksum: snap.Checksum, CreatedBy: user.ID, CreatedByRole: RoleApprover,
		}); err != nil {
			return err
		}
		if err := letters.ApplyTransition(ctx, l.ID, user.ID, Decision{
			Action: UploadSigned, ActorRole: RoleApprover, From: StatusApproved, To: StatusSent,
		}, nil); err != nil {
			return err
		}
		archive := []int64{l.DrafterID}
		for _, id := range slices.Concat(reviewers, approvers) {
			if !slices.Contains(archive, id) {
				archive = append(archive, id)
			}
		}
		if err := s.repos.Inbox(q).DeliverDirect(ctx, l.ID, archive, nil); err != nil {
			return err
		}
		e := letterEntry(user, l.ID, id, audit.ActionSignedUpload, snap)
		e.IPAddress = in.ClientIP
		return s.repos.Audit(q).Record(ctx, e)
	})
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, user, id)
}

var errNotSignable = &apperr.InvalidStateError{
	Message: "Scan bertanda tangan hanya bisa diunggah approver terakhir surat eksternal yang sedang menunggu tanda tangan.",
}

// IncomingInput adalah isi form surat masuk eksternal apa adanya dari request.
type IncomingInput struct {
	Title           string   // perihal
	Sender          string   // pengirim (perusahaan/instansi)
	ReferenceNumber string   // nomor surat asal
	LetterDate      string   // tanggal surat asal, YYYY-MM-DD
	UserIDs         []string // tujuan disposisi: orang
	AliasLevelIDs   []string // tujuan disposisi: alias jabatan (diuraikan ke anggotanya)
	Note            *string  // catatan disposisi (opsional)
	FileName        string
	Data            []byte // scan PDF
	ClientIP        string
}

// incomingSnapshot adalah isi audit_logs.snapshot saat surat masuk eksternal dicatat.
type incomingSnapshot struct {
	Sender          string  `json:"sender"`
	ReferenceNumber string  `json:"referenceNumber"`
	LetterDate      string  `json:"letterDate"`
	UserIDs         []int64 `json:"userIds,omitempty"`
	AliasLevelIDs   []int64 `json:"aliasLevelIds,omitempty"`
	Recipients      []int64 `json:"recipients"`
	FileSize        int64   `json:"fileSize"`
	Checksum        string  `json:"checksum"`
}

// incomingData adalah isian surat masuk yang sudah divalidasi.
type incomingData struct {
	title, sender, reference, date string
	userIDs, aliasIDs              []int64
	note                           *string
}

// CreateIncoming mencatat surat masuk eksternal sekaligus mendisposisikannya ke tujuan.
func (s *Service) CreateIncoming(ctx context.Context, user *auth.CurrentUser, in IncomingInput) (*LetterDetail, error) {
	data, err := parseIncoming(in)
	if err != nil {
		return nil, err
	}
	recipients, err := s.incomingRecipients(ctx, user, data)
	if err != nil {
		return nil, err
	}
	if _, err := parseFinalPDF(ctx, in.Data); err != nil {
		return nil, err
	}
	typeID, err := s.repos.Lookups.LookupID(ctx, user.Company.Code, "LETTER_TYPE", masterdata.TypeExternalIn)
	if errors.Is(err, db.ErrNotFound) {
		return nil, &apperr.InvalidStateError{Message: "Jenis surat masuk eksternal belum tersedia; jalankan seed reference."}
	}
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(in.Data)
	snap := incomingSnapshot{Sender: data.sender, ReferenceNumber: data.reference, LetterDate: data.date,
		UserIDs: data.userIDs, AliasLevelIDs: data.aliasIDs, Recipients: recipients,
		FileSize: int64(len(in.Data)), Checksum: hex.EncodeToString(sum[:])}

	var id string
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		letters := s.repos.Letters(q)
		letterID, uid, err := letters.CreateIncoming(ctx, NewIncoming{
			CompanyCode: user.Company.Code, SenderUnitID: user.Unit.ID, UploaderID: user.ID,
			LetterTypeID: typeID, Title: data.title,
			External: ExternalInfo{Sender: data.sender, ReferenceNumber: data.reference, LetterDate: data.date},
		})
		if err != nil {
			return err
		}
		id = uid
		key := uuid.NewString()
		if err := s.repos.Files.PutDocument(ctx, key, in.Data, PDFContentType); err != nil {
			return err
		}
		if _, _, err := letters.AddVersion(ctx, NewVersion{
			LetterID: letterID, Status: StatusReceived, Type: VersionFinal, ObjectKey: key,
			FileName: incomingFileName(data.reference), FileExtension: "pdf",
			FileSize: snap.FileSize, Checksum: snap.Checksum, CreatedBy: user.ID, CreatedByRole: RoleDrafter,
		}); err != nil {
			return err
		}
		if err := s.repos.Inbox(q).DeliverIncoming(ctx, letterID, user.ID, user.Name, user.Position.Name,
			recipients, data.note); err != nil {
			return err
		}
		e := letterEntry(user, letterID, uid, audit.ActionIncoming, snap)
		e.IPAddress = in.ClientIP
		return s.repos.Audit(q).Record(ctx, e)
	})
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, user, id)
}

// parseIncoming memvalidasi isian surat masuk tanpa database.
func parseIncoming(in IncomingInput) (*incomingData, error) {
	var d incomingData
	required := []struct {
		dst   *string
		src   string
		max   int
		label string
	}{
		{&d.title, in.Title, maxTitleLength, "Perihal"},
		{&d.sender, in.Sender, maxIncomingSender, "Pengirim"},
		{&d.reference, in.ReferenceNumber, maxIncomingReference, "Nomor surat asal"},
	}
	for _, f := range required {
		v, err := optionalText(&f.src, f.max, f.label)
		if err != nil {
			return nil, err
		}
		if v == nil {
			return nil, apperr.Invalid(f.label + " wajib diisi")
		}
		*f.dst = *v
	}
	date, err := time.Parse(time.DateOnly, strings.TrimSpace(in.LetterDate))
	if err != nil {
		return nil, apperr.Invalid("Tanggal surat wajib diisi dengan format YYYY-MM-DD")
	}
	d.date = date.Format(time.DateOnly)
	if d.note, err = optionalText(in.Note, maxNoteLength, "Catatan disposisi"); err != nil {
		return nil, err
	}
	if d.userIDs, err = parseIDList(in.UserIDs, "Orang tujuan"); err != nil {
		return nil, err
	}
	if d.aliasIDs, err = parseIDList(in.AliasLevelIDs, "Alias tujuan"); err != nil {
		return nil, err
	}
	switch n := len(d.userIDs) + len(d.aliasIDs); {
	case n == 0:
		return nil, apperr.Invalid("Pilih minimal satu tujuan disposisi (orang atau alias jabatan)")
	case n > maxIncomingTargets:
		return nil, apperr.Invalid(fmt.Sprintf("Tujuan disposisi maksimal %d", maxIncomingTargets))
	}
	return &d, nil
}

// incomingRecipients menguraikan tujuan disposisi menjadi user aktif di perusahaan pengunggah:
// orang yang dipilih ditambah semua anggota alias, tanpa duplikat dan tanpa pengunggah sendiri.
func (s *Service) incomingRecipients(ctx context.Context, user *auth.CurrentUser, d *incomingData) ([]int64, error) {
	active, err := s.repos.Directory.ActiveUserIDs(ctx, user.Company.Code, d.userIDs)
	if err != nil {
		return nil, err
	}
	if !allIn(d.userIDs, active) {
		return nil, apperr.Invalid("Orang tujuan tidak ditemukan atau tidak aktif")
	}
	if len(d.aliasIDs) > 0 {
		categories, err := s.repos.Lookups.LookupCategories(ctx, user.Company.Code, d.aliasIDs)
		if err != nil {
			return nil, err
		}
		for _, id := range d.aliasIDs {
			if categories[id] != "POSITION_LEVEL" {
				return nil, apperr.Invalid("Alias tujuan tidak ditemukan atau tidak aktif")
			}
		}
	}
	members, err := s.repos.Directory.AliasMemberIDs(ctx, user.Company.Code, d.aliasIDs)
	if err != nil {
		return nil, err
	}
	out := []int64{}
	add := func(id int64) {
		if id != user.ID && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	for _, id := range d.userIDs {
		add(id)
	}
	for _, alias := range d.aliasIDs {
		if len(members[alias]) == 0 {
			return nil, apperr.Invalid("Alias tujuan belum punya anggota aktif")
		}
		for _, id := range members[alias] {
			add(id)
		}
	}
	if len(out) == 0 {
		return nil, apperr.Invalid("Tujuan disposisi tidak boleh hanya diri sendiri")
	}
	return out, nil
}

// parseIDList membaca daftar ID unik dari form ("" diabaikan).
func parseIDList(raw []string, label string) ([]int64, error) {
	var out []int64
	for _, r := range raw {
		if strings.TrimSpace(r) == "" {
			continue
		}
		id, err := parseID(r, label)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out, nil
}

// incomingFileName membentuk nama file scan surat masuk dari nomor surat asalnya.
func incomingFileName(reference string) string {
	return "Surat Masuk " + strings.NewReplacer("/", "-", `\`, "-").Replace(reference) + ".pdf"
}
