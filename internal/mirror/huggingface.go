package mirror

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"devbox/internal/config"
)

type HfMirror struct {
	mu       sync.RWMutex
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&HfMirror{})
}

func (h *HfMirror) Name() string    { return "hf" }
func (h *HfMirror) Pattern() string { return "/hf/" }
func (h *HfMirror) Upstream() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.upstream
}
func (h *HfMirror) SetUpstream(url string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.upstream = url
}
func (h *HfMirror) IsEnabled() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.enabled
}
func (h *HfMirror) SetEnabled(e bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = e
}
func (h *HfMirror) CacheTTL() string {
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

func (h *HfMirror) ApplyConfig(cfg config.MirrorConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = cfg.Enabled
	h.upstream = cfg.Upstream
	h.cacheTTL = cfg.CacheTTLd
}

func (h *HfMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	h.mu.RLock()
	upstream := h.upstream
	h.mu.RUnlock()
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/hf"):]
		// HuggingFace files can be large (model weights), use streaming proxy
		cache.ProxyStream(w, r, upstream)
	}
}

func (h *HfMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cacheTTL = d
	return nil
}

func (h *HfMirror) HealthCheck() error {
	resp, err := HealthGet(h.upstream + "/")
	if err != nil {
		return fmt.Errorf("huggingface upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("huggingface upstream returned %d", resp.StatusCode)
	}
	return nil
}
