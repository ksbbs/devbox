package mirror

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"devbox/internal/config"
)

type GhcrMirror struct {
	mu       sync.RWMutex
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&GhcrMirror{})
}

func (g *GhcrMirror) Name() string    { return "ghcr" }
func (g *GhcrMirror) Pattern() string { return "/ghcr/" }
func (g *GhcrMirror) Upstream() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.upstream
}
func (g *GhcrMirror) SetUpstream(url string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.upstream = url
}
func (g *GhcrMirror) IsEnabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.enabled
}
func (g *GhcrMirror) SetEnabled(e bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.enabled = e
}
func (g *GhcrMirror) CacheTTL() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	d := g.cacheTTL
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

func (g *GhcrMirror) ApplyConfig(cfg config.MirrorConfig) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.enabled = cfg.Enabled
	g.upstream = cfg.Upstream
	g.cacheTTL = cfg.CacheTTLd
}

func (g *GhcrMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		g.mu.RLock()
		upstream := g.upstream
		g.mu.RUnlock()
		upstream = strings.TrimRight(upstream, "/")
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/ghcr")
		cache.ProxyStream(w, r, upstream)
	}
}

func (g *GhcrMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cacheTTL = d
	return nil
}

func (g *GhcrMirror) HealthCheck() error {
	resp, err := HealthGet(g.Upstream() + "/v2/")
	if err != nil {
		return fmt.Errorf("ghcr upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		return nil
	}
	return fmt.Errorf("ghcr upstream returned %d", resp.StatusCode)
}
