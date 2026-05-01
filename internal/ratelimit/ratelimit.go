package ratelimit

import (
	"net/http"
	"sync"
	"time"
)

type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     int           // max requests per interval
	interval time.Duration // time window
	whitelist []string
}

type bucket struct {
	count    int
	expiry   time.Time
}

func New(rate int, interval time.Duration, whitelist []string) *Limiter {
	return &Limiter{
		buckets:   make(map[string]*bucket),
		rate:      rate,
		interval:  interval,
		whitelist: whitelist,
	}
}

func (l *Limiter) Allow(r *http.Request) bool {
	ip := extractIP(r)

	// Check whitelist
	for _, w := range l.whitelist {
		if ip == w {
			return true
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[ip]
	if !ok || time.Now().After(b.expiry) {
		l.buckets[ip] = &bucket{count: 1, expiry: time.Now().Add(l.interval)}
		return true
	}

	b.count++
	if b.count > l.rate {
		return false
	}
	return true
}

func (l *Limiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for ip, b := range l.buckets {
		if now.After(b.expiry) {
			delete(l.buckets, ip)
		}
	}
}

func extractIP(r *http.Request) string {
	// Check X-Real-IP first (set by nginx)
	ip := r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}
	ip = r.Header.Get("X-Forwarded-For")
	if ip != "" {
		// Take the first IP in the list
		if idx := len(ip); idx > 0 {
			for i := 0; i < len(ip); i++ {
				if ip[i] == ',' {
					return ip[:i]
				}
			}
			return ip
		}
	}
	// Fall back to RemoteAddr (strip port)
	host := r.RemoteAddr
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[:i]
		}
	}
	return host
}