package mirror

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"devbox/internal/config"
)

type HomebrewMirror struct {
	mu       sync.RWMutex
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&HomebrewMirror{})
}

func (h *HomebrewMirror) Name() string    { return "homebrew" }
func (h *HomebrewMirror) Pattern() string { return "/homebrew/" }
func (h *HomebrewMirror) Upstream() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.upstream
}
func (h *HomebrewMirror) SetUpstream(url string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.upstream = url
}
func (h *HomebrewMirror) IsEnabled() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.enabled
}
func (h *HomebrewMirror) SetEnabled(e bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = e
}
func (h *HomebrewMirror) CacheTTL() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	d := h.cacheTTL
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

func (h *HomebrewMirror) ApplyConfig(cfg config.MirrorConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = cfg.Enabled
	h.upstream = cfg.Upstream
	h.cacheTTL = cfg.CacheTTLd
}

func (h *HomebrewMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.RLock()
		upstream := h.upstream
		h.mu.RUnlock()
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/homebrew")
		cache.ProxyStream(w, r, upstream)
	}
}

func (h *HomebrewMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cacheTTL = d
	return nil
}

func (h *HomebrewMirror) HealthCheck() error {
	upstream := h.Upstream()
	if strings.HasSuffix(upstream, "/v2/") {
		// Keep registry base as-is.
	} else if strings.HasSuffix(upstream, "/v2") {
		upstream += "/"
	} else if idx := strings.Index(upstream, "/v2/"); idx >= 0 {
		upstream = upstream[:idx] + "/v2/"
	} else {
		upstream = strings.TrimRight(upstream, "/") + "/v2/"
	}

	resp, err := HealthGet(upstream)
	if err != nil {
		return fmt.Errorf("homebrew upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("homebrew upstream returned %d", resp.StatusCode)
	}
	return nil
}
