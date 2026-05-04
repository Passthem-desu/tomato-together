package middleware

import (
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
}

type visitor struct {
	count    int
	lastSeen time.Time
}

const (
	cleanupInterval = 5 * time.Minute
	generalLimit    = 60
	authLimit       = 10
	windowDuration  = 1 * time.Minute
)

var globalLimiter = &rateLimiter{
	visitors: make(map[string]*visitor),
}

func init() {
	go globalLimiter.cleanup()
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(cleanupInterval)
	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > windowDuration {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) allow(ip string, limit int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists || now.Sub(v.lastSeen) > windowDuration {
		rl.visitors[ip] = &visitor{count: 1, lastSeen: now}
		return true
	}

	v.lastSeen = now
	v.count++
	return v.count <= limit
}

func getIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Use the first IP in the chain (client's original IP)
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// RateLimit returns a middleware that limits requests per endpoint.
// The endpoint key is derived from the request path pattern.
func RateLimit(limit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)
			if !globalLimiter.allow(ip+"|"+r.URL.Path, limit) {
				http.Error(w, `{"error":"rate_limit_exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitPerRoute returns a middleware that rate limits based on a custom route key
func RateLimitPerRoute(limit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)
			if !globalLimiter.allow(ip, limit) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate_limit_exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GeneralRateLimit is 60 req/min per IP
func GeneralRateLimit() func(http.Handler) http.Handler {
	return RateLimitPerRoute(generalLimit)
}

// AuthRateLimit is 10 req/min per IP for sensitive endpoints
func AuthRateLimit() func(http.Handler) http.Handler {
	return RateLimitPerRoute(authLimit)
}
