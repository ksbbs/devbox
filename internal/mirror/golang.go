package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type GolangMirror struct {
	enabled   bool
	upstream  string
	cacheTTL  time.Duration
}

func init() {
	Register(&GolangMirror{})
}

func (g *GolangMirror) Name() string     { return "golang" }
func (g *GolangMirror) Pattern() string   { return "/golang/" }
func (g *GolangMirror) Upstream() string  { return g.upstream }
func (g *GolangMirror) SetUpstream(url string) { g.upstream = url }
func (g *GolangMirror) IsEnabled() bool   { return g.enabled }
func (g *GolangMirror) SetEnabled(e bool) { g.enabled = e }
func (g *GolangMirror) CacheTTL() string  { return fmt.Sprintf("%d", g.cacheTTL/time.Second) }

func (g *GolangMirror) ApplyConfig(cfg config.MirrorConfig) {
	g.enabled = cfg.Enabled
	g.upstream = cfg.Upstream
	g.cacheTTL = cfg.CacheTTLd
}

func (g *GolangMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/golang"):]
		cache.ProxyHTTP(w, r, g.upstream, g.cacheTTL)
	}
}

func (g *GolangMirror) HealthCheck() error {
	resp, err := http.Get(g.upstream + "/github.com/golang/go/@v/list")
	if err != nil {
		return fmt.Errorf("golang upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("golang upstream returned %d", resp.StatusCode)
	}
	return nil
}