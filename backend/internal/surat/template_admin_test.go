package surat

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidateNumberFormat(t *testing.T) {
	for _, ok := range []string{"{SEQ}/{COMP}/{MM_ROMAN}/{YYYY}", "ND-{SEQ}.{COMP}.{YYYY}", "{COMP}/{YYYY}/{SEQ}"} {
		if err := ValidateNumberFormat(ok); err != nil {
			t.Errorf("%q ditolak: %v", ok, err)
		}
	}
	bad := map[string]string{
		"":                           "kosong",
		"{SEQ}/{COMP}":               "tanpa tahun",
		"{SEQ}/{YYYY}":               "tanpa perusahaan",
		"{COMP}/{YYYY}":              "tanpa urutan",
		"{SEQ}/{UNIT}/{COMP}/{YYYY}": "token UNIT tidak ditawarkan",
		"{SEQ}/{COMP}/{YYYY}/{X}":    "token tidak dikenal",
		"{SEQ}/{COMP}/{YYYY}}":       "kurung kurawal liar",
		"{SEQ}/{COMP}/{YYYY}/é":      "bukan ASCII",
		"{SEQ}/{COMP}/{YYYY}" + strings.Repeat("x", 80): "terlalu panjang",
	}
	for format, why := range bad {
		if err := ValidateNumberFormat(format); err == nil {
			t.Errorf("%s (%q) diterima", why, format)
		}
	}
	at := time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)
	if got := NumberPreview("{SEQ}/{COMP}/{MM_ROMAN}/{YYYY}", "PKG", at); got != "001/PKG/XI/2026" {
		t.Errorf("contoh nomor = %q", got)
	}
}

func TestSetTemplateStatus(t *testing.T) {
	refs := newFakeRefs()
	refs.templates = map[int64]*Template{3: {ID: 3, Name: "Nota Dinas", ObjectKey: "kunci"}}
	svc, audit := newTestLetterServiceWithAudit(&fakeLetterStore{}, refs)
	ctx := context.Background()
	off := false

	got, err := svc.SetTemplateStatus(ctx, andiUser, "3", &off)
	if err != nil || got.IsActive || !refs.inactive[3] {
		t.Fatalf("nonaktif: %+v %v", got, err)
	}
	if len(audit.entries) != 1 || audit.entries[0].Action != "TEMPLATE_STATUS" || audit.entries[0].EntityID != 3 {
		t.Errorf("audit salah: %+v", audit.entries)
	}
	// Status sama: tidak dicatat lagi
	if _, err := svc.SetTemplateStatus(ctx, andiUser, "3", &off); err != nil || len(audit.entries) != 1 {
		t.Errorf("ulang: err=%v audit=%d", err, len(audit.entries))
	}
	if _, err := svc.SetTemplateStatus(ctx, andiUser, "99", &off); !errors.Is(err, errTemplateNotFound) {
		t.Errorf("template tidak ada: %v", err)
	}
	if _, err := svc.SetTemplateStatus(ctx, andiUser, "3", nil); err == nil {
		t.Error("active kosong diterima")
	}

	if err := svc.DeleteTemplate(ctx, andiUser, "3"); err != nil {
		t.Fatalf("hapus template: %v", err)
	}
	if _, ok := refs.templates[3]; ok {
		t.Error("template masih ada setelah dihapus")
	}
	if err := svc.DeleteTemplate(ctx, andiUser, "3"); !errors.Is(err, errTemplateNotFound) {
		t.Errorf("hapus template yang sudah tidak ada: %v", err)
	}
}
