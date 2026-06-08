package mirror

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"devbox/internal/config"
)

type McrMirror struct {
	mu       sync.RWMutex
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&McrMirror{})
}

func (m *McrMirror) Name() string    { return "mcr" }
func (m *McrMirror) Pattern() string { return "/mcr/" }
func (m *McrMirror) Upstream() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.upstream
}
func (m *McrMirror) SetUpstream(url string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upstream = url
}
func (m *McrMirror) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}
func (m *McrMirror) SetEnabled(e bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = e
}
func (m *McrMirror) CacheTTL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d := m.cacheTTL
	if d == 0 {
		return "0"
	}
	secs := d / time.Second
	if secs%86400 == 0 {
		return fmt.Sprintf("%dd", secs/86400)
	}
	if secs%3600 == 0 {
		return fmt.Sprintf("%dh", secs/3600)
	}
	return fmt.Sprintf("%dm", secs/60)
}

func (m *McrMirror) ApplyConfig(cfg config.MirrorConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = cfg.Enabled
	m.upstream = cfg.Upstream
	m.cacheTTL = cfg.CacheTTLd
}

func (m *McrMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	m.mu.RLock()
	upstream := m.upstream
	m.mu.RUnlock()
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/mcr")
		cache.ProxyStream(w, r, upstream)
	}
}

func (m *McrMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheTTL = d
	return nil
}

func (m *McrMirror) HealthCheck() error {
	resp, err := HealthGet(m.Upstream() + "/v2/")
	if err != nil {
		return fmt.Errorf("mcr upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		return nil
	}
	return fmt.Errorf("mcr upstream returned %d", resp.StatusCode)
}
