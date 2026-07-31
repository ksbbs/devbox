package mirror

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"devbox/internal/config"
)

type CargoMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
	mu       sync.RWMutex
}

func init() {
	Register(&CargoMirror{})
}

func (c *CargoMirror) Name() string    { return "cargo" }
func (c *CargoMirror) Pattern() string { return "/cargo/" }
func (c *CargoMirror) Upstream() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.upstream
}
func (c *CargoMirror) SetUpstream(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.upstream = url
}
func (c *CargoMirror) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}
func (c *CargoMirror) SetEnabled(e bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = e
}
func (c *CargoMirror) CacheTTL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	d := c.cacheTTL
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

func (c *CargoMirror) ApplyConfig(cfg config.MirrorConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = cfg.Enabled
	c.upstream = cfg.Upstream
	c.cacheTTL = cfg.CacheTTLd
}

func (c *CargoMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c.mu.RLock()
		upstream := c.upstream
		cacheTTL := c.cacheTTL
		c.mu.RUnlock()
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/cargo")
		cache.ProxyHTTP(w, r, upstream, cacheTTL)
	}
}

func (c *CargoMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cacheTTL = d
	return nil
}

func (c *CargoMirror) HealthCheck() error {
	resp, err := HealthGet(strings.TrimRight(c.Upstream(), "/") + "/serde/serde-1.0.0.crate")
	if err != nil {
		return fmt.Errorf("cargo upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cargo upstream returned %d", resp.StatusCode)
	}
	return nil
}
