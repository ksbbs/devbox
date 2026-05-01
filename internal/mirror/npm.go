package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type NpmMirror struct {
	enabled   bool
	upstream  string
	cacheTTL  time.Duration
}

func init() {
	Register(&NpmMirror{})
}

func (n *NpmMirror) Name() string      { return "npm" }
func (n *NpmMirror) Pattern() string    { return "/npm/" }
func (n *NpmMirror) Upstream() string   { return n.upstream }
func (n *NpmMirror) SetUpstream(url string) { n.upstream = url }
func (n *NpmMirror) IsEnabled() bool    { return n.enabled }
func (n *NpmMirror) SetEnabled(e bool)  { n.enabled = e }
func (n *NpmMirror) CacheTTL() string   { return fmt.Sprintf("%d", n.cacheTTL/time.Second) }

func (n *NpmMirror) ApplyConfig(cfg config.MirrorConfig) {
	n.enabled = cfg.Enabled
	n.upstream = cfg.Upstream
	n.cacheTTL = cfg.CacheTTLd
}

func (n *NpmMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// strip "/npm" prefix, pass remaining path to upstream
		r.URL.Path = r.URL.Path[len("/npm"):]
		if r.URL.Path == "" || r.URL.Path == "/" {
			r.URL.Path = "/"
		}
		cache.ProxyHTTP(w, r, n.upstream, n.cacheTTL)
	}
}

func (n *NpmMirror) HealthCheck() error {
	resp, err := http.Get(n.upstream + "/")
	if err != nil {
		return fmt.Errorf("npm upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("npm upstream returned %d", resp.StatusCode)
	}
	return nil
}