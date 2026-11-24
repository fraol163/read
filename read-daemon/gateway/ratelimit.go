package gateway

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// RateLimitBurst is the maximum burst capacity for non-whitelisted IPs.
	RateLimitBurst = 10.0
	// RateLimitSustainedPerSec is the refill rate of tokens per second.
	RateLimitSustainedPerSec = 2.0
)

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

// IPRateLimiter manages per-IP token buckets with localhost whitelisting.
type IPRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
}

// NewIPRateLimiter creates a new rate limiter.
func NewIPRateLimiter() *IPRateLimiter {
	return &IPRateLimiter{
		buckets: make(map[string]*tokenBucket),
	}
}

// isLoopback checks if an IP is a local loopback address.
func isLoopback(ipStr string) bool {
	if ipStr == "localhost" || ipStr == "127.0.0.1" || ipStr == "::1" || strings.HasPrefix(ipStr, "127.") {
		return true
	}
	parsed := net.ParseIP(ipStr)
	if parsed != nil && parsed.IsLoopback() {
		return true
	}
	return false
}

// extractClientIP extracts the remote client IP.
func extractClientIP(r *http.Request) string {
	// Respect X-Forwarded-For if set (e.g. for proxy simulation in chaos tests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		clientIP := strings.TrimSpace(parts[0])
		if clientIP != "" {
			return clientIP
		}
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Allow checks if the request from the client IP is allowed.
// Returns (allowed, retryAfterSeconds).
func (lim *IPRateLimiter) Allow(clientIP string) (bool, int) {
	if isLoopback(clientIP) {
		return true, 0
	}

	lim.mu.Lock()
	defer lim.mu.Unlock()

	now := time.Now()
	b, exists := lim.buckets[clientIP]
	if !exists {
		lim.buckets[clientIP] = &tokenBucket{
			tokens:     RateLimitBurst - 1.0,
			lastRefill: now,
		}
		return true, 0
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = math.Min(RateLimitBurst, b.tokens+elapsed*RateLimitSustainedPerSec)
	b.lastRefill = now

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true, 0
	}

	// Token bucket empty: compute Retry-After in seconds
	needed := 1.0 - b.tokens
	retryAfter := int(math.Ceil(needed / RateLimitSustainedPerSec))
	if retryAfter < 1 {
		retryAfter = 1
	}
	return false, retryAfter
}

// Middleware wraps an http.Handler with per-IP rate limiting.
func (lim *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := extractClientIP(r)
		allowed, retryAfter := lim.Allow(clientIP)
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":               "Rate limit exceeded. Too many requests.",
				"retry_after_seconds": retryAfter,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
