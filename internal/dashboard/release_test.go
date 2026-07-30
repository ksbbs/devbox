package dashboard

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"devbox/internal/store"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestParseGitHubReleasesURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"https://github.com/moesnow/March7thAssistant/releases", true},
		{"https://github.com/moesnow/March7thAssistant/releases/", true},
		{"http://github.com/moesnow/March7thAssistant/releases", false},
		{"https://example.com/moesnow/March7thAssistant/releases", false},
		{"https://github.com/moesnow/March7thAssistant", false},
		{"https://user@github.com/moesnow/March7thAssistant/releases", false},
		{"https://github.com/moesnow/March7thAssistant/releases?x=1", false},
		{"https://github.com/moesnow%2Fother/repo/releases", false},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			owner, repo, err := parseGitHubReleasesURL(tt.url)
			if tt.valid && (err != nil || owner != "moesnow" || repo != "March7thAssistant") {
				t.Fatalf("expected valid URL, owner=%q repo=%q err=%v", owner, repo, err)
			}
			if !tt.valid && err == nil {
				t.Fatal("expected invalid URL")
			}
		})
	}
}

func TestReleaseSourceCreateTicketAndDownload(t *testing.T) {
	dashboard := newReleaseTestDashboard(t)
	releaseJSON := `{
		"tag_name":"v1.2.3",
		"html_url":"https://github.com/moesnow/March7thAssistant/releases/tag/v1.2.3",
		"published_at":"2026-07-26T05:18:51Z",
		"assets":[{"id":42,"name":"update.7z","size":7,"content_type":"application/x-7z-compressed","digest":"sha256:test"}]
	}`
	dashboard.releaseHTTP = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != githubAPIBase+"/repos/moesnow/March7thAssistant/releases/latest" {
			t.Fatalf("unexpected release URL: %s", req.URL)
		}
		if req.Header.Get("X-GitHub-Api-Version") != githubAPIVersion {
			t.Fatal("missing GitHub API version header")
		}
		return testResponse(http.StatusOK, "application/json", releaseJSON), nil
	})}

	body := `{"name":"M7A","releaseUrl":"https://github.com/moesnow/March7thAssistant/releases","assetName":"update.7z"}`
	createReq := authorizedRequest(http.MethodPost, "/api/release-sources", body)
	createRec := httptest.NewRecorder()
	dashboard.ReleaseSourcesHandler(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createRec.Code, createRec.Body.String())
	}
	if got := createRec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("unexpected content type %q", got)
	}
	var source releaseSourceView
	if err := json.NewDecoder(createRec.Body).Decode(&source); err != nil {
		t.Fatal(err)
	}
	if !source.Available || source.TagName != "v1.2.3" || source.AssetSize != 7 {
		t.Fatalf("unexpected source: %+v", source)
	}

	ticketReq := authorizedRequest(http.MethodPost, "/api/release-sources/1/download-ticket", "")
	ticketRec := httptest.NewRecorder()
	dashboard.ReleaseSourceHandler(ticketRec, ticketReq)
	if ticketRec.Code != http.StatusOK {
		t.Fatalf("ticket status=%d body=%s", ticketRec.Code, ticketRec.Body.String())
	}
	var ticketResult map[string]string
	if err := json.NewDecoder(ticketRec.Body).Decode(&ticketResult); err != nil {
		t.Fatal(err)
	}
	if ticketResult["ticket"] == "" {
		t.Fatal("empty download ticket")
	}

	dashboard.downloadHTTP = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != githubAPIBase+"/repos/moesnow/March7thAssistant/releases/assets/42" {
			t.Fatalf("unexpected asset URL: %s", req.URL)
		}
		if req.Header.Get("Accept") != "application/octet-stream" {
			t.Fatal("missing asset accept header")
		}
		resp := testResponse(http.StatusOK, "application/x-7z-compressed", "payload")
		resp.Header.Set("Content-Length", "7")
		return resp, nil
	})}
	downloadURL := "/api/release-download?ticket=" + ticketResult["ticket"]
	downloadRec := httptest.NewRecorder()
	dashboard.ReleaseDownloadHandler(downloadRec, httptest.NewRequest(http.MethodGet, downloadURL, nil))
	if downloadRec.Code != http.StatusOK || downloadRec.Body.String() != "payload" {
		t.Fatalf("download status=%d body=%q", downloadRec.Code, downloadRec.Body.String())
	}
	if disposition := downloadRec.Header().Get("Content-Disposition"); !strings.Contains(disposition, "update.7z") {
		t.Fatalf("unexpected disposition %q", disposition)
	}

	replayRec := httptest.NewRecorder()
	dashboard.ReleaseDownloadHandler(replayRec, httptest.NewRequest(http.MethodGet, downloadURL, nil))
	if replayRec.Code != http.StatusUnauthorized {
		t.Fatalf("replayed ticket status=%d", replayRec.Code)
	}
}

func TestReleaseHandlersRequireAuth(t *testing.T) {
	dashboard := newReleaseTestDashboard(t)
	rec := httptest.NewRecorder()
	dashboard.ReleaseSourcesHandler(rec, httptest.NewRequest(http.MethodGet, "/api/release-sources", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestReleaseDownloadRedirectPolicy(t *testing.T) {
	client := newReleaseDownloadClient()
	check := client.CheckRedirect
	via := []*http.Request{{URL: mustURL(t, "https://api.github.com/start")}}
	for _, target := range []string{
		"https://github.com/file",
		"https://objects.githubusercontent.com/file",
		"https://release-assets.githubusercontent.com/file",
	} {
		if err := check(&http.Request{URL: mustURL(t, target), Header: make(http.Header)}, via); err != nil {
			t.Fatalf("expected allowed redirect %s: %v", target, err)
		}
	}
	for _, target := range []string{
		"http://objects.githubusercontent.com/file",
		"https://githubusercontent.com.example.com/file",
		"https://example.com/file",
	} {
		if err := check(&http.Request{URL: mustURL(t, target), Header: make(http.Header)}, via); err == nil {
			t.Fatalf("expected blocked redirect %s", target)
		}
	}
}

func newReleaseTestDashboard(t *testing.T) *Dashboard {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "devbox.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return &Dashboard{
		store:           st,
		authToken:       "secret",
		releaseCache:    make(map[string]cachedRelease),
		downloadTickets: make(map[string]downloadTicket),
	}
}

func authorizedRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer secret")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func testResponse(status int, contentType, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{contentType}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
