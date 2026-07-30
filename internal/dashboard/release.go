package dashboard

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"devbox/internal/store"
)

const (
	githubAPIBase      = "https://api.github.com"
	githubAPIVersion   = "2026-03-10"
	releaseCacheTTL    = 5 * time.Minute
	downloadTicketTTL  = 2 * time.Minute
	maxDownloadTickets = 4096
)

var githubNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	HTMLURL     string        `json:"html_url"`
	PublishedAt string        `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
	Digest             string `json:"digest"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type cachedRelease struct {
	release   githubRelease
	expiresAt time.Time
}

type downloadTicket struct {
	sourceID  int64
	owner     string
	repo      string
	asset     githubAsset
	tagName   string
	expiresAt time.Time
}

type releaseSourceView struct {
	store.ReleaseSource
	RepositoryURL string `json:"repositoryUrl"`
	TagName       string `json:"tagName,omitempty"`
	ReleaseURL    string `json:"releaseUrl,omitempty"`
	PublishedAt   string `json:"publishedAt,omitempty"`
	AssetSize     int64  `json:"assetSize,omitempty"`
	Digest        string `json:"digest,omitempty"`
	Available     bool   `json:"available"`
	Error         string `json:"error,omitempty"`
}

func (d *Dashboard) ReleaseSourcesHandler(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		d.listReleaseSources(w, r, r.URL.Query().Get("refresh") == "1")
	case http.MethodPost:
		d.createReleaseSource(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (d *Dashboard) ReleaseSourceHandler(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/release-sources/"), "/")
	parts := strings.Split(rest, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid release source id", http.StatusBadRequest)
		return
	}

	if len(parts) == 1 && r.Method == http.MethodDelete {
		deleted, err := d.store.DeleteReleaseSource(id)
		if err != nil {
			http.Error(w, "delete release source failed", http.StatusInternalServerError)
			return
		}
		if !deleted {
			http.Error(w, "release source not found", http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 2 && parts[1] == "download-ticket" && r.Method == http.MethodPost {
		d.createDownloadTicket(w, r, id)
		return
	}
	http.Error(w, "not found", http.StatusNotFound)
}

func (d *Dashboard) listReleaseSources(w http.ResponseWriter, r *http.Request, refresh bool) {
	sources, err := d.store.ListReleaseSources()
	if err != nil {
		http.Error(w, "list release sources failed", http.StatusInternalServerError)
		return
	}

	views := make([]releaseSourceView, len(sources))
	type sourceGroup struct {
		source  store.ReleaseSource
		indexes []int
	}
	groups := make(map[string]*sourceGroup)
	for i, source := range sources {
		key := strings.ToLower(source.Owner + "/" + source.Repo)
		if group, ok := groups[key]; ok {
			group.indexes = append(group.indexes, i)
		} else {
			groups[key] = &sourceGroup{source: source, indexes: []int{i}}
		}
	}

	semaphore := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, group := range groups {
		wg.Add(1)
		go func(group *sourceGroup) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			release, err := d.latestRelease(r, group.source.Owner, group.source.Repo, refresh)
			for _, index := range group.indexes {
				errorMessage := ""
				if err != nil {
					errorMessage = err.Error()
				}
				views[index] = d.releaseSourceViewFromRelease(sources[index], release, errorMessage)
			}
		}(group)
	}
	wg.Wait()
	writeJSON(w, views)
}

func (d *Dashboard) createReleaseSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		ReleaseURL string `json:"releaseUrl"`
		AssetName  string `json:"assetName"`
	}
	if !readJSON(r, &req) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	assetName := strings.TrimSpace(req.AssetName)
	owner, repo, err := parseGitHubReleasesURL(req.ReleaseURL)
	if err != nil || name == "" || len(name) > 80 || !validAssetName(assetName) {
		http.Error(w, "invalid name, GitHub Releases URL, or asset name", http.StatusBadRequest)
		return
	}

	release, err := d.latestRelease(r, owner, repo, true)
	if err != nil {
		http.Error(w, "get latest release: "+err.Error(), http.StatusBadGateway)
		return
	}
	if _, ok := findAsset(release, assetName); !ok {
		http.Error(w, "asset not found in latest release", http.StatusUnprocessableEntity)
		return
	}

	source, err := d.store.CreateReleaseSource(name, owner, repo, assetName)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			http.Error(w, "release source already exists", http.StatusConflict)
			return
		}
		http.Error(w, "save release source failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, d.releaseSourceViewFromRelease(source, release, ""))
}

func (d *Dashboard) releaseSourceViewFromRelease(source store.ReleaseSource, release githubRelease, releaseErr string) releaseSourceView {
	view := releaseSourceView{
		ReleaseSource: source,
		RepositoryURL: fmt.Sprintf("https://github.com/%s/%s", source.Owner, source.Repo),
		TagName:       release.TagName,
		ReleaseURL:    release.HTMLURL,
		PublishedAt:   release.PublishedAt,
		Error:         releaseErr,
	}
	if releaseErr != "" {
		return view
	}
	asset, ok := findAsset(release, source.AssetName)
	if !ok {
		view.Error = "asset not found in latest release"
		return view
	}
	view.Available = true
	view.AssetSize = asset.Size
	view.Digest = asset.Digest
	return view
}

func (d *Dashboard) createDownloadTicket(w http.ResponseWriter, r *http.Request, sourceID int64) {
	source, err := d.store.GetReleaseSource(sourceID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "release source not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "get release source failed", http.StatusInternalServerError)
		return
	}

	release, err := d.latestRelease(r, source.Owner, source.Repo, true)
	if err != nil {
		http.Error(w, "get latest release: "+err.Error(), http.StatusBadGateway)
		return
	}
	asset, ok := findAsset(release, source.AssetName)
	if !ok {
		http.Error(w, "asset not found in latest release", http.StatusUnprocessableEntity)
		return
	}
	ticket, err := randomTicket()
	if err != nil {
		http.Error(w, "create download ticket failed", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	d.ticketMu.Lock()
	for key, entry := range d.downloadTickets {
		if now.After(entry.expiresAt) {
			delete(d.downloadTickets, key)
		}
	}
	if len(d.downloadTickets) >= maxDownloadTickets {
		d.ticketMu.Unlock()
		http.Error(w, "too many pending download tickets", http.StatusTooManyRequests)
		return
	}
	d.downloadTickets[ticket] = downloadTicket{
		sourceID:  source.ID,
		owner:     source.Owner,
		repo:      source.Repo,
		asset:     asset,
		tagName:   release.TagName,
		expiresAt: now.Add(downloadTicketTTL),
	}
	d.ticketMu.Unlock()
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, map[string]string{
		"ticket":   ticket,
		"fileName": asset.Name,
		"tagName":  release.TagName,
	})
}

func (d *Dashboard) ReleaseDownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ticketValue := r.URL.Query().Get("ticket")
	d.ticketMu.Lock()
	ticket, ok := d.downloadTickets[ticketValue]
	if ok {
		delete(d.downloadTickets, ticketValue)
	}
	d.ticketMu.Unlock()
	if !ok || time.Now().After(ticket.expiresAt) {
		http.Error(w, "invalid or expired download ticket", http.StatusUnauthorized)
		return
	}

	target := fmt.Sprintf("%s/repos/%s/%s/releases/assets/%d", githubAPIBase,
		url.PathEscape(ticket.owner), url.PathEscape(ticket.repo), ticket.asset.ID)
	upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		http.Error(w, "create upstream request failed", http.StatusInternalServerError)
		return
	}
	upstreamReq.Header.Set("Accept", "application/octet-stream")
	setGitHubHeaders(upstreamReq)
	resp, err := d.downloadHTTP.Do(upstreamReq)
	if err != nil {
		slog.Error("release download upstream failed", "source_id", ticket.sourceID, "error", err)
		http.Error(w, "release download upstream failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Warn("release download upstream status", "source_id", ticket.sourceID, "status", resp.StatusCode)
		http.Error(w, "release download upstream returned "+resp.Status, http.StatusBadGateway)
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(strings.ToLower(contentType), "application/json") {
		http.Error(w, "GitHub returned asset metadata instead of file content", http.StatusBadGateway)
		return
	}
	if contentType == "" {
		contentType = ticket.asset.ContentType
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		w.Header().Set("Content-Length", contentLength)
	} else if ticket.asset.Size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(ticket.asset.Size, 10))
	}
	if etag := resp.Header.Get("ETag"); etag != "" {
		w.Header().Set("ETag", etag)
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": ticket.asset.Name}))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, resp.Body); err != nil && !errors.Is(err, r.Context().Err()) {
		slog.Warn("release download interrupted", "source_id", ticket.sourceID, "tag", ticket.tagName, "error", err)
	}
}

func (d *Dashboard) latestRelease(r *http.Request, owner, repo string, refresh bool) (githubRelease, error) {
	key := strings.ToLower(owner + "/" + repo)
	if !refresh {
		d.releaseCacheMu.Lock()
		entry, ok := d.releaseCache[key]
		d.releaseCacheMu.Unlock()
		if ok && time.Now().Before(entry.expiresAt) {
			return entry.release, nil
		}
	}

	target := fmt.Sprintf("%s/repos/%s/%s/releases/latest", githubAPIBase, url.PathEscape(owner), url.PathEscape(repo))
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	setGitHubHeaders(req)
	resp, err := d.releaseHTTP.Do(req)
	if err != nil {
		return githubRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return githubRelease{}, fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("decode GitHub release: %w", err)
	}
	d.releaseCacheMu.Lock()
	d.releaseCache[key] = cachedRelease{release: release, expiresAt: time.Now().Add(releaseCacheTTL)}
	d.releaseCacheMu.Unlock()
	return release, nil
}

func parseGitHubReleasesURL(rawURL string) (string, string, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Hostname(), "github.com") || u.Port() != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", "", errors.New("invalid GitHub Releases URL")
	}
	parts := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	if len(parts) != 3 || !strings.EqualFold(parts[2], "releases") {
		return "", "", errors.New("URL must end with /owner/repo/releases")
	}
	owner, err := url.PathUnescape(parts[0])
	if err != nil {
		return "", "", err
	}
	repo, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", "", err
	}
	if !githubNamePattern.MatchString(owner) || !githubNamePattern.MatchString(repo) || len(owner) > 100 || len(repo) > 100 {
		return "", "", errors.New("invalid GitHub owner or repository")
	}
	return owner, repo, nil
}

func validAssetName(name string) bool {
	if name == "" || len(name) > 255 || name != path.Base(name) || strings.ContainsAny(name, `/\\`) || name == "." || name == ".." {
		return false
	}
	for _, char := range name {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func findAsset(release githubRelease, name string) (githubAsset, bool) {
	for _, asset := range release.Assets {
		if asset.Name == name && asset.ID > 0 {
			return asset, true
		}
	}
	return githubAsset{}, false
}

func randomTicket() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func setGitHubHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "DevBox-Release-Downloader")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
}

func newReleaseDownloadClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "https" || req.URL.User != nil || !allowedGitHubDownloadHost(req.URL.Hostname()) {
				return errors.New("unsafe release download redirect")
			}
			req.Header.Del("Authorization")
			return nil
		},
	}
}

func allowedGitHubDownloadHost(host string) bool {
	host = strings.ToLower(host)
	return host == "api.github.com" || host == "github.com" || strings.HasSuffix(host, ".githubusercontent.com")
}
