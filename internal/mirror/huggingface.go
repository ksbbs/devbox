package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type HfMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&HfMirror{})
}

func (h *HfMirror) Name() string            { return "hf" }
func (h *HfMirror) Pattern() string         { return "/hf/" }
func (h *HfMirror) Upstream() string        { return h.upstream }
func (h *HfMirror) SetUpstream(url string)  { h.upstream = url }
func (h *HfMirror) IsEnabled() bool         { return h.enabled }
func (h *HfMirror) SetEnabled(e bool)       { h.enabled = e }
func (h *HfMirror) CacheTTL() string        { return fmt.Sprintf("%d", h.cacheTTL/time.Second) }

func (h *HfMirror) ApplyConfig(cfg config.MirrorConfig) {
	h.enabled = cfg.Enabled
	h.upstream = cfg.Upstream
	h.cacheTTL = cfg.CacheTTLd
}

func (h *HfMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/hf"):]
		// HuggingFace files can be large (model weights), use streaming proxy
		cache.ProxyStream(w, r, h.upstream)
	}
}

func (h *HfMirror) HealthCheck() error {
	resp, err := http.Get(h.upstream + "/")
	if err != nil {
		return fmt.Errorf("huggingface upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("huggingface upstream returned %d", resp.StatusCode)
	}
	return nil
}