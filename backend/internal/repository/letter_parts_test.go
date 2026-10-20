package repository

import (
	"slices"
	"testing"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

func TestPlanParticipants(t *testing.T) {
	rev := func(id, userID int64, step int) participant {
		return participant{ID: id, UserID: userID, Role: model.RoleReviewer, Step: step}
	}
	appr := func(id, userID int64) participant {
		return participant{ID: id, UserID: userID, Role: model.RoleApprover, Step: 1}
	}
	current := []participant{rev(1, 9, 1), rev(2, 4, 2), appr(3, 3)}

	tests := []struct {
		name       string
		want       []participant
		wantDelete []int64
		wantInsert []participant
	}{
		{"tidak berubah", []participant{rev(0, 9, 1), rev(0, 4, 2), appr(0, 3)}, nil, nil},
		{"urutan reviewer ditukar", []participant{rev(0, 4, 1), rev(0, 9, 2), appr(0, 3)},
			[]int64{1, 2}, []participant{rev(0, 4, 1), rev(0, 9, 2)}},
		{"approver diganti", []participant{rev(0, 9, 1), rev(0, 4, 2), appr(0, 5)},
			[]int64{3}, []participant{appr(0, 5)}},
		{"reviewer terakhir dihapus", []participant{rev(0, 9, 1), appr(0, 3)}, []int64{2}, nil},
		{"semua dikosongkan", nil, []int64{1, 2, 3}, nil},
	}
	for _, tt := range tests {
		gotDelete, gotInsert := planParticipants(current, tt.want)
		if !slices.Equal(gotDelete, tt.wantDelete) || !slices.Equal(gotInsert, tt.wantInsert) {
			t.Errorf("%s: delete=%v insert=%v, want delete=%v insert=%v", tt.name, gotDelete, gotInsert, tt.wantDelete, tt.wantInsert)
		}
	}
}

func TestPlanRecipients(t *testing.T) {
	current := []recipient{
		{ID: 1, Type: model.RecipientTo, UnitID: 7, Sort: 0},
		{ID: 2, Type: model.RecipientTo, AliasID: 60, Sort: 1},
		{ID: 3, Type: model.RecipientCC, UnitID: 9, Sort: 2},
	}
	tests := []struct {
		name       string
		want       []recipient
		wantDelete []int64
		wantUpdate []recipient
		wantInsert []recipient
	}{
		{"tidak berubah", []recipient{
			{Type: model.RecipientTo, UnitID: 7, Sort: 0},
			{Type: model.RecipientTo, AliasID: 60, Sort: 1},
			{Type: model.RecipientCC, UnitID: 9, Sort: 2},
		}, nil, nil, nil},
		{"TO dan CC bertukar, unit baru, alias dihapus", []recipient{
			{Type: model.RecipientTo, UnitID: 9, Sort: 0},
			{Type: model.RecipientCC, UnitID: 7, Sort: 1},
			{Type: model.RecipientCC, UnitID: 11, Sort: 2},
		}, []int64{2}, []recipient{
			{ID: 3, Type: model.RecipientTo, UnitID: 9, Sort: 0},
			{ID: 1, Type: model.RecipientCC, UnitID: 7, Sort: 1},
		}, []recipient{{Type: model.RecipientCC, UnitID: 11, Sort: 2}}},
		{"unit dan alias dengan ID sama dibedakan", []recipient{
			{Type: model.RecipientTo, AliasID: 7, Sort: 0},
		}, []int64{1, 2, 3}, nil, []recipient{{Type: model.RecipientTo, AliasID: 7, Sort: 0}}},
	}
	for _, tt := range tests {
		gotDelete, gotUpdate, gotInsert := planRecipients(current, tt.want)
		if !slices.Equal(gotDelete, tt.wantDelete) || !slices.Equal(gotUpdate, tt.wantUpdate) || !slices.Equal(gotInsert, tt.wantInsert) {
			t.Errorf("%s:\n delete=%v update=%v insert=%v\n want delete=%v update=%v insert=%v",
				tt.name, gotDelete, gotUpdate, gotInsert, tt.wantDelete, tt.wantUpdate, tt.wantInsert)
		}
	}
}
