package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type QuayMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&QuayMirror{})
}

func (q *QuayMirror) Name() string      { return "quay" }
func (q *QuayMirror) Pattern() string    { return "/quay/" }
func (q *QuayMirror) Upstream() string   { return q.upstream }
func (q *QuayMirror) SetUpstream(url string) { q.upstream = url }
func (q *QuayMirror) IsEnabled() bool    { return q.enabled }
func (q *QuayMirror) SetEnabled(e bool)  { q.enabled = e }
func (q *QuayMirror) CacheTTL() string   { return fmt.Sprintf("%d", q.cacheTTL/time.Second) }

func (q *QuayMirror) ApplyConfig(cfg config.MirrorConfig) {
	q.enabled = cfg.Enabled
	q.upstream = cfg.Upstream
	q.cacheTTL = cfg.CacheTTLd
}

func (q *QuayMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/quay"):]
		cache.ProxyStream(w, r, q.upstream)
	}
}

func (q *QuayMirror) HealthCheck() error {
	resp, err := http.Get(q.upstream + "/v2/")
	if err != nil {
		return fmt.Errorf("quay upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		return nil
	}
	return fmt.Errorf("quay upstream returned %d", resp.StatusCode)
}