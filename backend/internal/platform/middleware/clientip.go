// Package middleware berisi middleware HTTP teknis yang tidak bergantung pada aturan bisnis.
package middleware

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type clientIPKey struct{}

// ClientIP menentukan IP klien dan menyimpannya di context (ambil dengan ClientIPFrom).
// Header X-Forwarded-For / X-Real-IP hanya dipercaya bila trustProxy=true, yaitu saat server
// berada di belakang reverse proxy yang menimpa header tersebut. Tanpa itu, IP diambil dari
// koneksi TCP, sehingga pembatasan login tidak bisa dilewati dengan memalsukan header.
func ClientIP(trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := remoteIP(r.RemoteAddr)
			if trustProxy {
				if fwd := forwardedIP(r.Header); fwd != "" {
					ip = fwd
					// Supaya log request menampilkan IP klien yang sebenarnya
					r.RemoteAddr = net.JoinHostPort(ip, "0")
				}
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey{}, ip)))
		})
	}
}

// ClientIPFrom mengambil IP klien dari context ("" bila middleware ClientIP tidak dipasang).
func ClientIPFrom(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}

func remoteIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return normalizeIP(host)
}

// forwardedIP mengambil IP pertama yang valid dari X-Forwarded-For, lalu X-Real-IP.
func forwardedIP(h http.Header) string {
	if xff := h.Get("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		if ip := normalizeIP(strings.TrimSpace(first)); ip != "" {
			return ip
		}
	}
	return normalizeIP(strings.TrimSpace(h.Get("X-Real-IP")))
}

// normalizeIP mengembalikan bentuk baku IP (IPv4 yang dibungkus IPv6 diubah ke IPv4), atau "".
func normalizeIP(s string) string {
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return ""
	}
	return addr.Unmap().String()
}
