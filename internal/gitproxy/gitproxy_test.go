package gitproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGitProxyGitHubClone(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("git-smart-http-response"))
	}))
	defer ts.Close()

	gp := New(ts.URL, "https://gitlab.com", ts.URL, time.Minute, nil)

	req := httptest.NewRequest("GET", "/gh/user/repo.git/info/refs?service=git-upload-pack", nil)
	w := httptest.NewRecorder()
	gp.Handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "git-smart-http-response" {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestGitProxyGitHubRaw(t *testing.T) {
	raw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/repo/branch/file.go" {
			t.Fatalf("expected path /user/repo/branch/file.go, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("raw-file-content"))
	}))
	defer raw.Close()

	gp := New("https://github.com", "https://gitlab.com", raw.URL, time.Minute, nil)

	req := httptest.NewRequest("GET", "/gh/user/repo/raw/branch/file.go", nil)
	w := httptest.NewRecorder()
	gp.Handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "raw-file-content" {
		t.Fatalf("expected 'raw-file-content', got '%s'", w.Body.String())
	}
}

func TestGitProxyGitHubBlobRedirect(t *testing.T) {
	raw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/repo/branch/file.go" {
			t.Fatalf("expected path /user/repo/branch/file.go, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("blob-content"))
	}))
	defer raw.Close()

	gp := New("https://github.com", "https://gitlab.com", raw.URL, time.Minute, nil)

	req := httptest.NewRequest("GET", "/gh/user/repo/blob/branch/file.go", nil)
	w := httptest.NewRecorder()
	gp.Handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "blob-content" {
		t.Fatalf("expected 'blob-content', got '%s'", w.Body.String())
	}
}

func TestGitProxyGitHubArchive(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("archive-data"))
	}))
	defer upstream.Close()

	gp := New(upstream.URL, "https://gitlab.com", "https://raw.githubusercontent.com", time.Minute, nil)

	req := httptest.NewRequest("GET", "/gh/user/repo/archive/main.tar.gz", nil)
	w := httptest.NewRecorder()
	gp.Handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "archive-data" {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestGitProxyGitLabClone(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("gitlab-smart-http"))
	}))
	defer ts.Close()

	gp := New("https://github.com", ts.URL, "https://raw.githubusercontent.com", time.Minute, nil)

	req := httptest.NewRequest("GET", "/gl/user/repo.git/info/refs", nil)
	w := httptest.NewRecorder()
	gp.Handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "gitlab-smart-http" {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestGitProxyHTMLBlock(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html>blocked</html>"))
	}))
	defer upstream.Close()

	gp := New(upstream.URL, "https://gitlab.com", "https://raw.githubusercontent.com", time.Minute, nil)

	req := httptest.NewRequest("GET", "/gh/user/repo/archive/main.tar.gz", nil)
	w := httptest.NewRecorder()
	gp.Handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for HTML response, got %d", w.Code)
	}
}

func TestGitProxyUnknownPath(t *testing.T) {
	gp := New("https://github.com", "https://gitlab.com", "https://raw.githubusercontent.com", time.Minute, nil)

	req := httptest.NewRequest("GET", "/unknown/path", nil)
	w := httptest.NewRecorder()
	gp.Handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown path, got %d", w.Code)
	}
}

func TestGitProxyRawUpstream(t *testing.T) {
	raw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("custom-raw"))
	}))
	defer raw.Close()

	gp := New("https://github.example.com", "https://gitlab.com", raw.URL, time.Minute, nil)

	req := httptest.NewRequest("GET", "/gh/user/repo/raw/branch/file.go", nil)
	w := httptest.NewRecorder()
	gp.Handler(w, req)

	if w.Body.String() != "custom-raw" {
		t.Fatalf("expected 'custom-raw', got '%s'", w.Body.String())
	}
}
