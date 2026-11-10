package auth

import (
	"slices"
	"testing"
)

func TestMergePermissions(t *testing.T) {
	pegawai := []string{LetterReview, LetterCreate, TemplateCreate}
	cases := []struct {
		level string
		want  []string
	}{
		{"VP", []string{InboxDispose, LetterApprove, LetterCreate, LetterReview, TemplateCreate}},
		{"avp", []string{InboxDispose, LetterApprove, LetterCreate, LetterReview, TemplateCreate}},
		{"DIREKTUR", []string{InboxDispose, LetterApprove, LetterCreate, LetterReview, TemplateCreate}},
		{"STAF", []string{LetterCreate, LetterReview, TemplateCreate}},
		{"", []string{LetterCreate, LetterReview, TemplateCreate}},
	}
	for _, tc := range cases {
		if got := mergePermissions(pegawai, tc.level); !slices.Equal(got, tc.want) {
			t.Errorf("posisi %q: %v, want %v", tc.level, got, tc.want)
		}
	}
	// Tidak ganda bila role kebetulan juga punya permission yang sama; nil menjadi daftar kosong
	if got := mergePermissions([]string{LetterApprove}, "SVP"); !slices.Equal(got, []string{InboxDispose, LetterApprove}) {
		t.Errorf("gabungan ganda: %v", got)
	}
	if got := mergePermissions(nil, ""); got == nil || len(got) != 0 {
		t.Errorf("tanpa permission: %#v, want []", got)
	}
}

func TestPositionRules(t *testing.T) {
	if !GrantedByPosition(LetterApprove) || !GrantedByPosition(InboxDispose) || GrantedByPosition(LetterReview) {
		t.Error("hanya approve dan disposisi yang datang dari posisi")
	}
	levels := OfficialLevels()
	levels[0] = "STAF" // salinan, tidak mengubah aturan
	if IsOfficialLevel("STAF") || !IsOfficialLevel("DIREKTUR") {
		t.Error("OfficialLevels harus mengembalikan salinan")
	}
}
