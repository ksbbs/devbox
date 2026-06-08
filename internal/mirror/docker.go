package mirror

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"devbox/internal/config"
)

type DockerMirror struct {
	mu       sync.RWMutex
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&DockerMirror{})
}

func (d *DockerMirror) Name() string    { return "docker" }
func (d *DockerMirror) Pattern() string { return "/docker/" }
func (d *DockerMirror) Upstream() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.upstream
}
func (d *DockerMirror) SetUpstream(url string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.upstream = url
}
func (d *DockerMirror) IsEnabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}
func (d *DockerMirror) SetEnabled(e bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = e
}
func (d *DockerMirror) CacheTTL() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	dur := d.cacheTTL
	if dur == 0 {
		return "0"
	}
	secs := dur / time.Second
	if secs%86400 == 0 {
		return fmt.Sprintf("%dd", secs/86400)
	}
	if secs%3600 == 0 {
		return fmt.Sprintf("%dh", secs/3600)
	}
	return fmt.Sprintf("%dm", secs/60)
}

func (d *DockerMirror) ApplyConfig(cfg config.MirrorConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = cfg.Enabled
	d.upstream = cfg.Upstream
	d.cacheTTL = cfg.CacheTTLd
}

func (d *DockerMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d.mu.RLock()
		upstream := d.upstream
		d.mu.RUnlock()
		r.URL.Path = r.URL.Path[len("/docker"):]
		cache.ProxyStream(w, r, upstream)
	}
}

func (d *DockerMirror) SetCacheTTL(ttl string) error {
	dur, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cacheTTL = dur
	return nil
}

func (d *DockerMirror) HealthCheck() error {
	resp, err := HealthGet(d.Upstream() + "/v2/")
	if err != nil {
		return fmt.Errorf("docker upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	// Docker registry v2 returns 401 for unauthenticated, that's healthy
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		return nil
	}
	return fmt.Errorf("docker upstream returned %d", resp.StatusCode)
}
