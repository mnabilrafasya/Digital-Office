package surat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Template naskah: file .docx di MinIO (bucket templates) dan datanya di tabel letter_templates.
// Semua pemegang TEMPLATE_CREATE (seluruh role) boleh membuat dan mengganti template milik
// perusahaannya. File lama tidak dihapus saat diganti, supaya bisa dipulihkan dari jejak audit.

// Batas isi template (mengikuti kolom letter_templates).
const (
	maxTemplateNameLength        = 150
	maxTemplateDescriptionLength = 500
)

// TemplateInput adalah isi form template apa adanya dari request multipart.
type TemplateInput struct {
	ID           string  // kosong saat membuat
	Name         *string // wajib
	Description  *string
	LetterTypeID *string // kosong = berlaku untuk semua jenis surat
	FileName     string
	Data         []byte // nil saat mengubah = file tidak diganti
	ClientIP     string // dicatat di audit
}

// templateSnapshot adalah isi kolom audit_logs.snapshot saat template dibuat atau diganti.
type templateSnapshot struct {
	Name              string  `json:"name"`
	Description       *string `json:"description,omitempty"`
	LetterTypeID      *int64  `json:"letterTypeId,string,omitempty"`
	FileName          string  `json:"fileName,omitempty"` // kosong = file tidak diganti
	FileSize          int64   `json:"fileSize,omitempty"`
	Checksum          string  `json:"checksum,omitempty"`
	PreviousObjectKey string  `json:"previousObjectKey,omitempty"` // file lama yang diganti (masih di MinIO)
}

// Templates mengambil template aktif milik perusahaan user. letterTypeID "" = semua jenis surat.
func (s *Service) Templates(ctx context.Context, user *auth.CurrentUser, letterTypeID string) ([]Template, error) {
	typeID, err := optionalID(&letterTypeID, "Jenis surat")
	if err != nil {
		return nil, err
	}
	items, err := s.repos.Templates.ListTemplates(ctx, user.Company.Code, typeID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		s.localTemplate(&items[i])
	}
	return items, nil
}

// OpenTemplate membuka file .docx template untuk diunduh. Pemanggil wajib menutup Body.
func (s *Service) OpenTemplate(ctx context.Context, user *auth.CurrentUser, templateID string) (*DocumentFile, error) {
	t, err := s.findTemplate(ctx, user, templateID)
	if err != nil {
		return nil, err
	}
	body, size, err := s.repos.Files.OpenTemplate(ctx, t.ObjectKey)
	if err != nil {
		return nil, err
	}
	return &DocumentFile{Name: downloadName(t.Name, ".docx"), ContentType: DocxContentType, Size: size, Body: body}, nil
}

// CreateTemplate menyimpan template baru beserta file .docx-nya.
func (s *Service) CreateTemplate(ctx context.Context, user *auth.CurrentUser, in TemplateInput) (*Template, error) {
	if !user.HasPermission(auth.TemplateCreate) {
		return nil, apperr.ErrForbidden
	}
	d, err := s.parseTemplate(ctx, user.Company.Code, in, 0)
	if err != nil {
		return nil, err
	}
	if in.Data == nil {
		return nil, apperr.Invalid("File template wajib diunggah")
	}
	fileName, err := parseDocx(in.FileName, in.Data, "template")
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(in.Data)

	var id int64
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		// Diulang saat deadlock: objek lama tertinggal tanpa baris template (tidak pernah dirujuk)
		key := uuid.NewString()
		if err := s.repos.Files.PutTemplate(ctx, key, in.Data, DocxContentType); err != nil {
			return err
		}
		var err error
		if id, err = s.repos.TemplateWriter(q).InsertTemplate(ctx, user.Company.Code, *d, key, user.ID); err != nil {
			return err
		}
		return s.repos.Audit(q).Record(ctx, templateEntry(user, id, audit.ActionTemplateCreate, in.ClientIP, templateSnapshot{
			Name: d.Name, Description: d.Description, LetterTypeID: d.LetterTypeID,
			FileName: fileName, FileSize: int64(len(in.Data)), Checksum: hex.EncodeToString(sum[:]),
		}))
	})
	if err != nil {
		return nil, err
	}
	return s.templateByID(ctx, user, id)
}

// UpdateTemplate mengganti data template; file .docx ikut diganti bila diunggah.
func (s *Service) UpdateTemplate(ctx context.Context, user *auth.CurrentUser, in TemplateInput) (*Template, error) {
	if !user.HasPermission(auth.TemplateCreate) {
		return nil, apperr.ErrForbidden
	}
	id, err := templateID(in.ID)
	if err != nil {
		return nil, err
	}
	if _, err := s.repos.Templates.FindTemplate(ctx, user.Company.Code, id); err != nil {
		return nil, templateNotFoundAs(err)
	}
	d, err := s.parseTemplate(ctx, user.Company.Code, in, id)
	if err != nil {
		return nil, err
	}
	snap := templateSnapshot{Name: d.Name, Description: d.Description, LetterTypeID: d.LetterTypeID}
	if in.Data != nil {
		if snap.FileName, err = parseDocx(in.FileName, in.Data, "template"); err != nil {
			return nil, err
		}
		sum := sha256.Sum256(in.Data)
		snap.FileSize, snap.Checksum = int64(len(in.Data)), hex.EncodeToString(sum[:])
	}

	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		writer := s.repos.TemplateWriter(q)
		old, err := writer.LockTemplate(ctx, user.Company.Code, id)
		if err != nil {
			return templateNotFoundAs(err)
		}
		key := ""
		if in.Data != nil {
			key = uuid.NewString()
			if err := s.repos.Files.PutTemplate(ctx, key, in.Data, DocxContentType); err != nil {
				return err
			}
			snap.PreviousObjectKey = old.ObjectKey
		}
		if err := writer.UpdateTemplate(ctx, id, *d, key); err != nil {
			return err
		}
		return s.repos.Audit(q).Record(ctx, templateEntry(user, id, audit.ActionTemplateUpdate, in.ClientIP, snap))
	})
	if err != nil {
		return nil, err
	}
	return s.templateByID(ctx, user, id)
}

// parseTemplate memvalidasi nama, deskripsi, dan jenis surat template. exceptID adalah template
// yang sedang diubah (0 saat membuat), supaya namanya sendiri tidak dianggap dobel.
func (s *Service) parseTemplate(ctx context.Context, companyCode string, in TemplateInput, exceptID int64) (*TemplateData, error) {
	name, err := optionalText(in.Name, maxTemplateNameLength, "Nama template")
	if err != nil {
		return nil, err
	}
	if name == nil {
		return nil, apperr.Invalid("Nama template wajib diisi")
	}
	d := TemplateData{Name: *name}
	if d.Description, err = optionalText(in.Description, maxTemplateDescriptionLength, "Deskripsi template"); err != nil {
		return nil, err
	}
	if d.LetterTypeID, err = optionalID(in.LetterTypeID, "Jenis surat"); err != nil {
		return nil, err
	}

	if d.LetterTypeID != nil {
		categories, err := s.repos.Lookups.LookupCategories(ctx, companyCode, []int64{*d.LetterTypeID})
		if err != nil {
			return nil, err
		}
		if categories[*d.LetterTypeID] != "LETTER_TYPE" {
			return nil, apperr.Invalid("Jenis surat tidak ditemukan atau tidak aktif")
		}
	}
	taken, err := s.repos.Templates.TemplateNameTaken(ctx, companyCode, d.Name, exceptID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, errTemplateNameTaken
	}
	return &d, nil
}

// findTemplate mencari template aktif milik perusahaan user dari ID di URL.
func (s *Service) findTemplate(ctx context.Context, user *auth.CurrentUser, rawID string) (*Template, error) {
	id, err := templateID(rawID)
	if err != nil {
		return nil, err
	}
	return s.templateByID(ctx, user, id)
}

func (s *Service) templateByID(ctx context.Context, user *auth.CurrentUser, id int64) (*Template, error) {
	t, err := s.repos.Templates.FindTemplate(ctx, user.Company.Code, id)
	if err != nil {
		return nil, templateNotFoundAs(err)
	}
	s.localTemplate(t)
	return t, nil
}

// localTemplate mengubah waktu template ke zona waktu klien.
func (s *Service) localTemplate(t *Template) {
	t.CreatedAt, t.UpdatedAt = t.CreatedAt.In(s.loc), t.UpdatedAt.In(s.loc)
}

// templateEntry menyiapkan baris audit untuk satu template.
func templateEntry(user *auth.CurrentUser, id int64, action, ip string, snapshot templateSnapshot) audit.Entry {
	return audit.Entry{
		CompanyCode: user.Company.Code,
		EntityType:  audit.EntityTemplate,
		EntityID:    id,
		Action:      action,
		ActorID:     user.ID,
		ActorName:   user.Name,
		IPAddress:   ip,
		Snapshot:    snapshot,
	}
}

// templateID memvalidasi ID template dari URL; ID tidak valid dijawab 404 seperti template tidak ada.
func templateID(raw string) (int64, error) {
	id, err := parseID(raw, "Template")
	if err != nil {
		return 0, errTemplateNotFound
	}
	return id, nil
}

// templateNotFoundAs mengubah db.ErrNotFound menjadi errTemplateNotFound; error lain diteruskan.
func templateNotFoundAs(err error) error {
	if errors.Is(err, db.ErrNotFound) {
		return errTemplateNotFound
	}
	return err
}

// downloadName membuat nama file unduhan dari teks bebas: karakter yang dilarang di nama file
// Windows diganti "-", lalu ext ditambahkan.
func downloadName(name, ext string) string {
	clean := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`\/:*?"<>|`, r) || r < 32 {
			return '-'
		}
		return r
	}, strings.TrimSpace(name))
	if clean == "" {
		clean = "dokumen"
	}
	return fmt.Sprintf("%s%s", clean, ext)
}
