package mirror

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"devbox/internal/config"
)

type testMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func (m *testMirror) Name() string           { return "test" }
func (m *testMirror) Pattern() string        { return "/test/" }
func (m *testMirror) Upstream() string       { return m.upstream }
func (m *testMirror) SetUpstream(url string) { m.upstream = url }
func (m *testMirror) IsEnabled() bool        { return m.enabled }
func (m *testMirror) SetEnabled(e bool)      { m.enabled = e }
func (m *testMirror) CacheTTL() string       { return fmt.Sprintf("%d", m.cacheTTL/time.Second) }
func (m *testMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	m.cacheTTL = d
	return nil
}
func (m *testMirror) ApplyConfig(cfg config.MirrorConfig) {
	m.enabled = cfg.Enabled
	m.upstream = cfg.Upstream
	m.cacheTTL = cfg.CacheTTLd
}
func (m *testMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/test")
		cache.ProxyHTTP(w, r, m.upstream, m.cacheTTL)
	}
}
func (m *testMirror) HealthCheck() error {
	return nil
}

func TestRegisterAndGet(t *testing.T) {
	Register(&testMirror{})
	defer func() {
		registry.Lock()
		delete(registry.m, "test")
		registry.Unlock()
	}()

	m, ok := Get("test")
	if !ok {
		t.Fatal("expected mirror 'test' to be registered")
	}
	if m.Name() != "test" {
		t.Fatalf("expected name 'test', got '%s'", m.Name())
	}
}

func TestAllMirrors(t *testing.T) {
	mirrors := All()
	if len(mirrors) == 0 {
		t.Fatal("expected at least one mirror registered")
	}

	names := make(map[string]bool)
	for _, m := range mirrors {
		if names[m.Name()] {
			t.Fatalf("duplicate mirror name: %s", m.Name())
		}
		names[m.Name()] = true
		if m.Pattern() == "" {
			t.Fatalf("mirror %s has empty pattern", m.Name())
		}
	}
}

func TestMirrorSetCacheTTLValid(t *testing.T) {
	m := &testMirror{}
	if err := m.SetCacheTTL("7d"); err != nil {
		t.Fatalf("expected no error for '7d', got %v", err)
	}
	if m.cacheTTL != 7*24*time.Hour {
		t.Fatalf("expected 7d, got %v", m.cacheTTL)
	}
}

func TestMirrorSetCacheTTLInvalid(t *testing.T) {
	m := &testMirror{}
	if err := m.SetCacheTTL("invalid"); err == nil {
		t.Fatal("expected error for invalid TTL string")
	}
}

func TestMirrorHealthGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := HealthGet(ts.URL)
	if err != nil {
		t.Fatalf("HealthGet failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestMirrorHealthGetFailure(t *testing.T) {
	_, err := HealthGet("http://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error for unreachable upstream")
	}
}

func TestApplyConfig(t *testing.T) {
	m := &testMirror{}
	cfg := config.MirrorConfig{
		Enabled:   true,
		Upstream:  "https://example.com",
		CacheTTL:  "7d",
		CacheTTLd: 7 * 24 * time.Hour,
	}
	m.ApplyConfig(cfg)

	if !m.IsEnabled() {
		t.Fatal("expected mirror to be enabled")
	}
	if m.Upstream() != "https://example.com" {
		t.Fatalf("expected upstream 'https://example.com', got '%s'", m.Upstream())
	}
	if m.cacheTTL != 7*24*time.Hour {
		t.Fatalf("expected 7d TTL, got %v", m.cacheTTL)
	}
}

func TestMirrorProxyHandlerThroughCache(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("mirror-proxy"))
	}))
	defer ts.Close()

	dir := t.TempDir()
	c := NewCache(dir, 1<<20)

	m := &testMirror{enabled: true, upstream: ts.URL, cacheTTL: time.Minute}

	handler := m.ProxyHandler(c)

	req := httptest.NewRequest("GET", "/test/some-path", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "mirror-proxy" {
		t.Fatalf("expected 'mirror-proxy', got '%s'", w.Body.String())
	}
}

func TestGetNonExistent(t *testing.T) {
	_, ok := Get("this-mirror-does-not-exist")
	if ok {
		t.Fatal("expected false for non-existent mirror")
	}
}
