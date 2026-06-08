package mirror

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"devbox/internal/config"
)

type QuayMirror struct {
	mu       sync.RWMutex
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&QuayMirror{})
}

func (q *QuayMirror) Name() string    { return "quay" }
func (q *QuayMirror) Pattern() string { return "/quay/" }
func (q *QuayMirror) Upstream() string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.upstream
}
func (q *QuayMirror) SetUpstream(url string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.upstream = url
}
func (q *QuayMirror) IsEnabled() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.enabled
}
func (q *QuayMirror) SetEnabled(e bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.enabled = e
}
func (q *QuayMirror) CacheTTL() string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	d := q.cacheTTL
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

func (q *QuayMirror) ApplyConfig(cfg config.MirrorConfig) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.enabled = cfg.Enabled
	q.upstream = cfg.Upstream
	q.cacheTTL = cfg.CacheTTLd
}

func (q *QuayMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	q.mu.RLock()
	upstream := q.upstream
	q.mu.RUnlock()
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/quay"):]
		cache.ProxyStream(w, r, upstream)
	}
}

func (q *QuayMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.cacheTTL = d
	return nil
}

func (q *QuayMirror) HealthCheck() error {
	resp, err := HealthGet(q.upstream + "/v2/")
	if err != nil {
		return fmt.Errorf("quay upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		return nil
	}
	return fmt.Errorf("quay upstream returned %d", resp.StatusCode)
}
