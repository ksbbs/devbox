package mirror

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"devbox/internal/config"
)

type CranMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
	mu       sync.RWMutex
}

func init() {
	Register(&CranMirror{})
}

func (c *CranMirror) Name() string    { return "cran" }
func (c *CranMirror) Pattern() string { return "/cran/" }
func (c *CranMirror) Upstream() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.upstream
}
func (c *CranMirror) SetUpstream(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.upstream = url
}
func (c *CranMirror) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}
func (c *CranMirror) SetEnabled(e bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = e
}
func (c *CranMirror) CacheTTL() string {
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

func (c *CranMirror) ApplyConfig(cfg config.MirrorConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = cfg.Enabled
	c.upstream = cfg.Upstream
	c.cacheTTL = cfg.CacheTTLd
}

func (c *CranMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	c.mu.RLock()
	upstream := c.upstream
	cacheTTL := c.cacheTTL
	c.mu.RUnlock()
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/cran"):]
		cache.ProxyHTTP(w, r, upstream, cacheTTL)
	}
}

func (c *CranMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cacheTTL = d
	return nil
}

func (c *CranMirror) HealthCheck() error {
	resp, err := HealthGet(c.Upstream() + "/")
	if err != nil {
		return fmt.Errorf("cran upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cran upstream returned %d", resp.StatusCode)
	}
	return nil
}
