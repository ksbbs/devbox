package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type CondaMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&CondaMirror{})
}

func (c *CondaMirror) Name() string           { return "conda" }
func (c *CondaMirror) Pattern() string        { return "/conda/" }
func (c *CondaMirror) Upstream() string       { return c.upstream }
func (c *CondaMirror) SetUpstream(url string) { c.upstream = url }
func (c *CondaMirror) IsEnabled() bool        { return c.enabled }
func (c *CondaMirror) SetEnabled(e bool)      { c.enabled = e }
func (c *CondaMirror) CacheTTL() string       { return fmt.Sprintf("%d", c.cacheTTL/time.Second) }

func (c *CondaMirror) ApplyConfig(cfg config.MirrorConfig) {
	c.enabled = cfg.Enabled
	c.upstream = cfg.Upstream
	c.cacheTTL = cfg.CacheTTLd
}

func (c *CondaMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/conda"):]
		cache.ProxyHTTP(w, r, c.upstream, c.cacheTTL)
	}
}

func (c *CondaMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	c.cacheTTL = d
	return nil
}

func (c *CondaMirror) HealthCheck() error {
	resp, err := HealthGet(c.upstream + "/")
	if err != nil {
		return fmt.Errorf("conda upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("conda upstream returned %d", resp.StatusCode)
	}
	return nil
}
