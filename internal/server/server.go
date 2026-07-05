package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"devbox/internal/alert"
	"devbox/internal/config"
	"devbox/internal/dashboard"
	"devbox/internal/gitproxy"
	"devbox/internal/mirror"
	"devbox/internal/ratelimit"
	"devbox/internal/store"
)

type configApplier interface {
	ApplyConfig(cfg config.MirrorConfig)
}

// tokenCache stores registry auth tokens with expiry
type tokenCache struct {
	mu     sync.RWMutex
	tokens map[string]*cachedToken
}

type cachedToken struct {
	token     string
	expiresAt time.Time
}

type registryInfo struct {
	upstream string
	authURL  string
	service  string
}

var registries = map[string]registryInfo{
	"docker": {upstream: "https://registry-1.docker.io", authURL: "https://auth.docker.io/token", service: "registry.docker.io"},
	"ghcr":   {upstream: "https://ghcr.io", authURL: "https://ghcr.io/token", service: "ghcr.io"},
	"quay":   {upstream: "https://quay.io", authURL: "https://quay.io/v2/auth", service: "quay.io"},
	"mcr":    {upstream: "https://mcr.microsoft.com", authURL: "https://mcr.microsoft.com/v2/auth", service: "mcr.microsoft.com"},
}

var authClient = &http.Client{Timeout: 15 * time.Second}

type Server struct {
	cfg         *config.Config
	cfgMu       sync.RWMutex
	configPath  string
	cache       *mirror.Cache
	gitProxy    *gitproxy.GitProxy
	dash        *dashboard.Dashboard
	store       *store.Store
	limiterMu   sync.RWMutex
	limiter     *ratelimit.Limiter
	search      *dashboard.SearchHandler
	alertEngine *alert.Engine
	tokenCache  *tokenCache
	// Custom client: strip Authorization when following 307 redirects to CDN
	// (Docker Hub blob storage on Cloudflare rejects auth headers)
	registryClient *http.Client
	httpServer     *http.Server
	frontDir       string
}

func New(cfg *config.Config, configPath string, frontDir string) (*Server, error) {
	dbDir := filepath.Dir(filepath.Clean(cfg.Cache.Dir))
	dbPath := filepath.Join(dbDir, "devbox.db")
	st, err := store.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("init store: %w", err)
	}

	cache := mirror.NewCache(cfg.Cache.Dir, cfg.Cache.MaxSizeBytes)

	gp := gitproxy.New(
		cfg.GitProxy.GithubUpstream,
		cfg.GitProxy.GitlabUpstream,
		cfg.GitProxy.RawUpstream,
		cfg.GitProxy.CacheTTLd,
		cache,
	)

	for name, mCfg := range cfg.Mirrors {
		m, ok := mirror.Get(name)
		if ok {
			if applier, ok2 := m.(configApplier); ok2 {
				applier.ApplyConfig(mCfg)
			} else {
				m.SetEnabled(mCfg.Enabled)
				if mCfg.Upstream != "" {
					m.SetUpstream(mCfg.Upstream)
				}
			}
		}
	}

	dash := dashboard.New(st, cfg.Server.AuthToken, cfg.Server.PublicURL)

	ae := alert.NewEngine(cfg.Alerts.CooldownD)
	if cfg.Alerts.WebhookURL != "" {
		ae.AddProvider(alert.NewWebhook(cfg.Alerts.WebhookURL))
	} else {
		ae.AddProvider(alert.NewLog())
	}
	dash.SetAlertEngine(ae)

	var limiter *ratelimit.Limiter
	if cfg.RateLimit.Enabled {
		limiter = ratelimit.New(cfg.RateLimit.Rate, cfg.RateLimit.IntervalDur, cfg.RateLimit.Whitelist, cfg.RateLimit.Blacklist)
	}

	s := &Server{
		cfg:            cfg,
		configPath:     configPath,
		cache:          cache,
		gitProxy:       gp,
		dash:           dash,
		store:          st,
		limiter:        limiter,
		search:         dashboard.NewSearchHandler(),
		alertEngine:    ae,
		tokenCache:     &tokenCache{tokens: make(map[string]*cachedToken)},
		registryClient: newRegistryClient(),
		frontDir:       frontDir,
	}

	dash.SetRateLimitConfigAccessor(s)
	dash.SetSaveConfig(s.saveConfig)

	return s, nil
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Mirror proxy routes — register all regardless of enabled state
	// so enabling/disabling at runtime works without mux rebuild
	for _, m := range mirror.All() {
		handler := s.mirrorEnabledWrapper(m, m.ProxyHandler(s.cache))
		mux.HandleFunc(m.Pattern(), s.wrapWithStats(m.Name(), handler))
	}

	// Git proxy routes
	mux.HandleFunc("/gh/", s.wrapWithStats("gitproxy", s.gitProxyHandler))
	mux.HandleFunc("/gl/", s.wrapWithStats("gitproxy", s.gitProxyHandler))

	// Dashboard API routes
	mux.HandleFunc("/api/status", s.dash.StatusHandler)
	mux.HandleFunc("/api/stats/traffic", s.dash.TrafficHandler)
	mux.HandleFunc("/api/stats/logs", s.dash.LogHandler)
	mux.HandleFunc("/api/config/mirrors", s.dash.MirrorConfigHandler)
	mux.HandleFunc("/api/config/ratelimit", s.dash.RateLimitConfigHandler)
	mux.HandleFunc("/api/config/public", s.dash.PublicConfigHandler)
	mux.HandleFunc("/api/auth/login", s.dash.LoginHandler)
	mux.HandleFunc("/api/auth/check", s.dash.AuthCheckHandler)
	mux.HandleFunc("/api/search", s.search.Search)

	// Docker v2 registry API — proxy handles auth transparently
	mux.HandleFunc("/v2/", s.wrapWithDynamicStats(registryStatsName, s.registryV2Handler))

	// Health check endpoint (no auth, for k8s/docker probes)
	mux.HandleFunc("/health", s.healthHandler)

	// Prometheus metrics endpoint
	mux.HandleFunc("/metrics", s.metricsHandler)

	// Frontend static files
	if s.frontDir != "" {
		if _, err := os.Stat(s.frontDir); err == nil {
			fileServer := http.FileServer(http.Dir(s.frontDir))
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path
				if path == "/" {
					path = "/index.html"
				}
				if _, err := os.Stat(s.frontDir + path); err != nil {
					r.URL.Path = "/index.html"
				}
				fileServer.ServeHTTP(w, r)
			})
		} else {
			slog.Info("frontend dir not found, serving API only", "dir", s.frontDir)
		}
	}

	go s.cacheCleanup()
	go s.trafficCleanup()
	go s.rateLimitCleanup()
	go s.configWatcher()
	go s.tokenCacheCleanup()

	port, accessLog := s.serverConfigSnapshot()
	rl := s.rateLimitConfigSnapshot()
	addr := fmt.Sprintf(":%d", port)
	slog.Info("DevBox starting", "addr", addr)

	handler := logMiddleware(mux, accessLog)
	handler = s.bodyLimitMiddleware(handler)
	if s.getLimiter() != nil {
		handler = s.rateLimitMiddleware(handler)
		slog.Info("rate limiting enabled", "rate", rl.Rate, "interval", rl.Interval)
	}

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       0, // streaming uploads (e.g. git push) need no read timeout
		WriteTimeout:      0, // large downloads need no write timeout
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server and closes the store.
func (s *Server) Shutdown(ctx context.Context) error {
	var firstErr error
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			slog.Error("HTTP server shutdown error", "error", err)
			firstErr = err
		}
	}
	s.Close()
	return firstErr
}

func (s *Server) registryV2Handler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Docker v2 API ping — always return 200 (auth handled transparently by proxy)
	if path == "/v2/" || path == "/v2" {
		w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
		w.WriteHeader(http.StatusOK)
		return
	}

	// /v2/{registry}/{path} → extract registry name
	rest := strings.TrimPrefix(path, "/v2/")
	parts := strings.SplitN(rest, "/", 2)
	registryName := parts[0]
	if len(parts) < 2 {
		http.Error(w, "invalid registry path", http.StatusBadRequest)
		return
	}

	regInfo, ok := registries[registryName]
	if !ok {
		http.Error(w, "unknown registry", http.StatusNotFound)
		return
	}

	// Build upstream URL (strip registry alias, keep real path)
	target := regInfo.upstream + "/v2/" + parts[1]
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	// Derive auth scope from path for token request
	scope := deriveScope(registryName, parts[1])

	// Get token (cached or fresh)
	token, err := s.getRegistryToken(regInfo, scope)
	if err != nil {
		slog.Error("registry token error", "registry", registryName, "error", err)
		// Try without token (some repos are public)
		s.proxyRegistryRequest(w, r, target, "")
		return
	}

	slog.Info("registry request", "method", r.Method, "path", path, "target", target, "token_prefix", token[:min(10, len(token))])
	s.proxyRegistryRequest(w, r, target, token)
}

func newRegistryClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Strip Authorization when redirecting to a different host
			// (CDN blob storage like Cloudflare rejects auth headers)
			if len(via) > 0 && req.URL.Host != via[0].URL.Host {
				req.Header.Del("Authorization")
			}
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

func (s *Server) proxyRegistryRequest(w http.ResponseWriter, r *http.Request, target string, token string) {
	upstreamReq, err := http.NewRequest(r.Method, target, r.Body)
	if err != nil {
		http.Error(w, "request error", http.StatusInternalServerError)
		return
	}

	// Copy client headers (except Host)
	for k, vv := range r.Header {
		if k == "Host" {
			continue
		}
		for _, v := range vv {
			upstreamReq.Header.Add(k, v)
		}
	}

	// Inject our proxy token if available (replaces any client token)
	if token != "" {
		upstreamReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := s.registryClient.Do(upstreamReq)
	if err != nil {
		slog.Error("registry upstream error", "error", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// If 401 with our token, clear cache and retry once
	if resp.StatusCode == 401 && token != "" {
		// Clear stale token from cache using proper key
		rest := strings.TrimPrefix(r.URL.Path, "/v2/")
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) >= 1 {
			regInfo, ok := registries[parts[0]]
			if ok {
				scope := deriveScope(parts[0], parts[1])
				cacheKey := regInfo.service + ":" + scope
				s.tokenCache.mu.Lock()
				delete(s.tokenCache.tokens, cacheKey)
				s.tokenCache.mu.Unlock()
			}
		}

		resp.Body.Close()
		// Retry without token — let upstream give fresh 401
		slog.Warn("registry token rejected, retrying without token")
		s.proxyRegistryRequest(w, r, target, "")
		return
	}

	// If 401 without token, get a new token with scope and retry
	if resp.StatusCode == 401 && token == "" {
		// Parse scope from WWW-Authenticate header
		wwAuth := resp.Header.Get("Www-Authenticate")
		scope := extractScopeFromAuthHeader(wwAuth)

		// Find registry info from path
		rest := strings.TrimPrefix(r.URL.Path, "/v2/")
		parts := strings.SplitN(rest, "/", 2)
		regInfo, ok := registries[parts[0]]

		if ok && scope != "" {
			newToken, err := s.getRegistryToken(regInfo, scope)
			if err == nil && newToken != "" {
				resp.Body.Close()
				slog.Info("registry got new token, retrying", "scope", scope)
				s.proxyRegistryRequest(w, r, target, newToken)
				return
			}
		}

		// Can't get token, return 401 with rewritten WWW-Authenticate
		// to let Docker client try to authenticate itself
		if wwAuth != "" {
			proxyHost := s.publicURL()
			if proxyHost == "" {
				proxyHost = "http://" + r.Host
			}
			proxyHost = strings.TrimRight(proxyHost, "/")
			for _, domain := range []string{
				"https://auth.docker.io",
				"https://ghcr.io",
				"https://quay.io",
				"https://mcr.microsoft.com",
			} {
				wwAuth = strings.ReplaceAll(wwAuth, domain, proxyHost)
			}
			w.Header().Set("Www-Authenticate", wwAuth)
		}
		w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
		w.WriteHeader(401)
		io.Copy(w, resp.Body)
		return
	}

	// Copy response headers
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	for k, vv := range resp.Header {
		if k == "Docker-Distribution-API-Version" {
			continue
		}
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) getRegistryToken(regInfo registryInfo, scope string) (string, error) {
	// Check cache first
	cacheKey := regInfo.service + ":" + scope
	s.tokenCache.mu.RLock()
	ct, ok := s.tokenCache.tokens[cacheKey]
	s.tokenCache.mu.RUnlock()
	if ok && time.Now().Before(ct.expiresAt) {
		return ct.token, nil
	}

	// Get fresh token from upstream auth server
	url := regInfo.authURL + "?service=" + regInfo.service
	if scope != "" {
		url += "&scope=" + scope
	}

	resp, err := authClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("auth server returned %d", resp.StatusCode)
	}

	var tokenResp struct {
		Token     string `json:"token"`
		ExpiresIn int    `json:"expires_in"` // seconds
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}

	if tokenResp.Token == "" {
		return "", fmt.Errorf("empty token")
	}

	// Cache token (use expires_in minus 5 min safety margin, min 2 min)
	ttl := time.Duration(tokenResp.ExpiresIn) * time.Second
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	ttl = ttl - 5*time.Minute
	if ttl < 2*time.Minute {
		ttl = 2 * time.Minute
	}

	s.tokenCache.mu.Lock()
	s.tokenCache.tokens[cacheKey] = &cachedToken{
		token:     tokenResp.Token,
		expiresAt: time.Now().Add(ttl),
	}
	s.tokenCache.mu.Unlock()

	slog.Info("got token", "service", regInfo.service, "scope", scope, "expires_in", tokenResp.ExpiresIn)
	return tokenResp.Token, nil
}

func (s *Server) tokenCacheCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.tokenCache.mu.Lock()
		now := time.Now()
		for k, v := range s.tokenCache.tokens {
			if now.After(v.expiresAt) {
				delete(s.tokenCache.tokens, k)
			}
		}
		s.tokenCache.mu.Unlock()
	}
}

func deriveScope(registryName string, path string) string {
	// Derive scope from path: owner/image/manifests/ref → repository:owner/image:pull
	segments := strings.Split(path, "/")
	if len(segments) < 2 {
		return ""
	}
	// Detect if segments[1] is an action keyword (manifests/blobs/tags)
	// vs an image name. Official images like nginx have path: nginx/manifests/latest
	actions := map[string]bool{"manifests": true, "blobs": true, "tags": true}
	if len(segments) >= 3 && actions[segments[1]] {
		// Official image (no namespace): nginx → library/nginx
		repo := segments[0]
		if registryName == "docker" {
			repo = "library/" + repo
		}
		return "repository:" + repo + ":pull"
	}
	if len(segments) >= 3 {
		repo := segments[0] + "/" + segments[1]
		return "repository:" + repo + ":pull"
	}
	return ""
}

func extractScopeFromAuthHeader(header string) string {
	// Parse scope from WWW-Authenticate: Bearer realm="...",service="...",scope="..."
	idx := strings.Index(header, "scope=")
	if idx == -1 {
		return ""
	}
	start := idx + 6
	if start < len(header) && header[start] == '"' {
		end := strings.Index(header[start+1:], "\"")
		if end != -1 {
			return header[start+1 : start+1+end]
		}
	}
	// Unquoted scope
	end := strings.Index(header[start:], ",")
	if end == -1 {
		return header[start:]
	}
	return header[start : start+end]
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// bodyLimitMiddleware caps request body size to prevent memory exhaustion.
// Mirror proxy and git proxy paths are exempt (large uploads/downloads).
const maxBodySize = 10 << 20 // 10MB

func (s *Server) bodyLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Only limit API and auth endpoints
		if strings.HasPrefix(path, "/api/") {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	hits := s.cache.Hits()
	misses := s.cache.Misses()
	total := hits + misses

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "# HELP devbox_cache_hits_total Cache hits\n")
	fmt.Fprintf(w, "# TYPE devbox_cache_hits_total counter\n")
	fmt.Fprintf(w, "devbox_cache_hits_total %d\n", hits)
	fmt.Fprintf(w, "# HELP devbox_cache_misses_total Cache misses\n")
	fmt.Fprintf(w, "# TYPE devbox_cache_misses_total counter\n")
	fmt.Fprintf(w, "devbox_cache_misses_total %d\n", misses)
	if total > 0 {
		fmt.Fprintf(w, "# HELP devbox_cache_hit_ratio Cache hit ratio\n")
		fmt.Fprintf(w, "# TYPE devbox_cache_hit_ratio gauge\n")
		fmt.Fprintf(w, "devbox_cache_hit_ratio %0.4f\n", float64(hits)/float64(total))
	}
	fmt.Fprintf(w, "# HELP devbox_mirrors_total Total registered mirrors\n")
	fmt.Fprintf(w, "# TYPE devbox_mirrors_total gauge\n")
	fmt.Fprintf(w, "devbox_mirrors_total %d\n", len(mirror.All()))
}

func (s *Server) mirrorEnabledWrapper(m mirror.Mirror, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.IsEnabled() {
			http.Error(w, "mirror disabled", http.StatusServiceUnavailable)
			return
		}
		next(w, r)
	}
}

func (s *Server) configWatcher() {
	if s.configPath == "" {
		return
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	var lastMtime time.Time
	if fi, err := os.Stat(s.configPath); err == nil {
		lastMtime = fi.ModTime()
	}

	for range ticker.C {
		fi, err := os.Stat(s.configPath)
		if err != nil {
			continue
		}
		if fi.ModTime().Equal(lastMtime) {
			continue
		}
		lastMtime = fi.ModTime()

		cfg, err := config.Load(s.configPath)
		if err != nil {
			slog.Error("config hot-reload failed to load, keeping old config", "error", err)
			continue
		}

		s.applyRuntimeConfig(cfg)
		slog.Info("config hot-reloaded", "path", s.configPath)
	}
}

func (s *Server) applyRuntimeConfig(cfg *config.Config) {
	// Apply mirror settings
	for name, mCfg := range cfg.Mirrors {
		m, ok := mirror.Get(name)
		if ok {
			if applier, ok2 := m.(configApplier); ok2 {
				applier.ApplyConfig(mCfg)
			}
		}
	}

	// Rebuild rate limiter
	s.limiterMu.Lock()
	if cfg.RateLimit.Enabled {
		s.limiter = ratelimit.New(cfg.RateLimit.Rate, cfg.RateLimit.IntervalDur, cfg.RateLimit.Whitelist, cfg.RateLimit.Blacklist)
	} else {
		s.limiter = nil
	}
	s.limiterMu.Unlock()

	gp := gitproxy.New(
		cfg.GitProxy.GithubUpstream,
		cfg.GitProxy.GitlabUpstream,
		cfg.GitProxy.RawUpstream,
		cfg.GitProxy.CacheTTLd,
		s.cache,
	)

	// Swap runtime config atomically.
	s.cfgMu.Lock()
	s.gitProxy = gp
	s.cfg = cfg
	s.cfgMu.Unlock()
}

func (s *Server) gitProxyHandler(w http.ResponseWriter, r *http.Request) {
	s.cfgMu.RLock()
	gp := s.gitProxy
	s.cfgMu.RUnlock()
	gp.Handler(w, r)
}

func (s *Server) saveConfig() error {
	if s.configPath == "" {
		return nil
	}

	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()

	for _, m := range mirror.All() {
		existing, ok := s.cfg.Mirrors[m.Name()]
		if !ok {
			existing = config.MirrorConfig{}
		}
		existing.Enabled = m.IsEnabled()
		existing.Upstream = m.Upstream()
		existing.CacheTTL = m.CacheTTL()
		s.cfg.Mirrors[m.Name()] = existing
	}

	if err := s.cfg.Save(s.configPath); err != nil {
		slog.Error("config failed to persist", "error", err)
		return err
	}
	slog.Info("config persisted", "path", s.configPath)
	return nil
}

func (s *Server) GetRateLimitConfig() dashboard.RateLimitConfigView {
	rl := s.rateLimitConfigSnapshot()
	return dashboard.RateLimitConfigView{
		Enabled:   rl.Enabled,
		Rate:      rl.Rate,
		Interval:  rl.Interval,
		Whitelist: rl.Whitelist,
		Blacklist: rl.Blacklist,
	}
}

func (s *Server) SetRateLimitEnabled(enabled bool) {
	s.cfgMu.Lock()
	s.cfg.RateLimit.Enabled = enabled
	s.cfgMu.Unlock()
	s.rebuildLimiter()
}

func (s *Server) SetRateLimitRate(rate int) {
	s.cfgMu.Lock()
	s.cfg.RateLimit.Rate = rate
	s.cfgMu.Unlock()
	s.rebuildLimiter()
}

func (s *Server) SetRateLimitInterval(interval string) {
	if d, err := config.ParseDuration(interval); err == nil {
		s.cfgMu.Lock()
		s.cfg.RateLimit.Interval = interval
		s.cfg.RateLimit.IntervalDur = d
		s.cfgMu.Unlock()
		s.rebuildLimiter()
	}
}

func (s *Server) SetRateLimitWhitelist(list []string) {
	s.cfgMu.Lock()
	s.cfg.RateLimit.Whitelist = list
	s.cfgMu.Unlock()
	s.rebuildLimiter()
}

func (s *Server) SetRateLimitBlacklist(list []string) {
	s.cfgMu.Lock()
	s.cfg.RateLimit.Blacklist = list
	s.cfgMu.Unlock()
	s.rebuildLimiter()
}

func (s *Server) rebuildLimiter() {
	rl := s.rateLimitConfigSnapshot()
	s.limiterMu.Lock()
	defer s.limiterMu.Unlock()
	if !rl.Enabled {
		s.limiter = nil
		return
	}
	s.limiter = ratelimit.New(rl.Rate, rl.IntervalDur, rl.Whitelist, rl.Blacklist)
}

func (s *Server) serverConfigSnapshot() (int, bool) {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg.Server.Port, s.cfg.Logging.AccessLog
}

func (s *Server) publicURL() string {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg.Server.PublicURL
}

func (s *Server) rateLimitConfigSnapshot() config.RateLimitConfig {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg.RateLimit
}

func (s *Server) getLimiter() *ratelimit.Limiter {
	s.limiterMu.RLock()
	defer s.limiterMu.RUnlock()
	return s.limiter
}

func registryStatsName(r *http.Request) string {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/v2/"), "/", 2)
	if len(parts) > 1 {
		if _, ok := registries[parts[0]]; ok {
			return parts[0]
		}
	}
	return "docker"
}

func (s *Server) wrapWithStats(name string, handler http.HandlerFunc) http.HandlerFunc {
	return s.wrapWithDynamicStats(func(*http.Request) string { return name }, handler)
}

func (s *Server) wrapWithDynamicStats(nameFor func(*http.Request) string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w}
		start := time.Now()
		handler(sw, r)
		name := nameFor(r)
		s.store.RecordTraffic(name, r.Method, r.URL.Path, 0, sw.bytesWritten, sw.status)
		slog.Info("request stats",
			"name", name,
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes_written", sw.bytesWritten)
	}
}

func (s *Server) cacheCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		s.cache.CleanExpired()
	}
}

func (s *Server) trafficCleanup() {
	ticker := time.NewTicker(6 * time.Hour)
	for range ticker.C {
		retentionDays := s.loggingRetentionDays()
		n, err := s.store.PurgeOldTraffic(retentionDays)
		if err != nil {
			slog.Error("traffic cleanup error", "error", err)
		} else if n > 0 {
			slog.Info("purged old traffic records", "count", n, "retention_days", retentionDays)
		}
	}
}

func (s *Server) loggingRetentionDays() int {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg.Logging.RetentionDays
}

func (s *Server) Close() {
	s.store.Close()
}

type statusWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Write(b []byte) (int, error) {
	if sw.status == 0 {
		sw.status = 200
	}
	n, err := sw.ResponseWriter.Write(b)
	sw.bytesWritten += n
	return n, err
}

func logMiddleware(next http.Handler, accessLog bool) http.Handler {
	if !accessLog {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("access log",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
			"duration_ms", time.Since(start).Milliseconds())
	})
}

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/v2/") ||
			strings.HasPrefix(path, "/token") || path == "/" ||
			strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".js") ||
			strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".ico") {
			next.ServeHTTP(w, r)
			return
		}
		limiter := s.getLimiter()
		if limiter != nil && !limiter.Allow(r) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) rateLimitCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		if limiter := s.getLimiter(); limiter != nil {
			limiter.Cleanup()
		}
	}
}
