package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type DockerMirror struct {
	enabled   bool
	upstream  string
	cacheTTL  time.Duration
}

func init() {
	Register(&DockerMirror{})
}

func (d *DockerMirror) Name() string     { return "docker" }
func (d *DockerMirror) Pattern() string   { return "/docker/" }
func (d *DockerMirror) Upstream() string  { return d.upstream }
func (d *DockerMirror) SetUpstream(url string) { d.upstream = url }
func (d *DockerMirror) IsEnabled() bool   { return d.enabled }
func (d *DockerMirror) SetEnabled(e bool) { d.enabled = e }
func (d *DockerMirror) CacheTTL() string  { return fmt.Sprintf("%d", d.cacheTTL/time.Second) }

func (d *DockerMirror) ApplyConfig(cfg config.MirrorConfig) {
	d.enabled = cfg.Enabled
	d.upstream = cfg.Upstream
	d.cacheTTL = cfg.CacheTTLd
}

func (d *DockerMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/docker"):]
		// Docker registry uses streaming for large layers
		cache.ProxyStream(w, r, d.upstream)
	}
}

func (d *DockerMirror) HealthCheck() error {
	resp, err := http.Get(d.upstream + "/v2/")
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