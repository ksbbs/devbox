package mirror

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"devbox/internal/config"
)

type NuGetMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
	mu       sync.RWMutex
}

func init() {
	Register(&NuGetMirror{})
}

func (n *NuGetMirror) Name() string    { return "nuget" }
func (n *NuGetMirror) Pattern() string { return "/nuget/" }
func (n *NuGetMirror) Upstream() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.upstream
}
func (n *NuGetMirror) SetUpstream(url string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.upstream = url
}
func (n *NuGetMirror) IsEnabled() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.enabled
}
func (n *NuGetMirror) SetEnabled(e bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enabled = e
}
func (n *NuGetMirror) CacheTTL() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	d := n.cacheTTL
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
	if secs%60 == 0 {
		return fmt.Sprintf("%dm", secs/60)
	}
	return fmt.Sprintf("%ds", secs)
}

func (n *NuGetMirror) ApplyConfig(cfg config.MirrorConfig) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enabled = cfg.Enabled
	n.upstream = cfg.Upstream
	n.cacheTTL = cfg.CacheTTLd
}

func (n *NuGetMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n.mu.RLock()
		upstream := n.upstream
		cacheTTL := n.cacheTTL
		n.mu.RUnlock()
		upstream = strings.TrimRight(upstream, "/")
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/nuget")
		cache.ProxyHTTP(w, r, upstream, cacheTTL)
	}
}

func (n *NuGetMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.cacheTTL = d
	return nil
}

func (n *NuGetMirror) HealthCheck() error {
	target := strings.TrimRight(n.Upstream(), "/")
	if strings.HasSuffix(target, "/v3") {
		// The v3 API root itself is not a valid endpoint; probe its index.
		target += "/index.json"
	}
	resp, err := HealthGet(target)
	if err != nil {
		return fmt.Errorf("nuget upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("nuget upstream returned %d", resp.StatusCode)
	}
	return nil
}
