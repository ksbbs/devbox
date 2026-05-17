package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type Limiter struct {
	mu        sync.Mutex
	visitors  map[string]*visitorWindow
	limit     int
	window    time.Duration
	whitelist []*net.IPNet
	blacklist []*net.IPNet
}

type visitorWindow struct {
	requests []time.Time
	lastSeen time.Time
}

func New(limit int, window time.Duration, whitelist []string, blacklist []string) *Limiter {
	wlNets := parseCIDRList(whitelist)
	blNets := parseCIDRList(blacklist)

	return &Limiter{
		visitors:  make(map[string]*visitorWindow),
		limit:     limit,
		window:    window,
		whitelist: wlNets,
		blacklist: blNets,
	}
}

func (l *Limiter) Allow(r *http.Request) bool {
	ip := extractIP(r)
	ipNet := parseIP(ip)

	for _, cidr := range l.blacklist {
		if cidr.Contains(ipNet) {
			return false
		}
	}

	for _, cidr := range l.whitelist {
		if cidr.Contains(ipNet) {
			return true
		}
	}

	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	v, ok := l.visitors[ip]
	if !ok {
		v = &visitorWindow{}
		l.visitors[ip] = v
	}
	v.lastSeen = now

	kept := v.requests[:0]
	for _, t := range v.requests {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	v.requests = kept

	if len(v.requests) >= l.limit {
		return false
	}

	v.requests = append(v.requests, now)
	return true
}

func (l *Limiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for ip, v := range l.visitors {
		if now.Sub(v.lastSeen) > l.window {
			delete(l.visitors, ip)
		}
	}
}

func parseCIDRList(list []string) []*net.IPNet {
	var nets []*net.IPNet
	for _, entry := range list {
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
