package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"devbox/internal/config"
	"devbox/internal/dashboard"
	"devbox/internal/gitproxy"
	"devbox/internal/mirror"
	"devbox/internal/ratelimit"
	"devbox/internal/store"
)

type Server struct {
	cfg        *config.Config
	cache      *mirror.Cache
	gitProxy   *gitproxy.GitProxy
	dash       *dashboard.Dashboard
	store      *store.Store
	limiter    *ratelimit.Limiter
	search     *dashboard.SearchHandler
	frontDir   string
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
		limiter = ratelimit.New(cfg.RateLimit.Rate, cfg.RateLimit.IntervalDur, cfg.RateLimit.Whitelist)
	}

	search := dashboard.NewSearchHandler()

	return &Server{
		cfg:      cfg,
		cache:    cache,
		gitProxy: gp,
		dash:     dash,
		store:    st,
		limiter:  limiter,
		search:   search,
		frontDir: frontDir,
	}, nil
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
	mux.HandleFunc("/api/config/public", s.dash.PublicConfigHandler)
	mux.HandleFunc("/api/auth/login", s.dash.LoginHandler)

	// Mirror search API
	mux.HandleFunc("/api/search", s.search.Search)

	// Docker v2 registry API routes
	// Allows: docker pull <host>/ghcr/owner/image:tag
	// Docker sends /v2/ ping first, then /v2/{registry}/... requests
	mux.HandleFunc("/v2/", s.registryV2Handler)

	// Docker v2 token auth proxy — intercept token requests
	// When upstream returns 401 + WWW-Authenticate, Docker needs a token
	// We proxy the token request so Docker doesn't need direct upstream access
	mux.HandleFunc("/token", s.tokenAuthHandler)

	// Frontend static files from directory
	if s.frontDir != "" {
		if _, err := os.Stat(s.frontDir); err == nil {
			fileServer := http.FileServer(http.Dir(s.frontDir))
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path
				if path == "/" {
					path = "/index.html"
				}
				// Check if file exists on disk
				if _, err := os.Stat(s.frontDir + path); err != nil {
					// SPA fallback to index.html
					r.URL.Path = "/index.html"
				}
				fileServer.ServeHTTP(w, r)
			})
		} else {
			log.Printf("frontend dir %s not found, serving API only", s.frontDir)
		}
	}

	// Start cache cleanup timer
	go s.cacheCleanup()

	// Start traffic log cleanup timer
	go s.trafficCleanup()

	// Start rate limiter cleanup timer
	go s.rateLimitCleanup()

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

	// Docker v2 API ping: /v2/ → return API version header
	if path == "/v2/" || path == "/v2" {
		w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
		w.WriteHeader(http.StatusOK)
		return
	}

	// /v2/{registry}/... → proxy to corresponding upstream registry
	registryMap := map[string]string{
		"docker": "https://registry-1.docker.io",
		"ghcr":   "https://ghcr.io",
		"quay":   "https://quay.io",
		"mcr":    "https://mcr.microsoft.com",
	}

	// Extract registry name from path: /v2/{registry}/...
	rest := strings.TrimPrefix(path, "/v2/")
	parts := strings.SplitN(rest, "/", 2)
	registryName := parts[0]
	if len(parts) < 2 {
		http.Error(w, "invalid registry path", http.StatusBadRequest)
		return
	}

	upstream, ok := registryMap[registryName]
	if !ok {
		http.Error(w, "unknown registry", http.StatusNotFound)
		return
	}

	// Build upstream request — preserve method, body, and all headers
	target := upstream + "/v2/" + parts[1]
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	upstreamReq, err := http.NewRequest(r.Method, target, r.Body)
	if err != nil {
		http.Error(w, "request error", http.StatusInternalServerError)
		return
	}
	for k, vv := range r.Header {
		if k == "Host" {
			continue // let Go set the correct Host for the upstream
		}
		for _, v := range vv {
			upstreamReq.Header.Add(k, v)
		}
	}

	resp, err := http.DefaultClient.Do(upstreamReq)
	if err != nil {
		log.Printf("[registry] upstream error: %v", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Always include Docker v2 API version header in responses
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")

	// If 401, rewrite WWW-Authenticate to point to our /token endpoint
	if resp.StatusCode == http.StatusUnauthorized {
		wwAuth := resp.Header.Get("Www-Authenticate")
		if wwAuth != "" {
			newAuth := s.rewriteAuthHeader(wwAuth, r)
			log.Printf("[registry] rewrite auth: %s → %s", wwAuth, newAuth)
			resp.Header.Del("Www-Authenticate")
			resp.Header.Set("Www-Authenticate", newAuth)
		} else {
			log.Printf("[registry] 401 without Www-Authenticate header")
		}
	}

	log.Printf("[registry] %s %s → %s %d", r.Method, path, target, resp.StatusCode)

	// Copy response headers (excluding any we've already set)
	for k, vv := range resp.Header {
		if k == "Docker-Distribution-API-Version" {
			continue // we already set this above
		}
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) rewriteAuthHeader(authHeader string, r *http.Request) string {
	// Simple string replacement approach (same as hubproxy)
	// Replace all upstream auth URLs with our proxy's /token endpoint
	proxyHost := s.cfg.Server.PublicURL
	if proxyHost == "" {
		proxyHost = "http://" + r.Host
	}
	proxyHost = strings.TrimRight(proxyHost, "/")

	replacements := map[string]string{
		"https://auth.docker.io":   proxyHost,
		"https://ghcr.io":         proxyHost,
		"https://quay.io":         proxyHost,
		"https://mcr.microsoft.com": proxyHost,
	}
	for old, new := range replacements {
		authHeader = strings.ReplaceAll(authHeader, old, new)
	}
	return authHeader
}

func (s *Server) tokenAuthHandler(w http.ResponseWriter, r *http.Request) {
	// Proxy Docker v2 token requests to upstream auth servers
	// Docker sends: GET /token?service=ghcr.io&scope=repository:ksbbs/devbox:pull
	service := r.URL.Query().Get("service")

	authMap := map[string]string{
		"docker.io":            "https://auth.docker.io/token",
		"registry.docker.io":   "https://auth.docker.io/token",
		"ghcr.io":              "https://ghcr.io/token",
		"quay.io":              "https://quay.io/v2/auth",
		"mcr.microsoft.com":    "https://mcr.microsoft.com/v2/auth",
	}

	target, ok := authMap[service]
	if !ok {
		// Default: try Docker Hub auth
		target = "https://auth.docker.io/token"
	}

	// Forward the full query string
	targetURL := target + "?" + r.URL.RawQuery

	log.Printf("[token] service=%s → %s", service, targetURL)

	resp, err := http.Get(targetURL)
	if err != nil {
		log.Printf("[token] upstream error: %v", err)
		http.Error(w, "auth server error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	log.Printf("[token] upstream responded: %d", resp.StatusCode)

	// Also rewrite WWW-Authenticate in token response (some auth servers include it)
	for k, vv := range resp.Header {
		if k == "Www-Authenticate" || k == "WWW-Authenticate" {
			for _, v := range vv {
				w.Header().Add(k, s.rewriteAuthHeader(v, r))
			}
		} else {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
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
		// Skip rate limiting for dashboard API and frontend
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

// Start cleanup goroutines for rate limiter expired buckets
func (s *Server) rateLimitCleanup() {
	if s.limiter == nil {
		return
	}
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		s.limiter.Cleanup()
	}
}