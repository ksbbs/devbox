package mirror

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"devbox/internal/config"
)

type GithubAPIMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
	mu       sync.RWMutex
}

func init() {
	Register(&GithubAPIMirror{})
}

func (g *GithubAPIMirror) Name() string    { return "ghapi" }
func (g *GithubAPIMirror) Pattern() string { return "/ghapi/" }
func (g *GithubAPIMirror) Upstream() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.upstream
}
func (g *GithubAPIMirror) SetUpstream(url string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.upstream = url
}
func (g *GithubAPIMirror) IsEnabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.enabled
}
func (g *GithubAPIMirror) SetEnabled(e bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.enabled = e
}
func (g *GithubAPIMirror) CacheTTL() string {
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

func (g *GithubAPIMirror) ApplyConfig(cfg config.MirrorConfig) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.enabled = cfg.Enabled
	g.upstream = cfg.Upstream
	g.cacheTTL = cfg.CacheTTLd
}

func (g *GithubAPIMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		g.mu.RLock()
		upstream := g.upstream
		cacheTTL := g.cacheTTL
		g.mu.RUnlock()
		upstream = strings.TrimRight(upstream, "/")
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/ghapi")
		cache.ProxyHTTP(w, r, upstream, cacheTTL)
	}
}

func (g *GithubAPIMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cacheTTL = d
	return nil
}

func (g *GithubAPIMirror) HealthCheck() error {
	resp, err := HealthGet(g.Upstream() + "/rate_limit")
	if err != nil {
		return fmt.Errorf("ghapi upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil
	}
	return fmt.Errorf("ghapi upstream returned %d", resp.StatusCode)
}
