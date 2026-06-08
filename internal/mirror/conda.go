package mirror

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"devbox/internal/config"
)

type CondaMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
	mu       sync.RWMutex
}

func init() {
	Register(&CondaMirror{})
}

func (c *CondaMirror) Name() string    { return "conda" }
func (c *CondaMirror) Pattern() string { return "/conda/" }
func (c *CondaMirror) Upstream() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.upstream
}
func (c *CondaMirror) SetUpstream(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.upstream = url
}
func (c *CondaMirror) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}
func (c *CondaMirror) SetEnabled(e bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = e
}
func (c *CondaMirror) CacheTTL() string {
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

func (c *CondaMirror) ApplyConfig(cfg config.MirrorConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = cfg.Enabled
	c.upstream = cfg.Upstream
	c.cacheTTL = cfg.CacheTTLd
}

func (c *CondaMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	c.mu.RLock()
	upstream := c.upstream
	cacheTTL := c.cacheTTL
	c.mu.RUnlock()
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/conda"):]
		cache.ProxyHTTP(w, r, upstream, cacheTTL)
	}
}

func (c *CondaMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cacheTTL = d
	return nil
}

func (c *CondaMirror) HealthCheck() error {
	resp, err := HealthGet(c.Upstream() + "/")
	if err != nil {
		return fmt.Errorf("conda upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("conda upstream returned %d", resp.StatusCode)
	}
	return nil
}
