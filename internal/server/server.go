package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"devbox/internal/config"
	"devbox/internal/dashboard"
	"devbox/internal/gitproxy"
	"devbox/internal/mirror"
	"devbox/internal/store"
)

type Server struct {
	cfg      *config.Config
	cache    *mirror.Cache
	gitProxy *gitproxy.GitProxy
	dash     *dashboard.Dashboard
	store    *store.Store
	frontDir string
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

	return &Server{
		cfg:      cfg,
		cache:    cache,
		gitProxy: gp,
		dash:     dash,
		store:    st,
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

	// Docker v2 registry API routes
	// Allows: docker pull <host>/ghcr/owner/image:tag
	// Docker sends /v2/ ping first, then /v2/{registry}/... requests
	mux.HandleFunc("/v2/", s.registryV2Handler)

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

	addr := fmt.Sprintf(":%d", s.cfg.Server.Port)
	log.Printf("DevBox starting on %s", addr)
	return http.ListenAndServe(addr, logMiddleware(mux, s.cfg.Logging.AccessLog))
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
	// e.g. /v2/ghcr/ksbbs/devbox/manifests/latest → ghcr.io/v2/ksbbs/devbox/manifests/latest
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

	// Strip registry name prefix: /v2/ghcr/ksbbs/devbox/... → /v2/ksbbs/devbox/...
	r.URL.Path = "/v2/" + parts[1]
	s.cache.ProxyStream(w, r, upstream)
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