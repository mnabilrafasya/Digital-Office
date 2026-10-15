package repository

import "testing"

func TestContainsPattern(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{`50%_[a]\b`, `%50\%\_\[a]\\b%`},
		{`agus`, `%agus%`},
		{``, `%%`},
		{`a]b`, `%a]b%`}, // "]" tidak perlu di-escape
		{`\\`, `%\\\\%`},
	}
	for _, tt := range tests {
		if got := ContainsPattern(tt.in); got != tt.want {
			t.Errorf("ContainsPattern(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
