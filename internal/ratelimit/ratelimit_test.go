package ratelimit

import (
	"net/http/httptest"
	"testing"
)

func TestExtractIPIgnoresForgedProxyHeaders(t *testing.T) {
	// Directly exposed clients must not be able to forge proxy headers.
	l := New(10, 0, nil, nil, nil)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.7:5678"
	req.Header.Set("X-Real-IP", "10.0.0.1")
	req.Header.Set("X-Forwarded-For", "10.0.0.2, 10.0.0.3")

	if got := l.extractIP(req); got != "203.0.113.7" {
		t.Fatalf("expected real peer IP 203.0.113.7, got %s", got)
	}
}

func TestExtractIPTrustsProxyBehindLoopback(t *testing.T) {
	// Loopback peers are trusted by default (local reverse proxy).
	l := New(10, 0, nil, nil, nil)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("X-Real-IP", "203.0.113.9")

	if got := l.extractIP(req); got != "203.0.113.9" {
		t.Fatalf("expected forwarded IP 203.0.113.9, got %s", got)
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "[::1]:8080"
	req2.Header.Set("X-Forwarded-For", "203.0.113.10, 203.0.113.11")

	if got := l.extractIP(req2); got != "203.0.113.10" {
		t.Fatalf("expected first forwarded IP 203.0.113.10, got %s", got)
	}
}

func TestExtractIPTrustsConfiguredProxyCIDR(t *testing.T) {
	// Docker deployment: nginx on the host, container peers are the bridge gateway.
	l := New(10, 0, nil, nil, []string{"172.16.0.1/12", "127.0.0.1/8"})
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "172.16.0.1:5678"
	req.Header.Set("X-Real-IP", "__VG_IPV4_1baa9e3bb4e1__")

	if got := l.extractIP(req); got != "__VG_IPV4_1baa9e3bb4e1__" {
		t.Fatalf("expected forwarded IP __VG_IPV4_1baa9e3bb4e1__, got %s", got)
	}

	// Same bridge CIDR, but header absent → peer IP used.
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "172.16.0.1:5678"
	if got := l.extractIP(req2); got != "172.16.0.1" {
		t.Fatalf("expected peer IP 172.16.0.1, got %s", got)
	}
}

func TestExtractIPIPv6Peer(t *testing.T) {
	l := New(10, 0, nil, nil, nil)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "[2001:db8::1]:443"

	if got := l.extractIP(req); got != "2001:db8::1" {
		t.Fatalf("expected IPv6 peer, got %s", got)
	}
}
