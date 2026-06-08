package mirror

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"devbox/internal/config"
)

type NpmMirror struct {
	mu       sync.RWMutex
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&NpmMirror{})
}

func (n *NpmMirror) Name() string    { return "npm" }
func (n *NpmMirror) Pattern() string { return "/npm/" }
func (n *NpmMirror) Upstream() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.upstream
}
func (n *NpmMirror) SetUpstream(url string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.upstream = url
}
func (n *NpmMirror) IsEnabled() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.enabled
}
func (n *NpmMirror) SetEnabled(e bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enabled = e
}
func (n *NpmMirror) CacheTTL() string {
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
	return fmt.Sprintf("%dm", secs/60)
}

func (n *NpmMirror) ApplyConfig(cfg config.MirrorConfig) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enabled = cfg.Enabled
	n.upstream = cfg.Upstream
	n.cacheTTL = cfg.CacheTTLd
}

func (n *NpmMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n.mu.RLock()
		upstream := n.upstream
		cacheTTL := n.cacheTTL
		n.mu.RUnlock()
		r.URL.Path = r.URL.Path[len("/npm"):]
		if r.URL.Path == "" || r.URL.Path == "/" {
			r.URL.Path = "/"
		}
		if strings.Contains(r.URL.Path, ".tgz") ||
			strings.Contains(r.URL.Path, ".tar.gz") ||
			strings.Contains(r.URL.Path, "/-/") {
			cache.ProxyStream(w, r, upstream)
		} else {
			cache.ProxyHTTP(w, r, upstream, cacheTTL)
		}
	}
}

func (n *NpmMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.cacheTTL = d
	return nil
}

func (n *NpmMirror) HealthCheck() error {
	resp, err := HealthGet(n.upstream + "/")
	if err != nil {
		return fmt.Errorf("npm upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("npm upstream returned %d", resp.StatusCode)
	}
	return nil
}
