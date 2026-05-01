package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"devbox/internal/config"
	"devbox/internal/dashboard"
	"devbox/internal/gitproxy"
	"devbox/internal/mirror"
	"devbox/internal/ratelimit"
	"devbox/internal/store"
)

// tokenCache stores registry auth tokens with expiry
type tokenCache struct {
	mu    sync.RWMutex
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

type Server struct {
	cfg        *config.Config
	cache      *mirror.Cache
	gitProxy   *gitproxy.GitProxy
	dash       *dashboard.Dashboard
	store      *store.Store
	limiter    *ratelimit.Limiter
	search     *dashboard.SearchHandler
	tokenCache *tokenCache
	// Custom client: strip Authorization when following 307 redirects to CDN
	// (Docker Hub blob storage on Cloudflare rejects auth headers)
	registryClient *http.Client
	frontDir       string
}

func New(cfg *config.Config, frontDir string) (*Server, error) {
	st, err := store.New(cfg.Cache.Dir + "/../devbox.db")
	if err != nil {
		return nil, fmt.Errorf("init store: %w", err)
	}

	cache := mirror.NewCache(cfg.Cache.Dir, cfg.Cache.MaxSizeBytes)

	gp := gitproxy.New(
		cfg.GitProxy.GithubUpstream,
		cfg.GitProxy.GitlabUpstream,
		cfg.GitProxy.CacheTTLd,
		cfg.Cache.Dir,
	)

	for name, mCfg := range cfg.Mirrors {
		m, ok := mirror.Get(name)
		if ok {
			m.SetEnabled(mCfg.Enabled)
			if mCfg.Upstream != "" {
				m.SetUpstream(mCfg.Upstream)
			}
		}
	}

	dash := dashboard.New(st, cfg.Server.AuthToken, cfg.Server.PublicURL)

	var limiter *ratelimit.Limiter
	if cfg.RateLimit.Enabled {
		limiter = ratelimit.New(cfg.RateLimit.Rate, cfg.RateLimit.IntervalDur, cfg.RateLimit.Whitelist, cfg.RateLimit.Blacklist)
	}

	s := &Server{
		cfg:            cfg,
		cache:          cache,
		gitProxy:       gp,
		dash:           dash,
		store:          st,
		limiter:        limiter,
		search:         dashboard.NewSearchHandler(),
		tokenCache:     &tokenCache{tokens: make(map[string]*cachedToken)},
		registryClient: newRegistryClient(),
		frontDir:       frontDir,
	}

	dash.SetRateLimitConfigAccessor(s)

	return s, nil
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Mirror proxy routes
	for _, m := range mirror.All() {
		if m.IsEnabled() {
			handler := m.ProxyHandler(s.cache)
			mux.HandleFunc(m.Pattern(), s.wrapWithStats(m.Name(), handler))
		}
	}

	// Git proxy routes
	if s.cfg.GitProxy.Enabled {
		mux.HandleFunc("/gh/", s.wrapWithStats("gitproxy", s.gitProxy.Handler))
		mux.HandleFunc("/gl/", s.wrapWithStats("gitproxy", s.gitProxy.Handler))
	}

	// Dashboard API routes
	mux.HandleFunc("/api/status", s.dash.StatusHandler)
	mux.HandleFunc("/api/stats/traffic", s.dash.TrafficHandler)
	mux.HandleFunc("/api/stats/logs", s.dash.LogHandler)
	mux.HandleFunc("/api/config/mirrors", s.dash.MirrorConfigHandler)
	mux.HandleFunc("/api/config/ratelimit", s.dash.RateLimitConfigHandler)
	mux.HandleFunc("/api/config/public", s.dash.PublicConfigHandler)
	mux.HandleFunc("/api/auth/login", s.dash.LoginHandler)
	mux.HandleFunc("/api/search", s.search.Search)

	// Docker v2 registry API — proxy handles auth transparently
	mux.HandleFunc("/v2/", s.registryV2Handler)

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
			log.Printf("frontend dir %s not found, serving API only", s.frontDir)
		}
	}

	go s.cacheCleanup()
	go s.trafficCleanup()
	go s.rateLimitCleanup()
	go s.tokenCacheCleanup()

	addr := fmt.Sprintf(":%d", s.cfg.Server.Port)
	log.Printf("DevBox starting on %s", addr)

	handler := logMiddleware(mux, s.cfg.Logging.AccessLog)
	if s.limiter != nil {
		handler = s.rateLimitMiddleware(handler)
		log.Printf("rate limiting enabled: %d requests per %s", s.cfg.RateLimit.Rate, s.cfg.RateLimit.Interval)
	}
	return http.ListenAndServe(addr, handler)
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
		log.Printf("[registry] token error for %s: %v", registryName, err)
		// Try without token (some repos are public)
		s.proxyRegistryRequest(w, r, target, "")
		return
	}

	log.Printf("[registry] %s %s → %s (token=%s...)", r.Method, path, target, token[:min(10, len(token))])
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
		log.Printf("[registry] upstream error: %v", err)
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
		log.Printf("[registry] token rejected, retrying without token")
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
				log.Printf("[registry] got new token with scope=%s, retrying", scope)
				s.proxyRegistryRequest(w, r, target, newToken)
				return
			}
		}

		// Can't get token, return 401 with rewritten WWW-Authenticate
		// to let Docker client try to authenticate itself
		if wwAuth != "" {
			proxyHost := s.cfg.Server.PublicURL
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

	resp, err := http.Get(url)
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

	log.Printf("[token] got token for %s scope=%s expires_in=%ds", regInfo.service, scope, tokenResp.ExpiresIn)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Server) GetRateLimitConfig() dashboard.RateLimitConfigView {
	return dashboard.RateLimitConfigView{
		Enabled:   s.cfg.RateLimit.Enabled,
		Rate:      s.cfg.RateLimit.Rate,
		Interval:  s.cfg.RateLimit.Interval,
		Whitelist: s.cfg.RateLimit.Whitelist,
		Blacklist: s.cfg.RateLimit.Blacklist,
	}
}

func (s *Server) SetRateLimitEnabled(enabled bool) {
	s.cfg.RateLimit.Enabled = enabled
}

func (s *Server) SetRateLimitRate(rate int) {
	s.cfg.RateLimit.Rate = rate
}

func (s *Server) SetRateLimitWhitelist(list []string) {
	s.cfg.RateLimit.Whitelist = list
}

func (s *Server) SetRateLimitBlacklist(list []string) {
	s.cfg.RateLimit.Blacklist = list
}

func (s *Server) wrapWithStats(name string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w}
		start := time.Now()
		handler(sw, r)
		s.store.RecordTraffic(name, r.Method, r.URL.Path, 0, sw.bytesWritten, sw.status)
		log.Printf("[%s] %s %s %d %dms %dB",
			name, r.Method, r.URL.Path, sw.status,
			time.Since(start).Milliseconds(), sw.bytesWritten)
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
		n, err := s.store.PurgeOldTraffic(s.cfg.Logging.RetentionDays)
		if err != nil {
			log.Printf("traffic cleanup error: %v", err)
		} else if n > 0 {
			log.Printf("purged %d old traffic records (retention: %d days)", n, s.cfg.Logging.RetentionDays)
		}
	}
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
		log.Printf("%s %s %s %dms",
			r.Method, r.URL.Path, r.RemoteAddr,
			time.Since(start).Milliseconds())
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
		if !s.limiter.Allow(r) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) rateLimitCleanup() {
	if s.limiter == nil {
		return
	}
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		s.limiter.Cleanup()
	}
}