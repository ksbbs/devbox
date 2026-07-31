package ratelimit

import (
	"net"
	"net/http"
	"strings"
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
		if !strings.Contains(entry, "/") {
			if strings.Contains(entry, ":") {
				entry += "/128"
			} else {
				entry += "/32"
			}
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

func extractIP(r *http.Request) string {
	// Only trust proxy headers (X-Real-IP / X-Forwarded-For) when the direct
	// peer is a loopback address, i.e. traffic arrives via a local reverse
	// proxy. Directly exposed clients can otherwise forge these headers to
	// bypass or deflect rate limiting.
	remoteIP := peerIP(r.RemoteAddr)
	if remoteIP != nil && remoteIP.IsLoopback() {
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return strings.TrimSpace(ip)
		}
		if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			if idx := strings.IndexByte(ip, ','); idx > 0 {
				ip = ip[:idx]
			}
			return strings.TrimSpace(ip)
		}
	}
	if remoteIP != nil {
		return remoteIP.String()
	}
	return r.RemoteAddr
}

// peerIP extracts the IP from a RemoteAddr string ("1.2.3.4:5678", "[::1]:80").
func peerIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil
	}
	return ip
}
