package dashboard

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"devbox/internal/mirror"
	"devbox/internal/store"
)

type Dashboard struct {
	store     *store.Store
	authToken string
	publicURL string
}

func New(st *store.Store, authToken string, publicURL string) *Dashboard {
	return &Dashboard{store: st, authToken: authToken, publicURL: publicURL}
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

	usageMap := map[string]string{
		"npm":      fmt.Sprintf("npm config set registry %s/npm", baseURL),
		"pypi":     fmt.Sprintf("pip install <pkg> -i %s/pypi", baseURL),
		"docker":   fmt.Sprintf("配置 /etc/docker/daemon.json: registry-mirrors: [\"%s/docker\"]", baseURL),
		"golang":   fmt.Sprintf("go env -w GOPROXY=%s/golang,direct", baseURL),
		"cran":     fmt.Sprintf("options(repos=c(CRAN=\"%s/cran\"))", baseURL),
		"ghcr":     fmt.Sprintf("docker pull %s/ghcr/owner/image:tag", baseURL),
		"quay":     fmt.Sprintf("docker pull %s/quay/owner/image:tag", baseURL),
		"mcr":      fmt.Sprintf("docker pull %s/mcr/owner/image:tag", baseURL),
		"ghapi":    fmt.Sprintf("curl %s/ghapi/repos/owner/repo", baseURL),
		"gitproxy": fmt.Sprintf("git clone %s/gh/user/repo", baseURL),
	}

	mirrors := mirror.All()
	statuses := make([]map[string]interface{}, 0)
	for _, m := range mirrors {
		err := m.HealthCheck()
		status := "healthy"
		errMsg := ""
		if err != nil {
			status = "unhealthy"
			errMsg = err.Error()
		}
		d.store.RecordHealthCheck(m.Name(), status, errMsg)
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
		writeJSON(w, map[string]string{"status": "ok"})
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (d *Dashboard) PublicConfigHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"publicUrl": d.publicURL})
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

func (d *Dashboard) checkAuth(r *http.Request) bool {
	if d.authToken == "" {
		return true // no auth required
	}
	token := r.Header.Get("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	return token == d.authToken
}