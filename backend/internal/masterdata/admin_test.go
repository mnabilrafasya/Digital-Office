package masterdata

import (
	"context"
	"strings"
	"testing"

	"office.backend/internal/auth"
)

func TestEditableLookupCategories(t *testing.T) {
	for _, category := range []string{"LETTER_TYPE", "CLASSIFICATION"} {
		if err := editableCategory(category); err != nil {
			t.Fatal(err)
		}
	}
	for _, category := range []string{
		"CONFIDENTIALITY",
		"URGENCY",
		"LETTER_STATUS",
		"INBOX_STATUS",
		"POSITION_LEVEL",
		"",
		"LETTER_TYPE;DROP TABLE lookups",
	} {
		if editableCategory(category) == nil {
			t.Fatalf("kategori sistem boleh diedit: %s", category)
		}
	}
}

func TestParseLookup(t *testing.T) {
	good, err := parseLookup(LookupInput{
		Code:      " abc-01 ",
		Name:      " Arsip Umum ",
		Note:      " Contoh ",
		SortOrder: 3,
	}, true)
	if err != nil || good.Code != "ABC-01" || good.Name != "Arsip Umum" || good.Note != "Contoh" {
		t.Fatalf("normalisasi: %+v %v", good, err)
	}

	invalidInputs := []LookupInput{
		{Code: "A", Name: ""},
		{Code: "bad code", Name: "Valid"},
		{Code: strings.Repeat("A", 31), Name: "Valid"},
		{Code: "A", Name: strings.Repeat("字", 101)},
		{Code: "A", Name: "Valid", SortOrder: -1},
		{Code: "A", Name: "Valid", SortOrder: 10000},
		{Code: "A", Name: "Valid", Note: strings.Repeat("字", 2001)},
	}
	for _, input := range invalidInputs {
		if _, err := parseLookup(input, true); err == nil {
			t.Fatalf("input tidak ditolak: %+v", input)
		}
	}

	if _, err := parseLookup(LookupInput{Name: "Ganti nama"}, false); err != nil {
		t.Fatal("kode tidak wajib pada edit", err)
	}
}

func TestStandardLookupCannotBeMutated(t *testing.T) {
	if manageableLookup(&AdminLookup{ReadOnly: true}) == nil {
		t.Fatal("data global boleh diubah")
	}
	if err := manageableLookup(&AdminLookup{}); err != nil {
		t.Fatal(err)
	}
}

func TestLookupPermissionCheckedBeforeDatabase(t *testing.T) {
	service := &AdminService{}
	user := &auth.CurrentUser{}
	if _, err := service.List(context.Background(), user, "LETTER_TYPE", "", ""); err == nil {
		t.Fatal("list tanpa permission")
	}
	if _, err := service.Save(context.Background(), user, "LETTER_TYPE", "", LookupInput{}); err == nil {
		t.Fatal("create tanpa permission")
	}
	if _, err := service.SetStatus(context.Background(), user, "LETTER_TYPE", "1", nil); err == nil {
		t.Fatal("status tanpa permission")
	}
	if err := service.Delete(context.Background(), user, "LETTER_TYPE", "1"); err == nil {
		t.Fatal("delete tanpa permission")
	}
}
