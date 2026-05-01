package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type GithubAPIMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&GithubAPIMirror{})
}

func (g *GithubAPIMirror) Name() string      { return "ghapi" }
func (g *GithubAPIMirror) Pattern() string    { return "/ghapi/" }
func (g *GithubAPIMirror) Upstream() string   { return g.upstream }
func (g *GithubAPIMirror) SetUpstream(url string) { g.upstream = url }
func (g *GithubAPIMirror) IsEnabled() bool    { return g.enabled }
func (g *GithubAPIMirror) SetEnabled(e bool)  { g.enabled = e }
func (g *GithubAPIMirror) CacheTTL() string   { return fmt.Sprintf("%d", g.cacheTTL/time.Second) }

func (g *GithubAPIMirror) ApplyConfig(cfg config.MirrorConfig) {
	g.enabled = cfg.Enabled
	g.upstream = cfg.Upstream
	g.cacheTTL = cfg.CacheTTLd
}

func (g *GithubAPIMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/ghapi"):]
		cache.ProxyHTTP(w, r, g.upstream, g.cacheTTL)
	}
}

func (g *GithubAPIMirror) HealthCheck() error {
	resp, err := http.Get(g.upstream + "/rate_limit")
	if err != nil {
		return fmt.Errorf("ghapi upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil
	}
	return fmt.Errorf("ghapi upstream returned %d", resp.StatusCode)
}