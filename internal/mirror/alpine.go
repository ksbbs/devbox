package mirror

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"devbox/internal/config"
)

type AlpineMirror struct {
	mu       sync.RWMutex
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&AlpineMirror{})
}

func (a *AlpineMirror) Name() string    { return "alpine" }
func (a *AlpineMirror) Pattern() string { return "/alpine/" }
func (a *AlpineMirror) Upstream() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.upstream
}
func (a *AlpineMirror) SetUpstream(url string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.upstream = url
}
func (a *AlpineMirror) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}
func (a *AlpineMirror) SetEnabled(e bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = e
}
func (a *AlpineMirror) CacheTTL() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	d := a.cacheTTL
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

func (a *AlpineMirror) ApplyConfig(cfg config.MirrorConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = cfg.Enabled
	a.upstream = cfg.Upstream
	a.cacheTTL = cfg.CacheTTLd
}

func (a *AlpineMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	a.mu.RLock()
	upstream := a.upstream
	a.mu.RUnlock()
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/alpine")
		cache.ProxyStream(w, r, upstream)
	}
}

func (a *AlpineMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cacheTTL = d
	return nil
}

func (a *AlpineMirror) HealthCheck() error {
	resp, err := HealthGet(a.Upstream() + "/")
	if err != nil {
		return fmt.Errorf("alpine upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("alpine upstream returned %d", resp.StatusCode)
	}
	return nil
}
