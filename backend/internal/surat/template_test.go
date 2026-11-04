package surat

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

// Semua role memegang TEMPLATE_CREATE; user tanpa permission itu dipakai untuk uji penolakan.
var (
	templateUser = &auth.CurrentUser{ID: 15, Name: "Andi", Company: auth.UserCompany{Code: "PSP"}, Permissions: []string{auth.TemplateCreate}}
	otherUser    = &auth.CurrentUser{ID: 9, Name: "Fajar", Company: auth.UserCompany{Code: "PSP"}, Permissions: []string{auth.TemplateCreate}}
)

func templateInput(name string) TemplateInput {
	return TemplateInput{Name: ptr(name), Description: ptr("  Format nota  "), LetterTypeID: ptr("12"),
		FileName: "nota.docx", Data: testDocx, ClientIP: "10.0.0.5"}
}

func TestCreateTemplate(t *testing.T) {
	refs := newFakeRefs()
	svc, audit := newTestLetterServiceWithAudit(&fakeLetterStore{}, refs)
	ctx := context.Background()

	tpl, err := svc.CreateTemplate(ctx, templateUser, templateInput("Nota Dinas"))
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	if tpl.Name != "Nota Dinas" || *tpl.Description != "Format nota" || tpl.LetterType == nil || tpl.LetterType.ID != 12 {
		t.Errorf("template salah: %+v", tpl)
	}
	files := svc.repos.Files.(*fakeFiles)
	if string(files.objects["template/"+tpl.ObjectKey]) != string(testDocx) {
		t.Errorf("file template tidak tersimpan di bucket templates: %v", files.objects)
	}
	e := audit.entries[0]
	snap := e.Snapshot.(templateSnapshot)
	if e.Action != "TEMPLATE_CREATE" || e.EntityType != "TEMPLATE" || e.EntityID != tpl.ID || e.IPAddress != "10.0.0.5" ||
		snap.FileName != "nota.docx" || len(snap.Checksum) != 64 {
		t.Errorf("audit salah: %+v", e)
	}

	isInvalid := func(want string) func(error) bool {
		return func(err error) bool {
			var ve *apperr.ValidationError
			return errors.As(err, &ve) && strings.Contains(ve.Message, want)
		}
	}
	for _, tc := range []struct {
		name  string
		user  *auth.CurrentUser
		edit  func(*TemplateInput)
		check func(error) bool
	}{
		{"tanpa permission", outsiderUser, nil, func(err error) bool { return errors.Is(err, apperr.ErrForbidden) }},
		{"nama kosong", templateUser, func(in *TemplateInput) { in.Name = ptr("  ") }, isInvalid("Nama template wajib")},
		{"nama terlalu panjang", templateUser, func(in *TemplateInput) { in.Name = ptr(strings.Repeat("a", 151)) }, isInvalid("maksimal 150")},
		{"tanpa file", templateUser, func(in *TemplateInput) { in.Data = nil }, isInvalid("wajib diunggah")},
		{"bukan docx", templateUser, func(in *TemplateInput) { in.FileName = "nota.pdf" }, isInvalid("wajib berupa file .docx")},
		{"isi bukan docx", templateUser, func(in *TemplateInput) { in.Data = []byte("teks") }, isInvalid("bukan dokumen .docx")},
		{"jenis surat bukan LETTER_TYPE", templateUser, func(in *TemplateInput) { in.LetterTypeID = ptr("40") }, isInvalid("Jenis surat tidak ditemukan")},
		{"nama dobel (beda huruf besar)", templateUser, func(in *TemplateInput) { in.Name = ptr("nota dinas") },
			func(err error) bool { return errors.Is(err, errTemplateNameTaken) }},
	} {
		in := templateInput("Template Lain")
		if tc.edit != nil {
			tc.edit(&in)
		}
		if _, err := svc.CreateTemplate(ctx, tc.user, in); !tc.check(err) {
			t.Errorf("%s: error = %v", tc.name, err)
		}
	}
	if len(refs.templates) != 1 || len(audit.entries) != 1 {
		t.Errorf("input ditolak tetap tersimpan: %d template, %d audit", len(refs.templates), len(audit.entries))
	}
}

func TestUpdateTemplateByAnyUser(t *testing.T) {
	refs := newFakeRefs()
	svc, audit := newTestLetterServiceWithAudit(&fakeLetterStore{}, refs)
	ctx := context.Background()
	nota, _ := svc.CreateTemplate(ctx, templateUser, templateInput("Nota Dinas"))
	if _, err := svc.CreateTemplate(ctx, templateUser, templateInput("Undangan")); err != nil {
		t.Fatal(err)
	}
	oldKey := nota.ObjectKey
	id := "1"

	// Pengguna lain mengganti data saja: file tetap
	in := TemplateInput{ID: id, Name: ptr("Nota Dinas"), Description: ptr("Baru")}
	got, err := svc.UpdateTemplate(ctx, otherUser, in)
	if err != nil || got.ObjectKey != oldKey || *got.Description != "Baru" || got.LetterType != nil {
		t.Fatalf("ubah data: %+v err=%v", got, err)
	}
	if snap := audit.entries[2].Snapshot.(templateSnapshot); audit.entries[2].Action != "TEMPLATE_UPDATE" || snap.FileName != "" || snap.PreviousObjectKey != "" {
		t.Errorf("audit ubah data salah: %+v", audit.entries[2])
	}

	// Ganti file: objek baru, objek lama tetap ada untuk pemulihan
	newDocx := []byte("PK\x03\x04versi-baru")
	in = TemplateInput{ID: id, Name: ptr("Nota Dinas Internal"), FileName: "nota-v2.docx", Data: newDocx}
	if got, err = svc.UpdateTemplate(ctx, otherUser, in); err != nil || got.ObjectKey == oldKey || got.Name != "Nota Dinas Internal" {
		t.Fatalf("ganti file: %+v err=%v", got, err)
	}
	files := svc.repos.Files.(*fakeFiles)
	if files.objects["template/"+oldKey] == nil {
		t.Error("file lama ikut terhapus")
	}
	if snap := audit.entries[3].Snapshot.(templateSnapshot); snap.PreviousObjectKey != oldKey || snap.FileName != "nota-v2.docx" {
		t.Errorf("audit ganti file salah: %+v", snap)
	}
	f, err := svc.OpenTemplate(ctx, outsiderUser, id) // semua user perusahaan boleh mengunduh
	if err != nil {
		t.Fatalf("OpenTemplate: %v", err)
	}
	body, _ := io.ReadAll(f.Body)
	if string(body) != string(newDocx) || f.Name != "Nota Dinas Internal.docx" || f.ContentType != DocxContentType {
		t.Errorf("unduhan salah: %q %+v", body, f)
	}

	isNotFound := func(err error) bool { return errors.Is(err, errTemplateNotFound) }
	for _, tc := range []struct {
		name  string
		in    TemplateInput
		check func(error) bool
	}{
		{"nama milik template lain", TemplateInput{ID: id, Name: ptr("Undangan")}, func(err error) bool { return errors.Is(err, errTemplateNameTaken) }},
		{"template tidak ada", TemplateInput{ID: "99", Name: ptr("X")}, isNotFound},
		{"id bukan angka", TemplateInput{ID: "abc", Name: ptr("X")}, isNotFound},
		{"file kosong", TemplateInput{ID: id, Name: ptr("X"), FileName: "a.docx", Data: []byte{}},
			func(err error) bool { var ve *apperr.ValidationError; return errors.As(err, &ve) }},
	} {
		if _, err := svc.UpdateTemplate(ctx, otherUser, tc.in); !tc.check(err) {
			t.Errorf("%s: error = %v", tc.name, err)
		}
	}
	if _, err := svc.OpenTemplate(ctx, otherUser, "99"); !isNotFound(err) {
		t.Errorf("unduh template tidak ada: error = %v", err)
	}
}

func TestDownloadName(t *testing.T) {
	for in, want := range map[string]string{
		"Surat Perintah Tugas (SPT)": "Surat Perintah Tugas (SPT).docx",
		`A/B:C*"D"`:                  "A-B-C--D-.docx",
		"   ":                        "dokumen.docx",
	} {
		if got := downloadName(in, ".docx"); got != want {
			t.Errorf("downloadName(%q) = %q, want %q", in, got, want)
		}
	}
}
