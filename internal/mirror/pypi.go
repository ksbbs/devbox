package mirror

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"devbox/internal/config"
)

type PypiMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
	mu       sync.RWMutex
}

func init() {
	Register(&PypiMirror{})
}

func (p *PypiMirror) Name() string    { return "pypi" }
func (p *PypiMirror) Pattern() string { return "/pypi/" }
func (p *PypiMirror) Upstream() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.upstream
}
func (p *PypiMirror) SetUpstream(url string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.upstream = url
}
func (p *PypiMirror) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}
func (p *PypiMirror) SetEnabled(e bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = e
}
func (p *PypiMirror) CacheTTL() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	d := p.cacheTTL
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
	if secs%60 == 0 {
		return fmt.Sprintf("%dm", secs/60)
	}
	return fmt.Sprintf("%ds", secs)
}

func (p *PypiMirror) ApplyConfig(cfg config.MirrorConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = cfg.Enabled
	p.upstream = cfg.Upstream
	p.cacheTTL = cfg.CacheTTLd
}

func (p *PypiMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p.mu.RLock()
		upstream := p.upstream
		cacheTTL := p.cacheTTL
		p.mu.RUnlock()
		upstream = strings.TrimRight(upstream, "/")
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/pypi")
		cache.ProxyHTTP(w, r, upstream, cacheTTL)
	}
}

func (p *PypiMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cacheTTL = d
	return nil
}

func (p *PypiMirror) HealthCheck() error {
	resp, err := HealthGet(p.Upstream() + "/")
	if err != nil {
		return fmt.Errorf("pypi upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return fmt.Errorf("pypi upstream returned %d", resp.StatusCode)
	}
	return nil
}
