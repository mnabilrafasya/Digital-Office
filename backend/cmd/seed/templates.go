package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/google/uuid"

	"office.backend/database"
	"office.backend/internal/platform/config"
	"office.backend/internal/platform/storage"
	"office.backend/internal/surat"
)

// companyTemplates adalah template bawaan per perusahaan (nama file di database/seeds/templates).
// Template berkop Pusri Palembang hanya untuk PSP; perusahaan lain mendapat dokumen kosong.
var companyTemplates = []struct {
	company string
	files   []string
}{
	{"PSP", []string{"surat-dinas-standar.docx", "nota-dinas.docx", "undangan-rapat.docx", "surat-tugas.docx", "surat-external.docx", "dokumen-kosong.docx"}},
	{"PKG", []string{"dokumen-kosong.docx"}},
}

// defaultTemplates adalah template naskah bawaan (file di database/seeds/templates). letterType ""
// = berlaku untuk semua jenis surat.
var defaultTemplates = []struct {
	file, name, description, letterType string
}{
	{"surat-dinas-standar.docx", "Surat Dinas Resmi Standar",
		"Format surat resmi dengan kop PT Pupuk Sriwidjaja Palembang, nomor ###NoSurat###, Yth, tanda tangan, dan tembusan.", ""},
	{"nota-dinas.docx", "Nota Dinas Internal",
		"Format korespondensi internal (Yth, Dari, Tanggal, Perihal, Pembahasan, tanda tangan elektronik).", "NOTA_DINAS"},
	{"undangan-rapat.docx", "Surat Undangan Rapat",
		"Format undangan kegiatan atau rapat koordinasi dengan tabel hari, waktu, tempat, dan agenda.", "SURAT_UNDANGAN"},
	{"surat-tugas.docx", "Surat Perintah Tugas (SPT)",
		"Format penugasan kedinasan dengan tabel daftar pegawai dan pejabat penandatangan.", "SURAT_TUGAS"},
	{"surat-external.docx", "Surat Eksternal",
		"Format surat keluar kepada perusahaan atau instansi di luar perusahaan.", "SURAT_EKSTERNAL"},
	{"dokumen-kosong.docx", "Dokumen Kosong (Blank)",
		"Mulai dari halaman kosong A4 standar Word.", ""},
}

// seedTemplates mengunggah template bawaan setiap perusahaan ke MinIO dan mencatatnya di
// letter_templates. Idempoten: template yang namanya sudah ada (termasuk yang sudah diganti
// pengguna) dilewati. Pembuatnya dicatat sebagai admin aktif perusahaan; perusahaan tanpa admin
// dilewati.
func seedTemplates(ctx context.Context, conn *sql.DB, cfg *config.Config) error {
	files, err := storage.New(ctx, cfg)
	if err != nil {
		return err
	}
	repo := surat.NewRepository(conn)
	for _, ct := range companyTemplates {
		creator, err := templateCreator(ctx, conn, ct.company)
		if errors.Is(err, errNoAdmin) {
			fmt.Printf("Template %s dilewati: belum ada admin aktif.\n", ct.company)
			continue
		}
		if err != nil {
			return err
		}
		added := 0
		for _, file := range ct.files {
			t, ok := templateByFile(file)
			if !ok {
				return fmt.Errorf("template %s tidak terdaftar", file)
			}
			data, err := fs.ReadFile(database.TemplateSeeds, t.file)
			if err != nil {
				return fmt.Errorf("membaca template %s: %w", t.file, err)
			}
			taken, err := repo.TemplateNameTaken(ctx, ct.company, t.name, 0)
			if err != nil {
				return err
			}
			if taken {
				continue
			}
			d := surat.TemplateData{Name: t.name, Description: &t.description}
			if t.letterType != "" {
				id, err := letterTypeID(ctx, conn, t.letterType, ct.company)
				if err != nil {
					return err
				}
				d.LetterTypeID = &id
			}

			key := uuid.NewString()
			if err := files.PutTemplate(ctx, key, data, surat.DocxContentType); err != nil {
				return err
			}
			if _, err := repo.InsertTemplate(ctx, ct.company, d, key, creator); err != nil {
				return fmt.Errorf("menyimpan template %s %s: %w", ct.company, t.name, err)
			}
			added++
			fmt.Printf("  template/%s/%s\n", ct.company, t.file)
		}
		fmt.Printf("Seed template %s selesai: %d ditambah, %d sudah ada.\n", ct.company, added, len(ct.files)-added)
	}
	return nil
}

// templateByFile mencari template bawaan berdasarkan nama filenya.
func templateByFile(file string) (struct{ file, name, description, letterType string }, bool) {
	for _, t := range defaultTemplates {
		if t.file == file {
			return t, true
		}
	}
	return struct{ file, name, description, letterType string }{}, false
}

// errNoAdmin: perusahaan belum punya admin aktif untuk dicatat sebagai pembuat template.
var errNoAdmin = errors.New("belum ada admin aktif")

// templateCreator adalah user pembuat template bawaan: admin aktif perusahaan dengan id terkecil.
func templateCreator(ctx context.Context, conn *sql.DB, company string) (int64, error) {
	const q = `
SELECT TOP 1 u.id
FROM users u
JOIN roles r     ON r.id = u.role_id AND r.code = 'ADMIN'
JOIN positions p ON p.id = u.position_id
JOIN units un    ON un.id = p.unit_id AND un.company_code = @company
WHERE u.is_active = 1 AND u.deleted_at IS NULL
ORDER BY u.id`
	var id int64
	err := conn.QueryRowContext(ctx, q, sql.Named("company", company)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errNoAdmin
	}
	if err != nil {
		return 0, fmt.Errorf("mencari admin pembuat template: %w", err)
	}
	return id, nil
}

// letterTypeID mencari lookup LETTER_TYPE aktif berdasarkan kodenya.
func letterTypeID(ctx context.Context, conn *sql.DB, code, company string) (int64, error) {
	const q = `
SELECT id FROM lookups
WHERE category = 'LETTER_TYPE' AND code = @code AND is_active = 1 AND deleted_at IS NULL
  AND (company_code IS NULL OR company_code = @company)`
	var id int64
	err := conn.QueryRowContext(ctx, q, sql.Named("code", code), sql.Named("company", company)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("jenis surat %s belum ada; jalankan seed reference dulu", code)
	}
	if err != nil {
		return 0, fmt.Errorf("mencari jenis surat %s: %w", code, err)
	}
	return id, nil
}
