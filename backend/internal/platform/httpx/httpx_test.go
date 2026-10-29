package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestETagRoundTrip(t *testing.T) {
	rv := []byte{0, 0, 0, 0, 0, 0, 0x07, 0xD3}
	tag := FormatETag(rv)
	if tag != `"00000000000007D3"` {
		t.Fatalf("FormatETag = %s", tag)
	}
	for _, h := range []string{tag, `"00000000000007d3"`, "  " + tag + " "} {
		got, present, err := ParseIfMatch(h)
		if err != nil || !present || !bytes.Equal(got, rv) {
			t.Errorf("ParseIfMatch(%q) = %x, %v, %v", h, got, present, err)
		}
	}
}

func TestParseIfMatchInvalid(t *testing.T) {
	if _, present, err := ParseIfMatch("  "); present || err != nil {
		t.Errorf("kosong: present=%v err=%v, want false nil", present, err)
	}
	for _, h := range []string{`*`, `W/"00000000000007D3"`, `00000000000007D3`, `"XYZ"`, `"0007D3"`,
		`"00000000000007D3", "00000000000007D4"`} {
		if _, present, err := ParseIfMatch(h); !present || !errors.Is(err, ErrInvalidIfMatch) {
			t.Errorf("%q: present=%v err=%v, want true ErrInvalidIfMatch", h, present, err)
		}
	}
}

func TestParsePage(t *testing.T) {
	tests := []struct {
		query   string
		want    Page
		wantErr error
	}{
		{"", Page{1, 20}, nil},
		{"page=3&pageSize=50", Page{3, 50}, nil},
		{"pageSize=100", Page{1, 100}, nil},
		{"page=0", Page{}, ErrInvalidPage},
		{"page=abc", Page{}, ErrInvalidPage},
		{"pageSize=101", Page{}, ErrInvalidPageSize},
		{"pageSize=0", Page{}, ErrInvalidPageSize},
	}
	for _, tt := range tests {
		q, _ := url.ParseQuery(tt.query)
		got, err := ParsePage(q)
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("%q: err = %v, want %v", tt.query, err, tt.wantErr)
			continue
		}
		if err == nil && got != tt.want {
			t.Errorf("%q: got %+v, want %+v", tt.query, got, tt.want)
		}
	}
	if (Page{Page: 3, PageSize: 20}).Offset() != 40 {
		t.Error("Offset halaman 3 ukuran 20 harus 40")
	}
}

func TestTooManyRequests(t *testing.T) {
	rec := httptest.NewRecorder()
	TooManyRequests(rec, 1500*time.Millisecond, "tunggu")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "2" {
		t.Errorf("status=%d Retry-After=%q, want 429 dan 2", rec.Code, rec.Header().Get("Retry-After"))
	}
}

func TestDecodeJSON(t *testing.T) {
	type body struct {
		A string `json:"a"`
	}
	tests := []struct {
		name      string
		input     string
		optional  bool
		wantOK    bool
		wantInMsg string
	}{
		{"valid", `{"a":"x"}`, false, true, ""},
		{"kosong wajib", ``, false, false, "kosong"},
		{"kosong opsional", ``, true, true, ""},
		{"rusak", `{"a":`, false, false, "tidak valid"},
		{"lebih dari 1 MB", `{"a":"` + strings.Repeat("x", MaxBodyBytes) + `"}`, false, false, "terlalu besar"},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.input))
		var dst body
		decode := DecodeJSON
		if tt.optional {
			decode = DecodeOptionalJSON
		}
		if ok := decode(rec, req, &dst); ok != tt.wantOK {
			t.Errorf("%s: ok = %v, want %v", tt.name, ok, tt.wantOK)
			continue
		}
		if !tt.wantOK {
			var resp struct {
				Error errorBody `json:"error"`
			}
			_ = json.Unmarshal(rec.Body.Bytes(), &resp)
			if rec.Code != http.StatusBadRequest || !strings.Contains(resp.Error.Message, tt.wantInMsg) {
				t.Errorf("%s: status=%d pesan=%q", tt.name, rec.Code, resp.Error.Message)
			}
		}
	}
}
