package mirror

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"devbox/internal/config"
)

type AptMirror struct {
	mu       sync.RWMutex
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&AptMirror{})
}

func (a *AptMirror) Name() string    { return "apt" }
func (a *AptMirror) Pattern() string { return "/apt/" }
func (a *AptMirror) Upstream() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.upstream
}
func (a *AptMirror) SetUpstream(url string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.upstream = url
}
func (a *AptMirror) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}
func (a *AptMirror) SetEnabled(e bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = e
}
func (a *AptMirror) CacheTTL() string {
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

func (a *AptMirror) ApplyConfig(cfg config.MirrorConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = cfg.Enabled
	a.upstream = cfg.Upstream
	a.cacheTTL = cfg.CacheTTLd
}

func (a *AptMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.mu.RLock()
		upstream := a.upstream
		a.mu.RUnlock()
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/apt")
		cache.ProxyStream(w, r, upstream)
	}
}

func (a *AptMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cacheTTL = d
	return nil
}

func (a *AptMirror) HealthCheck() error {
	resp, err := HealthGet(a.Upstream() + "/")
	if err != nil {
		return fmt.Errorf("apt upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("apt upstream returned %d", resp.StatusCode)
	}
	return nil
}
