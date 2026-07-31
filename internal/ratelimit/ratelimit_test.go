package ratelimit

import (
	"net/http/httptest"
	"testing"
)

func TestExtractIPIgnoresForgedProxyHeaders(t *testing.T) {
	// Directly exposed clients must not be able to forge proxy headers.
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.7:5678"
	req.Header.Set("X-Real-IP", "10.0.0.1")
	req.Header.Set("X-Forwarded-For", "10.0.0.2, 10.0.0.3")

	if got := extractIP(req); got != "203.0.113.7" {
		t.Fatalf("expected real peer IP 203.0.113.7, got %s", got)
	}
}

func TestExtractIPTrustsProxyBehindLoopback(t *testing.T) {
	// Behind a local reverse proxy the client IP arrives via headers.
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("X-Real-IP", "203.0.113.9")

	if got := extractIP(req); got != "203.0.113.9" {
		t.Fatalf("expected forwarded IP 203.0.113.9, got %s", got)
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "[::1]:8080"
	req2.Header.Set("X-Forwarded-For", "203.0.113.10, 203.0.113.11")

	if got := extractIP(req2); got != "203.0.113.10" {
		t.Fatalf("expected first forwarded IP 203.0.113.10, got %s", got)
	}
}

func TestExtractIPIPv6Peer(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "[2001:db8::1]:443"

	if got := extractIP(req); got != "2001:db8::1" {
		t.Fatalf("expected IPv6 peer, got %s", got)
	}
}
