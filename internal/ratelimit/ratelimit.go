package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Limiter struct {
	mu        sync.Mutex
	visitors  map[string]*visitorLimiter
	rate      rate.Limit // tokens per second
	burst     int        // max burst size
	whitelist []*net.IPNet
	blacklist []*net.IPNet
}

type visitorLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func New(ratePerWindow int, window time.Duration, whitelist []string, blacklist []string) *Limiter {
	// Convert rate/interval to tokens per second
	r := rate.Limit(float64(ratePerWindow) / window.Seconds())
	burst := ratePerWindow

	wlNets := parseCIDRList(whitelist)
	blNets := parseCIDRList(blacklist)

	return &Limiter{
		visitors:  make(map[string]*visitorLimiter),
		rate:      r,
		burst:     burst,
		whitelist: wlNets,
		blacklist: blNets,
	}
}

func (l *Limiter) Allow(r *http.Request) bool {
	ip := extractIP(r)
	ipNet := parseIP(ip)

	// Check blacklist first
	for _, cidr := range l.blacklist {
		if cidr.Contains(ipNet) {
			return false
		}
	}

	// Check whitelist
	for _, cidr := range l.whitelist {
		if cidr.Contains(ipNet) {
			return true
		}
	}

	l.mu.Lock()
	v, ok := l.visitors[ip]
	if !ok {
		v = &visitorLimiter{
			limiter:  rate.NewLimiter(l.rate, l.burst),
			lastSeen: time.Now(),
		}
		l.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	l.mu.Unlock()

	return v.limiter.Allow()
}

func (l *Limiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for ip, v := range l.visitors {
		if now.Sub(v.lastSeen) > 3*time.Hour {
			delete(l.visitors, ip)
		}
	}
}

func parseCIDRList(list []string) []*net.IPNet {
	var nets []*net.IPNet
	for _, entry := range list {
		// If no CIDR mask, treat as single IP with /32 or /128
		if !containsSlash(entry) {
			entry += "/32"
		}
		_, ipNet, err := net.ParseCIDR(entry)
		if err == nil {
			nets = append(nets, ipNet)
		}
	}
	return nets
}

func parseIP(ipStr string) net.IP {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return net.ParseIP("0.0.0.0")
	}
	return ip
}

func containsSlash(s string) bool {
	for _, c := range s {
		if c == '/' {
			return true
		}
	}
	return false
}

func extractIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}
	ip = r.Header.Get("X-Forwarded-For")
	if ip != "" {
		for i := 0; i < len(ip); i++ {
			if ip[i] == ',' {
				return ip[:i]
			}
		}
		return ip
	}
	host := r.RemoteAddr
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[:i]
		}
	}
	return host
}