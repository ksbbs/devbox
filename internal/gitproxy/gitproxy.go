package gitproxy

import (
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type GitProxy struct {
	githubUpstream string
	gitlabUpstream string
	cacheTTL       time.Duration
	cacheDir       string
}

func New(githubUpstream, gitlabUpstream string, cacheTTL time.Duration, cacheDir string) *GitProxy {
	return &GitProxy{
		githubUpstream: githubUpstream,
		gitlabUpstream: gitlabUpstream,
		cacheTTL:       cacheTTL,
		cacheDir:       cacheDir,
	}
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
	// Auto-convert /blob/ URLs to /raw/ for file content
	if isBlobRequest(path) {
		path = strings.Replace(path, "/blob/", "/raw/", 1)
		gp.proxyRaw(w, r, path)
		return
	}
	// Determine proxy type based on path patterns
	if isArchiveRequest(path) {
		gp.proxyArchive(w, r, gp.githubUpstream, path)
	} else if isRawRequest(path) {
		gp.proxyRaw(w, r, path)
	} else if isSmartHTTP(r) {
		gp.proxySmartHTTP(w, r, gp.githubUpstream, path)
	} else {
		// Regular git clone via redirect
		gp.proxyRedirect(w, r, gp.githubUpstream, path)
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

func isSmartHTTP(r *http.Request) bool {
	q := r.URL.Query()
	return q.Get("service") == "git-upload-pack" ||
		q.Get("service") == "git-receive-pack" ||
		strings.HasSuffix(r.URL.Path, "info/refs") ||
		strings.HasSuffix(r.URL.Path, "git-upload-pack")
}

func (gp *GitProxy) proxyArchive(w http.ResponseWriter, r *http.Request, upstream, path string) {
	target := upstream + path
	resp, err := http.Get(target)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if isHTMLResponse(resp) {
		log.Printf("[gitproxy] blocking HTML response for %s", path)
		http.Error(w, "content blocked: HTML not allowed", http.StatusForbidden)
		return
	}
	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (gp *GitProxy) proxyRaw(w http.ResponseWriter, r *http.Request, path string) {
	// raw.githubusercontent.com
	rawURL := "https://raw.githubusercontent.com" + path
	resp, err := http.Get(rawURL)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if isHTMLResponse(resp) {
		log.Printf("[gitproxy] blocking HTML response for %s", path)
		http.Error(w, "content blocked: HTML not allowed", http.StatusForbidden)
		return
	}
	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (gp *GitProxy) proxySmartHTTP(w http.ResponseWriter, r *http.Request, upstream, path string) {
	target := upstream + path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	// Create new request with same method
	newReq, err := http.NewRequest(r.Method, target, r.Body)
	if err != nil {
		http.Error(w, "request error", http.StatusInternalServerError)
		return
	}
	copyRequestHeaders(newReq, r)

	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (gp *GitProxy) proxyRedirect(w http.ResponseWriter, r *http.Request, upstream, path string) {
	// For simple git clone, just redirect to upstream
	target := upstream + path + ".git"
	if !strings.HasSuffix(path, ".git") {
		target = upstream + path + ".git/info/refs?service=git-upload-pack"
	}
	http.Redirect(w, r, target, http.StatusFound)
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
		for _, v := range vv {
			newReq.Header.Add(k, v)
		}
	}
}

func isHTMLResponse(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "text/html")
}