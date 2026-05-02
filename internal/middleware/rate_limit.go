package middleware

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RateLimitOptions struct {
	Name        string
	MaxRequests int
	Window      time.Duration
	ScopeTenant bool
	ScopeUser   bool
}

type rateLimitBucket struct {
	Count     int
	ResetAt   time.Time
	UpdatedAt time.Time
}

type InMemoryRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]rateLimitBucket
}

func NewInMemoryRateLimiter() *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		buckets: make(map[string]rateLimitBucket),
	}
}

func (rl *InMemoryRateLimiter) Middleware(opts RateLimitOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			maxReq := opts.MaxRequests
			window := opts.Window
			if maxReq <= 0 || window <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			key := rl.buildKey(r, opts)
			allowed, retryAfter := rl.allow(key, maxReq, window)
			if !allowed {
				if retryAfter > 0 {
					w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
				}
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(struct {
					Error   string `json:"error"`
					Message string `json:"message"`
				}{Error: "too_many_requests", Message: "Too many requests"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rl *InMemoryRateLimiter) buildKey(r *http.Request, opts RateLimitOptions) string {
	parts := []string{strings.TrimSpace(opts.Name), clientIP(r)}

	if opts.ScopeTenant {
		if tenantID, ok := r.Context().Value(TenantIDKey).(uint64); ok && tenantID > 0 {
			parts = append(parts, fmt.Sprintf("tenant:%d", tenantID))
		}
	}
	if opts.ScopeUser {
		if userID, err := GetUserIDFromContext(r.Context()); err == nil && userID > 0 {
			parts = append(parts, fmt.Sprintf("user:%d", userID))
		}
	}

	return strings.Join(parts, "|")
}

func (rl *InMemoryRateLimiter) allow(key string, maxReq int, window time.Duration) (bool, time.Duration) {
	now := time.Now().UTC()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.buckets[key]
	if !exists || !now.Before(b.ResetAt) {
		rl.buckets[key] = rateLimitBucket{
			Count:     1,
			ResetAt:   now.Add(window),
			UpdatedAt: now,
		}
		return true, 0
	}

	if b.Count >= maxReq {
		return false, b.ResetAt.Sub(now)
	}

	b.Count++
	b.UpdatedAt = now
	rl.buckets[key] = b
	return true, 0
}

func clientIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		ip := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if ip != "" {
			return ip
		}
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && strings.TrimSpace(host) != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
