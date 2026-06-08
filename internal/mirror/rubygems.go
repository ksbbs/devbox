package mirror

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"devbox/internal/config"
)

type RubyGemsMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
	mu       sync.RWMutex
}

func init() {
	Register(&RubyGemsMirror{})
}

func (rg *RubyGemsMirror) Name() string    { return "rubygems" }
func (rg *RubyGemsMirror) Pattern() string { return "/rubygems/" }
func (rg *RubyGemsMirror) Upstream() string {
	rg.mu.RLock()
	defer rg.mu.RUnlock()
	return rg.upstream
}
func (rg *RubyGemsMirror) SetUpstream(url string) {
	rg.mu.Lock()
	defer rg.mu.Unlock()
	rg.upstream = url
}
func (rg *RubyGemsMirror) IsEnabled() bool {
	rg.mu.RLock()
	defer rg.mu.RUnlock()
	return rg.enabled
}
func (rg *RubyGemsMirror) SetEnabled(e bool) {
	rg.mu.Lock()
	defer rg.mu.Unlock()
	rg.enabled = e
}
func (rg *RubyGemsMirror) CacheTTL() string {
	rg.mu.RLock()
	defer rg.mu.RUnlock()
	d := rg.cacheTTL
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

func (rg *RubyGemsMirror) ApplyConfig(cfg config.MirrorConfig) {
	rg.mu.Lock()
	defer rg.mu.Unlock()
	rg.enabled = cfg.Enabled
	rg.upstream = cfg.Upstream
	rg.cacheTTL = cfg.CacheTTLd
}

func (rg *RubyGemsMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	rg.mu.RLock()
	upstream := rg.upstream
	cacheTTL := rg.cacheTTL
	rg.mu.RUnlock()
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/rubygems")
		cache.ProxyHTTP(w, r, upstream, cacheTTL)
	}
}

func (rg *RubyGemsMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	rg.mu.Lock()
	defer rg.mu.Unlock()
	rg.cacheTTL = d
	return nil
}

func (rg *RubyGemsMirror) HealthCheck() error {
	resp, err := HealthGet(rg.upstream + "/")
	if err != nil {
		return fmt.Errorf("rubygems upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rubygems upstream returned %d", resp.StatusCode)
	}
	return nil
}
