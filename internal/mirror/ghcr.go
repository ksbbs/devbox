package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type GhcrMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&GhcrMirror{})
}

func (g *GhcrMirror) Name() string      { return "ghcr" }
func (g *GhcrMirror) Pattern() string    { return "/ghcr/" }
func (g *GhcrMirror) Upstream() string   { return g.upstream }
func (g *GhcrMirror) SetUpstream(url string) { g.upstream = url }
func (g *GhcrMirror) IsEnabled() bool    { return g.enabled }
func (g *GhcrMirror) SetEnabled(e bool)  { g.enabled = e }
func (g *GhcrMirror) CacheTTL() string   { return fmt.Sprintf("%d", g.cacheTTL/time.Second) }

func (g *GhcrMirror) ApplyConfig(cfg config.MirrorConfig) {
	g.enabled = cfg.Enabled
	g.upstream = cfg.Upstream
	g.cacheTTL = cfg.CacheTTLd
}

func (g *GhcrMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/ghcr"):]
		cache.ProxyStream(w, r, g.upstream)
	}
}

func (g *GhcrMirror) HealthCheck() error {
	resp, err := http.Get(g.upstream + "/v2/")
	if err != nil {
		return fmt.Errorf("ghcr upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		return nil
	}
	return fmt.Errorf("ghcr upstream returned %d", resp.StatusCode)
}