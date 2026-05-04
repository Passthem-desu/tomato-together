package middleware

import (
	"io"
	"net/http"
	"strings"
)

const (
	maxBodySize = 1 << 20 // 1MB
)

// SecurityHeaders adds common security HTTP response headers
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; connect-src 'self' blob:; media-src 'self' blob: data:; img-src 'self' data: blob:; font-src 'self'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// MaxBytesReader limits the size of incoming request bodies to prevent memory exhaustion.
func MaxBytesReader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip GET/HEAD/OPTIONS requests (no body)
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		// SSE streams can have long connections but no body
		if strings.Contains(r.URL.Path, "/sse") {
			next.ServeHTTP(w, r)
			return
		}
		r.Body = io.NopCloser(io.LimitReader(r.Body, maxBodySize))
		next.ServeHTTP(w, r)
	})
}
