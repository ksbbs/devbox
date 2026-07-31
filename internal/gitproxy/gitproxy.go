package gitproxy

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"devbox/internal/config"
	"devbox/internal/mirror"
)

// client is used for upstream git requests with connection/header timeout only.
// We use http.Transport.ResponseHeaderTimeout instead of Client.Timeout to avoid
// cutting off streaming archive/raw/git-smart-HTTP body reads.
var client = &http.Client{
	Transport: &http.Transport{
		ResponseHeaderTimeout: 60 * time.Second,
	},
}

type GitProxy struct {
	githubUpstream string
	gitlabUpstream string
	rawUpstream    string
	cacheTTL       time.Duration
	cache          *mirror.Cache
	mu             sync.RWMutex
}

func New(githubUpstream, gitlabUpstream, rawUpstream string, cacheTTL time.Duration, cache *mirror.Cache) *GitProxy {
	return &GitProxy{
		githubUpstream: githubUpstream,
		gitlabUpstream: gitlabUpstream,
		rawUpstream:    rawUpstream,
		cacheTTL:       cacheTTL,
		cache:          cache,
	}
}

func (gp *GitProxy) CacheTTL() string {
	gp.mu.RLock()
	defer gp.mu.RUnlock()
	d := gp.cacheTTL
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
	if secs%60 == 0 {
		return fmt.Sprintf("%dm", secs/60)
	}
	return fmt.Sprintf("%ds", secs)
}

func (gp *GitProxy) SetCacheTTL(ttl string) error {
	d, err := config.ParseDuration(ttl)
	if err != nil {
		return err
	}
	gp.mu.Lock()
	defer gp.mu.Unlock()
	gp.cacheTTL = d
	return nil
}

func (gp *GitProxy) Handler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if strings.HasPrefix(path, "/gh/") {
		gp.proxyGitHub(w, r, path[len("/gh"):])
		return
	}
	if strings.HasPrefix(path, "/gl/") {
		gp.proxyGitLab(w, r, path[len("/gl"):])
		return
	}
	http.Error(w, "unknown git proxy path", http.StatusBadRequest)
}

func (gp *GitProxy) proxyGitHub(w http.ResponseWriter, r *http.Request, path string) {
	if isBlobRequest(path) {
		// /user/repo/blob/branch/file → raw.githubusercontent.com/user/repo/branch/file
		path = strings.Replace(path, "/blob/", "/", 1)
		gp.proxyRaw(w, r, path)
		return
	}
	if isArchiveRequest(path) {
		gp.proxyArchive(w, r, gp.githubUpstream, path)
	} else if isRawRequest(path) {
		// /user/repo/raw/branch/file → raw.githubusercontent.com/user/repo/branch/file
		path = strings.Replace(path, "/raw/", "/", 1)
		gp.proxyRaw(w, r, path)
	} else {
		gp.proxySmartHTTP(w, r, gp.githubUpstream, path)
	}
}

func (gp *GitProxy) proxyGitLab(w http.ResponseWriter, r *http.Request, path string) {
	gp.proxySmartHTTP(w, r, gp.gitlabUpstream, path)
}

func isArchiveRequest(path string) bool {
	return strings.Contains(path, "/archive/")
}

func isBlobRequest(path string) bool {
	return strings.Contains(path, "/blob/")
}

func isRawRequest(path string) bool {
	return strings.Contains(path, "/raw/")
}

func (gp *GitProxy) proxyArchive(w http.ResponseWriter, r *http.Request, upstream, path string) {
	gp.mu.RLock()
	cacheTTL := gp.cacheTTL
	gp.mu.RUnlock()
	if gp.cache != nil && cacheTTL > 0 {
		target := upstream + path
		if resp, err := client.Head(target); err == nil {
			if resp.StatusCode == http.StatusNotFound {
				resp.Body.Close()
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			if isHTMLResponse(resp) {
				resp.Body.Close()
				slog.Warn("gitproxy blocking HTML response (cache)", "path", path)
				http.Error(w, "content blocked: HTML not allowed", http.StatusForbidden)
				return
			}
			resp.Body.Close()
		}
		orig := r.URL.Path
		r.URL.Path = path
		gp.cache.ProxyHTTP(w, r, upstream, gp.cacheTTL)
		r.URL.Path = orig
		return
	}
	target := upstream + path
	resp, err := client.Get(target)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if isHTMLResponse(resp) {
		slog.Warn("gitproxy blocking HTML response", "path", path)
		http.Error(w, "content blocked: HTML not allowed", http.StatusForbidden)
		return
	}
	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (gp *GitProxy) proxyRaw(w http.ResponseWriter, r *http.Request, path string) {
	gp.mu.RLock()
	cacheTTL := gp.cacheTTL
	gp.mu.RUnlock()
	if gp.cache != nil && cacheTTL > 0 {
		target := gp.rawUpstream + path
		if resp, err := client.Head(target); err == nil {
			if resp.StatusCode == http.StatusNotFound {
				resp.Body.Close()
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			if isHTMLResponse(resp) {
				resp.Body.Close()
				slog.Warn("gitproxy blocking HTML response (cache)", "path", path)
				http.Error(w, "content blocked: HTML not allowed", http.StatusForbidden)
				return
			}
			resp.Body.Close()
		}
		orig := r.URL.Path
		r.URL.Path = path
		gp.cache.ProxyHTTP(w, r, gp.rawUpstream, cacheTTL)
		r.URL.Path = orig
		return
	}
	rawURL := gp.rawUpstream + path
	resp, err := client.Get(rawURL)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if isHTMLResponse(resp) {
		slog.Warn("gitproxy blocking HTML response", "path", path)
		http.Error(w, "content blocked: HTML not allowed", http.StatusForbidden)
		return
	}
	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (gp *GitProxy) proxySmartHTTP(w http.ResponseWriter, r *http.Request, upstream, path string) {
	gp.mu.RLock()
	cacheTTL := gp.cacheTTL
	gp.mu.RUnlock()
	if gp.cache != nil && cacheTTL > 0 && r.Method == "GET" {
		orig := r.URL.Path
		r.URL.Path = path
		gp.cache.ProxyHTTP(w, r, upstream, cacheTTL)
		r.URL.Path = orig
		return
	}
	target := upstream + path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	newReq, err := http.NewRequest(r.Method, target, r.Body)
	if err != nil {
		http.Error(w, "request error", http.StatusInternalServerError)
		return
	}
	newReq.ContentLength = r.ContentLength
	copyRequestHeaders(newReq, r)

	resp, err := client.Do(newReq)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
}

func copyRequestHeaders(newReq *http.Request, orig *http.Request) {
	for k, vv := range orig.Header {
		if isHopByHopHeader(k) {
			continue
		}
		for _, v := range vv {
			newReq.Header.Add(k, v)
		}
	}
}

func isHopByHopHeader(k string) bool {
	switch http.CanonicalHeaderKey(k) {
	case "Host", "Connection", "Keep-Alive", "Proxy-Connection",
		"Proxy-Authorization", "Transfer-Encoding", "Upgrade", "TE", "Trailer":
		return true
	}
	return false
}

func isHTMLResponse(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "text/html")
}
