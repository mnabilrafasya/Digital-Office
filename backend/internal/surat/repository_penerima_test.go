package surat

import (
	"slices"
	"testing"
)

func TestPlanRecipients(t *testing.T) {
	current := []recipient{
		{ID: 1, Type: RecipientTo, UnitID: 7, Sort: 0},
		{ID: 2, Type: RecipientTo, AliasID: 60, Sort: 1},
		{ID: 3, Type: RecipientCC, UnitID: 9, Sort: 2},
	}
	tests := []struct {
		name       string
		want       []recipient
		wantDelete []int64
		wantUpdate []recipient
		wantInsert []recipient
	}{
		{"tidak berubah", []recipient{
			{Type: RecipientTo, UnitID: 7, Sort: 0},
			{Type: RecipientTo, AliasID: 60, Sort: 1},
			{Type: RecipientCC, UnitID: 9, Sort: 2},
		}, nil, nil, nil},
		{"TO dan CC bertukar, unit baru, alias dihapus", []recipient{
			{Type: RecipientTo, UnitID: 9, Sort: 0},
			{Type: RecipientCC, UnitID: 7, Sort: 1},
			{Type: RecipientCC, UnitID: 11, Sort: 2},
		}, []int64{2}, []recipient{
			{ID: 3, Type: RecipientTo, UnitID: 9, Sort: 0},
			{ID: 1, Type: RecipientCC, UnitID: 7, Sort: 1},
		}, []recipient{{Type: RecipientCC, UnitID: 11, Sort: 2}}},
		{"unit dan alias dengan ID sama dibedakan", []recipient{
			{Type: RecipientTo, AliasID: 7, Sort: 0},
		}, []int64{1, 2, 3}, nil, []recipient{{Type: RecipientTo, AliasID: 7, Sort: 0}}},
	}
	for _, tt := range tests {
		gotDelete, gotUpdate, gotInsert := planRecipients(current, tt.want)
		if !slices.Equal(gotDelete, tt.wantDelete) || !slices.Equal(gotUpdate, tt.wantUpdate) || !slices.Equal(gotInsert, tt.wantInsert) {
			t.Errorf("%s:\n delete=%v update=%v insert=%v\n want delete=%v update=%v insert=%v",
				tt.name, gotDelete, gotUpdate, gotInsert, tt.wantDelete, tt.wantUpdate, tt.wantInsert)
		}
	}
}
