package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		trust      bool
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{"tanpa proxy", false, "203.0.113.5:51000", nil, "203.0.113.5"},
		{"header palsu diabaikan", false, "203.0.113.5:51000", map[string]string{"X-Forwarded-For": "1.2.3.4"}, "203.0.113.5"},
		{"proxy dipercaya: XFF pertama", true, "10.0.0.2:443", map[string]string{"X-Forwarded-For": "198.51.100.7, 10.0.0.1"}, "198.51.100.7"},
		{"proxy dipercaya: X-Real-IP", true, "10.0.0.2:443", map[string]string{"X-Real-IP": "198.51.100.8"}, "198.51.100.8"},
		{"proxy dipercaya: header rusak", true, "10.0.0.2:443", map[string]string{"X-Forwarded-For": "bukan-ip"}, "10.0.0.2"},
		{"IPv6 loopback", false, "[::1]:51000", nil, "::1"},
		{"IPv4 dalam IPv6", false, "[::ffff:192.0.2.1]:51000", nil, "192.0.2.1"},
	}
	for _, tt := range tests {
		var got string
		h := ClientIP(tt.trust)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			got = ClientIPFrom(r.Context())
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = tt.remoteAddr
		for k, v := range tt.headers {
			req.Header.Set(k, v)
		}
		h.ServeHTTP(httptest.NewRecorder(), req)
		if got != tt.want {
			t.Errorf("%s: IP = %q, want %q", tt.name, got, tt.want)
		}
	}
}
