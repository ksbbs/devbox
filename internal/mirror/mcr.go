package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type McrMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&McrMirror{})
}

func (m *McrMirror) Name() string      { return "mcr" }
func (m *McrMirror) Pattern() string    { return "/mcr/" }
func (m *McrMirror) Upstream() string   { return m.upstream }
func (m *McrMirror) SetUpstream(url string) { m.upstream = url }
func (m *McrMirror) IsEnabled() bool    { return m.enabled }
func (m *McrMirror) SetEnabled(e bool)  { m.enabled = e }
func (m *McrMirror) CacheTTL() string   { return fmt.Sprintf("%d", m.cacheTTL/time.Second) }

func (m *McrMirror) ApplyConfig(cfg config.MirrorConfig) {
	m.enabled = cfg.Enabled
	m.upstream = cfg.Upstream
	m.cacheTTL = cfg.CacheTTLd
}

func (m *McrMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/mcr"):]
		cache.ProxyStream(w, r, m.upstream)
	}
}

func (m *McrMirror) HealthCheck() error {
	resp, err := http.Get(m.upstream + "/v2/")
	if err != nil {
		return fmt.Errorf("mcr upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		return nil
	}
	return fmt.Errorf("mcr upstream returned %d", resp.StatusCode)
}