package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type CargoMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&CargoMirror{})
}

func (c *CargoMirror) Name() string           { return "cargo" }
func (c *CargoMirror) Pattern() string        { return "/cargo/" }
func (c *CargoMirror) Upstream() string       { return c.upstream }
func (c *CargoMirror) SetUpstream(url string) { c.upstream = url }
func (c *CargoMirror) IsEnabled() bool        { return c.enabled }
func (c *CargoMirror) SetEnabled(e bool)      { c.enabled = e }
func (c *CargoMirror) CacheTTL() string       { return fmt.Sprintf("%d", c.cacheTTL/time.Second) }

func (c *CargoMirror) ApplyConfig(cfg config.MirrorConfig) {
	c.enabled = cfg.Enabled
	c.upstream = cfg.Upstream
	c.cacheTTL = cfg.CacheTTLd
}

func (c *CargoMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/cargo"):]
		cache.ProxyHTTP(w, r, c.upstream, c.cacheTTL)
	}
}

func (c *CargoMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	c.cacheTTL = d
	return nil
}

func (c *CargoMirror) HealthCheck() error {
	resp, err := HealthGet(c.upstream + "/")
	if err != nil {
		return fmt.Errorf("cargo upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cargo upstream returned %d", resp.StatusCode)
	}
	return nil
}
