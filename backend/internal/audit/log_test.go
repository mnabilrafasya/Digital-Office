package audit

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

var admin = &auth.CurrentUser{ID: 13, Name: "Rudi", Company: auth.UserCompany{Code: "PSP"}}

func TestLogFilter(t *testing.T) {
	wib := time.FixedZone("WIB", 7*3600)
	s := &Service{loc: wib}

	f, err := s.logFilter(admin, LogQuery{From: "2026-11-01", To: "2026-11-03", ActorID: "8",
		Action: " submit, SIGN ,", EntityType: "letter", Search: "  015/TI "})
	if err != nil {
		t.Fatal(err)
	}
	if !f.From.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, wib)) || !f.To.Equal(time.Date(2026, 11, 4, 0, 0, 0, 0, wib)) {
		t.Errorf("rentang = %v .. %v, want 1 Nov 00:00 WIB .. 4 Nov 00:00 WIB (to ikut dihitung)", f.From, f.To)
	}
	if f.CompanyCode != "PSP" || f.ViewerID != 13 || f.ActorID != 8 || strings.Join(f.Actions, ",") != "SUBMIT,SIGN" ||
		f.EntityType != "LETTER" || f.Search != "015/TI" {
		t.Errorf("filter = %+v", f)
	}

	// Tanpa from/to: 7 hari terakhir sampai akhir hari ini
	f, _ = s.logFilter(admin, LogQuery{})
	if days := f.To.Sub(f.From).Hours() / 24; days != 7 {
		t.Errorf("rentang default = %v hari, want 7", days)
	}

	for _, lq := range []LogQuery{
		{From: "01-11-2026"}, {To: "kemarin"}, {From: "2026-11-05", To: "2026-11-01"},
		{ActorID: "abc"}, {ActorID: "0"}, {Action: "DROP TABLE"}, {Search: strings.Repeat("a", 101)},
	} {
		var ve *apperr.ValidationError
		if _, err := s.logFilter(admin, lq); !errors.As(err, &ve) {
			t.Errorf("%+v: error = %v, want ValidationError", lq, err)
		}
	}
}

func TestExportValidation(t *testing.T) {
	s := &Service{loc: time.UTC} // repo tidak dipakai: validasi gagal lebih dulu
	for _, lq := range []LogQuery{
		{},                                     // rentang wajib
		{From: "2026-10-01"},                   // to wajib
		{From: "2026-10-01", To: "2026-11-05"}, // lebih dari 31 hari
	} {
		var ve *apperr.ValidationError
		if err := s.Export(context.Background(), admin, lq, &bytes.Buffer{}); !errors.As(err, &ve) {
			t.Errorf("%+v: error = %v, want ValidationError", lq, err)
		}
	}
}

func TestMonitorValidation(t *testing.T) {
	s := &Service{loc: time.UTC, now: time.Now}
	for _, mq := range []MonitorQuery{{Status: "SENT"}, {Status: "DRAFT"}, {MinWaitingHours: "-1"}, {MinWaitingHours: "abc"}} {
		mq.Page, mq.PageSize = 1, 20
		var ve *apperr.ValidationError
		if _, _, err := s.Monitor(context.Background(), admin, mq); !errors.As(err, &ve) {
			t.Errorf("%+v: error = %v, want ValidationError", mq, err)
		}
	}
}
