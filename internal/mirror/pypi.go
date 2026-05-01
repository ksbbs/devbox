package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type PypiMirror struct {
	enabled   bool
	upstream  string
	cacheTTL  time.Duration
}

func init() {
	Register(&PypiMirror{})
}

func (p *PypiMirror) Name() string     { return "pypi" }
func (p *PypiMirror) Pattern() string   { return "/pypi/" }
func (p *PypiMirror) Upstream() string  { return p.upstream }
func (p *PypiMirror) SetUpstream(url string) { p.upstream = url }
func (p *PypiMirror) IsEnabled() bool   { return p.enabled }
func (p *PypiMirror) SetEnabled(e bool) { p.enabled = e }
func (p *PypiMirror) CacheTTL() string  { return fmt.Sprintf("%d", p.cacheTTL/time.Second) }

func (p *PypiMirror) ApplyConfig(cfg config.MirrorConfig) {
	p.enabled = cfg.Enabled
	p.upstream = cfg.Upstream
	p.cacheTTL = cfg.CacheTTLd
}

func (p *PypiMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/pypi"):]
		cache.ProxyHTTP(w, r, p.upstream, p.cacheTTL)
	}
}

func (p *PypiMirror) HealthCheck() error {
	resp, err := http.Get(p.upstream + "/")
	if err != nil {
		return fmt.Errorf("pypi upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return fmt.Errorf("pypi upstream returned %d", resp.StatusCode)
	}
	return nil
}