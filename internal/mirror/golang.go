package mirror

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"devbox/internal/config"
)

type GolangMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
	mu       sync.RWMutex
}

func init() {
	Register(&GolangMirror{})
}

func (g *GolangMirror) Name() string    { return "golang" }
func (g *GolangMirror) Pattern() string { return "/golang/" }
func (g *GolangMirror) Upstream() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.upstream
}
func (g *GolangMirror) SetUpstream(url string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.upstream = url
}
func (g *GolangMirror) IsEnabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.enabled
}
func (g *GolangMirror) SetEnabled(e bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.enabled = e
}
func (g *GolangMirror) CacheTTL() string {
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
	return fmt.Sprintf("%dm", secs/60)
}

func (g *GolangMirror) ApplyConfig(cfg config.MirrorConfig) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.enabled = cfg.Enabled
	g.upstream = cfg.Upstream
	g.cacheTTL = cfg.CacheTTLd
}

func (g *GolangMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	g.mu.RLock()
	upstream := g.upstream
	cacheTTL := g.cacheTTL
	g.mu.RUnlock()
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/golang"):]
		cache.ProxyHTTP(w, r, upstream, cacheTTL)
	}
}

func (g *GolangMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cacheTTL = d
	return nil
}

func (g *GolangMirror) HealthCheck() error {
	resp, err := HealthGet(g.Upstream() + "/github.com/golang/go/@v/list")
	if err != nil {
		return fmt.Errorf("golang upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("golang upstream returned %d", resp.StatusCode)
	}
	return nil
}
