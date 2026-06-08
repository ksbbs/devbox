package dashboard

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"devbox/internal/mirror"
	"devbox/internal/store"
)

type Dashboard struct {
	store      *store.Store
	authToken  string
	publicURL  string
	rlConfig   RateLimitConfigAccessor
	saveConfig func() error

	healthMu    sync.RWMutex
	healthCache map[string]cachedHealth
	healthReady bool
}

func (d *Dashboard) SetSaveConfig(fn func() error) {
	d.saveConfig = fn
}

type RateLimitConfigAccessor interface {
	GetRateLimitConfig() RateLimitConfigView
	SetRateLimitEnabled(enabled bool)
	SetRateLimitRate(rate int)
	SetRateLimitInterval(interval string)
	SetRateLimitWhitelist(list []string)
	SetRateLimitBlacklist(list []string)
}

type RateLimitConfigView struct {
	Enabled   bool     `json:"enabled"`
	Rate      int      `json:"rate"`
	Interval  string   `json:"interval"`
	Whitelist []string `json:"whitelist"`
	Blacklist []string `json:"blacklist"`
}

type cachedHealth struct {
	status string
	err    string
}

func New(st *store.Store, authToken string, publicURL string) *Dashboard {
	d := &Dashboard{
		store:       st,
		authToken:   authToken,
		publicURL:   publicURL,
		healthCache: make(map[string]cachedHealth),
	}
	go d.backgroundHealthCheck()
	return d
}

func (d *Dashboard) SetRateLimitConfigAccessor(rl RateLimitConfigAccessor) {
	d.rlConfig = rl
}

func (d *Dashboard) backgroundHealthCheck() {
	d.runHealthChecks()
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		d.runHealthChecks()
	}
}

func (d *Dashboard) runHealthChecks() {
	mirrors := mirror.All()
	type result struct {
		name   string
		status string
		err    string
	}
	ch := make(chan result, len(mirrors))
	for _, m := range mirrors {
		go func(m mirror.Mirror) {
			r := result{name: m.Name(), status: "healthy"}
			if err := m.HealthCheck(); err != nil {
				r.status = "unhealthy"
				r.err = err.Error()
			}
			d.store.RecordHealthCheck(m.Name(), r.status, r.err)
			ch <- r
		}(m)
	}
	cache := make(map[string]cachedHealth, len(mirrors))
	for range mirrors {
		r := <-ch
		cache[r.name] = cachedHealth{status: r.status, err: r.err}
	}
	d.healthMu.Lock()
	d.healthCache = cache
	d.healthReady = true
	d.healthMu.Unlock()
	log.Printf("[health] background check complete: %d mirrors", len(cache))
}

func (d *Dashboard) StatusHandler(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	baseURL := d.publicURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	// docker pull 命令不能带协议前缀，其他命令和 registry-mirrors 需要
	hostOnly := baseURL
	hostOnly = strings.TrimPrefix(hostOnly, "https://")
	hostOnly = strings.TrimPrefix(hostOnly, "http://")

	usageMap := map[string]string{
		"npm":      fmt.Sprintf("npm config set registry %s/npm", baseURL),
		"pypi":     fmt.Sprintf("pip install <pkg> -i %s/pypi", baseURL),
		"docker":   fmt.Sprintf("配置 /etc/docker/daemon.json: registry-mirrors: [\"%s/docker\"]", baseURL),
		"golang":   fmt.Sprintf("go env -w GOPROXY=%s/golang,direct", baseURL),
		"cran":     fmt.Sprintf("options(repos=c(CRAN=\"%s/cran\"))", baseURL),
		"ghcr":     fmt.Sprintf("docker pull %s/ghcr/owner/image:tag", hostOnly),
		"quay":     fmt.Sprintf("docker pull %s/quay/owner/image:tag", hostOnly),
		"mcr":      fmt.Sprintf("docker pull %s/mcr/owner/image:tag", hostOnly),
		"ghapi":    fmt.Sprintf("curl %s/ghapi/repos/owner/repo", baseURL),
		"gitproxy": fmt.Sprintf("git clone %s/gh/user/repo", baseURL),
		"hf":       fmt.Sprintf("huggingface-cli download --endpoint %s/hf model/name", baseURL),
		"conda":    fmt.Sprintf("conda config --add channels %s/conda", baseURL),
		"rubygems": fmt.Sprintf("gem source -a %s/rubygems", baseURL),
		"cargo":    fmt.Sprintf("export CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse; CARGO_REGISTRIES_CRATES_IO_INDEX=%s/cargo", baseURL),
		"nuget":    fmt.Sprintf("dotnet nuget add source %s/nuget/index.json", baseURL),
		"apt":      fmt.Sprintf("echo 'deb %s/apt stable main' > /etc/apt/sources.list", baseURL),
		"alpine":   fmt.Sprintf("sed -i 's|dl-cdn.alpinelinux.org|%s/alpine|g' /etc/apk/repositories", baseURL),
		"homebrew": fmt.Sprintf("export HOMEBREW_BOTTLE_DOMAIN=%s/homebrew", baseURL),
	}

	mirrors := mirror.All()
	statuses := make([]map[string]interface{}, 0)

	d.healthMu.RLock()
	hc := d.healthCache
	d.healthMu.RUnlock()

	for _, m := range mirrors {
		status := "healthy"
		errMsg := ""
		if cached, ok := hc[m.Name()]; ok {
			status = cached.status
			errMsg = cached.err
		}
		entry := map[string]interface{}{
			"name":     m.Name(),
			"pattern":  m.Pattern(),
			"upstream": m.Upstream(),
			"enabled":  m.IsEnabled(),
			"status":   status,
			"error":    errMsg,
		}
		if usage, ok := usageMap[m.Name()]; ok {
			entry["usage"] = usage
		}
		statuses = append(statuses, entry)
	}

	// Add git proxy as a virtual mirror entry
	gitproxyEntry := map[string]interface{}{
		"name":     "gitproxy",
		"pattern":  "/gh/, /gl/",
		"upstream": "github.com, gitlab.com",
		"enabled":  true,
		"status":   "healthy",
		"error":    "",
		"usage":    usageMap["gitproxy"],
	}
	statuses = append(statuses, gitproxyEntry)

	writeJSON(w, statuses)
}

func (d *Dashboard) TrafficHandler(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	granularity := r.URL.Query().Get("granularity")

	from := time.Now().Add(-7 * 24 * time.Hour)
	to := time.Now()

	if fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err == nil {
			from = t
		}
	}
	if toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err == nil {
			to = t
		}
	}

	if granularity == "hourly" {
		hourly, err := d.store.GetTrafficHourly(from, to)
		if err != nil {
			http.Error(w, "query error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, hourly)
		return
	}

	summaries, err := d.store.GetTrafficSummary(from, to)
	if err != nil {
		http.Error(w, "query error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, summaries)
}

func (d *Dashboard) LogHandler(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	logs, err := d.store.GetRecentTraffic(limit)
	if err != nil {
		http.Error(w, "query error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, logs)
}

func (d *Dashboard) MirrorConfigHandler(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		mirrors := mirror.All()
		configs := make([]map[string]interface{}, 0)
		for _, m := range mirrors {
			configs = append(configs, map[string]interface{}{
				"name":     m.Name(),
				"enabled":  m.IsEnabled(),
				"upstream": m.Upstream(),
				"cacheTTL": m.CacheTTL(),
			})
		}
		writeJSON(w, configs)
		return
	}

	if r.Method == http.MethodPut {
		var req struct {
			Name     string `json:"name"`
			Enabled  bool   `json:"enabled"`
			Upstream string `json:"upstream"`
			CacheTTL string `json:"cacheTTL"`
		}
		if !readJSON(r, &req) {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		m, ok := mirror.Get(req.Name)
		if !ok {
			http.Error(w, "mirror not found", http.StatusNotFound)
			return
		}
		m.SetEnabled(req.Enabled)
		if req.Upstream != "" {
			m.SetUpstream(req.Upstream)
		}
		if req.CacheTTL != "" {
			if err := m.SetCacheTTL(req.CacheTTL); err != nil {
				http.Error(w, "invalid cacheTTL: "+err.Error(), http.StatusBadRequest)
				return
			}
		}
		if d.saveConfig != nil {
			d.saveConfig()
		}
		writeJSON(w, map[string]string{"status": "ok"})
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (d *Dashboard) PublicConfigHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"publicUrl": d.publicURL})
}

func (d *Dashboard) AuthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if d.authToken == "" {
		writeJSON(w, map[string]bool{"authRequired": false})
		return
	}
	if d.checkAuth(r) {
		writeJSON(w, map[string]bool{"authRequired": true, "authenticated": true})
		return
	}
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func (d *Dashboard) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if !readJSON(r, &req) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Token != d.authToken {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]string{"status": "ok", "token": req.Token})
}

func (d *Dashboard) RateLimitConfigHandler(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if d.rlConfig == nil {
		http.Error(w, "rate limit not available", http.StatusNotFound)
		return
	}

	if r.Method == http.MethodGet {
		writeJSON(w, d.rlConfig.GetRateLimitConfig())
		return
	}

	if r.Method == http.MethodPut {
		var req RateLimitConfigView
		if !readJSON(r, &req) {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		d.rlConfig.SetRateLimitEnabled(req.Enabled)
		d.rlConfig.SetRateLimitRate(req.Rate)
		d.rlConfig.SetRateLimitInterval(req.Interval)
		d.rlConfig.SetRateLimitWhitelist(req.Whitelist)
		d.rlConfig.SetRateLimitBlacklist(req.Blacklist)
		if d.saveConfig != nil {
			d.saveConfig()
		}
		writeJSON(w, d.rlConfig.GetRateLimitConfig())
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (d *Dashboard) checkAuth(r *http.Request) bool {
	if d.authToken == "" {
		return true // no auth required
	}
	token := r.Header.Get("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	return token == d.authToken
}
