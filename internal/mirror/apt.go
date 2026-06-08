package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type AptMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&AptMirror{})
}

func (a *AptMirror) Name() string           { return "apt" }
func (a *AptMirror) Pattern() string        { return "/apt/" }
func (a *AptMirror) Upstream() string       { return a.upstream }
func (a *AptMirror) SetUpstream(url string) { a.upstream = url }
func (a *AptMirror) IsEnabled() bool        { return a.enabled }
func (a *AptMirror) SetEnabled(e bool)      { a.enabled = e }
func (a *AptMirror) CacheTTL() string       { return fmt.Sprintf("%d", a.cacheTTL/time.Second) }

func (a *AptMirror) ApplyConfig(cfg config.MirrorConfig) {
	a.enabled = cfg.Enabled
	a.upstream = cfg.Upstream
	a.cacheTTL = cfg.CacheTTLd
}

func (a *AptMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/apt"):]
		cache.ProxyStream(w, r, a.upstream)
	}
}

func (a *AptMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	a.cacheTTL = d
	return nil
}

func (a *AptMirror) HealthCheck() error {
	resp, err := HealthGet(a.upstream + "/")
	if err != nil {
		return fmt.Errorf("apt upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("apt upstream returned %d", resp.StatusCode)
	}
	return nil
}
