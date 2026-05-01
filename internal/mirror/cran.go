package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type CranMirror struct {
	enabled   bool
	upstream  string
	cacheTTL  time.Duration
}

func init() {
	Register(&CranMirror{})
}

func (c *CranMirror) Name() string     { return "cran" }
func (c *CranMirror) Pattern() string   { return "/cran/" }
func (c *CranMirror) Upstream() string  { return c.upstream }
func (c *CranMirror) SetUpstream(url string) { c.upstream = url }
func (c *CranMirror) IsEnabled() bool   { return c.enabled }
func (c *CranMirror) SetEnabled(e bool) { c.enabled = e }
func (c *CranMirror) CacheTTL() string  { return fmt.Sprintf("%d", c.cacheTTL/time.Second) }

func (c *CranMirror) ApplyConfig(cfg config.MirrorConfig) {
	c.enabled = cfg.Enabled
	c.upstream = cfg.Upstream
	c.cacheTTL = cfg.CacheTTLd
}

func (c *CranMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/cran"):]
		cache.ProxyHTTP(w, r, c.upstream, c.cacheTTL)
	}
}

func (c *CranMirror) HealthCheck() error {
	resp, err := http.Get(c.upstream + "/")
	if err != nil {
		return fmt.Errorf("cran upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cran upstream returned %d", resp.StatusCode)
	}
	return nil
}