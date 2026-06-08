package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type HomebrewMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&HomebrewMirror{})
}

func (h *HomebrewMirror) Name() string           { return "homebrew" }
func (h *HomebrewMirror) Pattern() string        { return "/homebrew/" }
func (h *HomebrewMirror) Upstream() string       { return h.upstream }
func (h *HomebrewMirror) SetUpstream(url string) { h.upstream = url }
func (h *HomebrewMirror) IsEnabled() bool        { return h.enabled }
func (h *HomebrewMirror) SetEnabled(e bool)      { h.enabled = e }
func (h *HomebrewMirror) CacheTTL() string       { return fmt.Sprintf("%d", h.cacheTTL/time.Second) }

func (h *HomebrewMirror) ApplyConfig(cfg config.MirrorConfig) {
	h.enabled = cfg.Enabled
	h.upstream = cfg.Upstream
	h.cacheTTL = cfg.CacheTTLd
}

func (h *HomebrewMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/homebrew"):]
		cache.ProxyStream(w, r, h.upstream)
	}
}

func (h *HomebrewMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	h.cacheTTL = d
	return nil
}

func (h *HomebrewMirror) HealthCheck() error {
	resp, err := HealthGet(h.upstream + "/")
	if err != nil {
		return fmt.Errorf("homebrew upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("homebrew upstream returned %d", resp.StatusCode)
	}
	return nil
}
