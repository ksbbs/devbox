package mirror

import (
	"fmt"
	"net/http"
	"time"

	"devbox/internal/config"
)

type RubyGemsMirror struct {
	enabled  bool
	upstream string
	cacheTTL time.Duration
}

func init() {
	Register(&RubyGemsMirror{})
}

func (rg *RubyGemsMirror) Name() string           { return "rubygems" }
func (rg *RubyGemsMirror) Pattern() string        { return "/rubygems/" }
func (rg *RubyGemsMirror) Upstream() string       { return rg.upstream }
func (rg *RubyGemsMirror) SetUpstream(url string) { rg.upstream = url }
func (rg *RubyGemsMirror) IsEnabled() bool        { return rg.enabled }
func (rg *RubyGemsMirror) SetEnabled(e bool)      { rg.enabled = e }
func (rg *RubyGemsMirror) CacheTTL() string       { return fmt.Sprintf("%d", rg.cacheTTL/time.Second) }

func (rg *RubyGemsMirror) ApplyConfig(cfg config.MirrorConfig) {
	rg.enabled = cfg.Enabled
	rg.upstream = cfg.Upstream
	rg.cacheTTL = cfg.CacheTTLd
}

func (rg *RubyGemsMirror) ProxyHandler(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/rubygems"):]
		cache.ProxyHTTP(w, r, rg.upstream, rg.cacheTTL)
	}
}

func (rg *RubyGemsMirror) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	rg.cacheTTL = d
	return nil
}

func (rg *RubyGemsMirror) HealthCheck() error {
	resp, err := HealthGet(rg.upstream + "/")
	if err != nil {
		return fmt.Errorf("rubygems upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rubygems upstream returned %d", resp.StatusCode)
	}
	return nil
}
