package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type NuGetMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&NuGetMirror{})
}

func (n *NuGetMirror) Name() string           { return "nuget" }
func (n *NuGetMirror) Pattern() string        { return "/nuget/" }
func (n *NuGetMirror) Upstream() string       { return n.upstream }
func (n *NuGetMirror) SetUpstream(url string) { n.upstream = url }
func (n *NuGetMirror) IsEnabled() bool        { return n.enabled }
func (n *NuGetMirror) SetEnabled(e bool)      { n.enabled = e }
func (n *NuGetMirror) CacheTTL() string       { return fmt.Sprintf("%d", n.cacheTTL/time.Second) }

func (n *NuGetMirror) ApplyConfig(cfg config.MirrorConfig) {
	n.enabled = cfg.Enabled
	n.upstream = cfg.Upstream
	n.cacheTTL = cfg.CacheTTLd
}

func (n *NuGetMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/nuget"):]
		cache.ProxyHTTP(w, r, n.upstream, n.cacheTTL)
	}
}

func (n *NuGetMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	n.cacheTTL = d
	return nil
}

func (n *NuGetMirror) HealthCheck() error {
	resp, err := HealthGet(n.upstream + "/")
	if err != nil {
		return fmt.Errorf("nuget upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("nuget upstream returned %d", resp.StatusCode)
	}
	return nil
}
