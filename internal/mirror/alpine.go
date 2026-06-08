package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type AlpineMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&AlpineMirror{})
}

func (a *AlpineMirror) Name() string           { return "alpine" }
func (a *AlpineMirror) Pattern() string        { return "/alpine/" }
func (a *AlpineMirror) Upstream() string       { return a.upstream }
func (a *AlpineMirror) SetUpstream(url string) { a.upstream = url }
func (a *AlpineMirror) IsEnabled() bool        { return a.enabled }
func (a *AlpineMirror) SetEnabled(e bool)      { a.enabled = e }
func (a *AlpineMirror) CacheTTL() string       { return fmt.Sprintf("%d", a.cacheTTL/time.Second) }

func (a *AlpineMirror) ApplyConfig(cfg config.MirrorConfig) {
	a.enabled = cfg.Enabled
	a.upstream = cfg.Upstream
	a.cacheTTL = cfg.CacheTTLd
}

func (a *AlpineMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/alpine"):]
		cache.ProxyStream(w, r, a.upstream)
	}
}

func (a *AlpineMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	a.cacheTTL = d
	return nil
}

func (a *AlpineMirror) HealthCheck() error {
	resp, err := HealthGet(a.upstream + "/")
	if err != nil {
		return fmt.Errorf("alpine upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("alpine upstream returned %d", resp.StatusCode)
	}
	return nil
}
