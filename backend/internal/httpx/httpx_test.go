package httpx

import (
	"bytes"
	"errors"
	"net/url"
	"testing"
)

func TestETagRoundTrip(t *testing.T) {
	rv := []byte{0, 0, 0, 0, 0, 0, 0x07, 0xD3}
	tag := FormatETag(rv)
	if tag != `"00000000000007D3"` {
		t.Fatalf("FormatETag = %s", tag)
	}
	got, present, err := ParseIfMatch(tag)
	if err != nil || !present || !bytes.Equal(got, rv) {
		t.Fatalf("ParseIfMatch(%s) = %x, %v, %v", tag, got, present, err)
	}
	// huruf kecil tetap diterima
	if got, _, err := ParseIfMatch(`"00000000000007d3"`); err != nil || !bytes.Equal(got, rv) {
		t.Fatalf("huruf kecil: %x, %v", got, err)
	}
}

func TestParseIfMatchInvalid(t *testing.T) {
	if _, present, err := ParseIfMatch("  "); present || err != nil {
		t.Errorf("kosong: present=%v err=%v, want false nil", present, err)
	}
	for _, h := range []string{`*`, `W/"00000000000007D3"`, `00000000000007D3`, `"XYZ"`, `"0007D3"`, `"00000000000007D3", "00000000000007D4"`} {
		if _, present, err := ParseIfMatch(h); !present || !errors.Is(err, ErrInvalidIfMatch) {
			t.Errorf("%q: present=%v err=%v, want true ErrInvalidIfMatch", h, present, err)
		}
	}
}

func TestParsePage(t *testing.T) {
	tests := []struct {
		query   string
		want    Page
		wantErr bool
	}{
		{"", Page{1, 20}, false},
		{"page=3&pageSize=50", Page{3, 50}, false},
		{"pageSize=100", Page{1, 100}, false},
		{"page=0", Page{}, true},
		{"page=abc", Page{}, true},
		{"pageSize=101", Page{}, true},
		{"pageSize=0", Page{}, true},
	}
	for _, tt := range tests {
		q, _ := url.ParseQuery(tt.query)
		got, err := ParsePage(q)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q: err = %v, wantErr %v", tt.query, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("%q: got %+v, want %+v", tt.query, got, tt.want)
		}
	}
	if (Page{Page: 3, PageSize: 20}).Offset() != 40 {
		t.Error("Offset halaman 3 ukuran 20 harus 40")
	}
}
