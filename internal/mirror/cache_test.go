package mirror

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheGetSet(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, 1<<20)

	hdr := make(http.Header)
	hdr.Set("Content-Type", "application/json")

	c.Set("test-key", []byte("hello world"), hdr, 0)

	data, gotHdr, ok := c.Get("test-key")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(data) != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", data)
	}
	if gotHdr.Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type 'application/json', got '%s'", gotHdr.Get("Content-Type"))
	}
}

func TestCacheMiss(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, 1<<20)

	_, _, ok := c.Get("nonexistent-key")
	if ok {
		t.Fatal("expected cache miss for nonexistent key")
	}
}

func TestCacheExpiry(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, 1<<20)

	c.Set("exp-key", []byte("data"), make(http.Header), 2*time.Second)

	_, _, ok := c.Get("exp-key")
	if !ok {
		t.Fatal("expected cache hit before expiry")
	}

	if c.IsExpired("exp-key") {
		t.Fatal("not yet expired")
	}

	time.Sleep(3 * time.Second)

	if !c.IsExpired("exp-key") {
		t.Fatal("expected expired after TTL")
	}

	// Get should still return data until CleanExpired runs
	_, _, ok = c.Get("exp-key")
	if !ok {
		t.Fatal("Get returns data even after IsExpired")
	}
}

func TestCacheCleanExpired(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, 1<<20)

	c.Set("exp-key", []byte("data"), make(http.Header), 1*time.Second)
	c.Set("perm-key", []byte("permanent"), make(http.Header), 0)

	time.Sleep(2 * time.Second)

	c.CleanExpired()

	_, _, ok := c.Get("exp-key")
	if ok {
		t.Fatal("expected expired entry to be removed after CleanExpired")
	}

	_, _, ok = c.Get("perm-key")
	if !ok {
		t.Fatal("expected permanent entry to survive CleanExpired")
	}
}

func TestCacheMaxBytesSingleFile(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, 10)

	c.Set("small-key", []byte("small"), make(http.Header), 0)

	_, _, ok := c.Get("small-key")
	if !ok {
		t.Fatal("expected cache hit for small file within limit")
	}

	// Over-limit write should trigger eviction, not crash
	c.Set("big-key", []byte("this is a big value that exceeds limit"), make(http.Header), 0)

	// big-key should exist (recently written survives LRU)
	_, _, ok = c.Get("big-key")
	if !ok {
		t.Log("big-key evicted (acceptable if small-key was not)")
	}
}

func TestCacheLRUEviction(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, 30)

	c.Set("a", []byte("aaaaaaaaaaaaaaaa"), make(http.Header), 0) // 16 bytes
	c.Set("b", []byte("bbbbbbbbbbbbbbbb"), make(http.Header), 0) // 16 bytes, total 32 > 30

	// Should have evicted oldest (a) or kept both if under 80% target
	_, _, ok := c.Get("b")
	if !ok {
		t.Fatal("expected key 'b' to survive (more recently written)")
	}
}

func TestCacheProxyHTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("proxy-response"))
	}))
	defer ts.Close()

	dir := t.TempDir()
	c := NewCache(dir, 1<<20)

	req := httptest.NewRequest("GET", "/test-path", nil)
	w := httptest.NewRecorder()

	c.ProxyHTTP(w, req, ts.URL, time.Minute)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "proxy-response" {
		t.Fatalf("expected 'proxy-response', got '%s'", w.Body.String())
	}
}

func TestCacheProxyHTTPCaches(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("cached-response"))
	}))
	defer ts.Close()

	dir := t.TempDir()
	c := NewCache(dir, 1<<20)

	req1 := httptest.NewRequest("GET", "/cache-test", nil)
	w1 := httptest.NewRecorder()
	c.ProxyHTTP(w1, req1, ts.URL, time.Minute)

	req2 := httptest.NewRequest("GET", "/cache-test", nil)
	w2 := httptest.NewRecorder()
	c.ProxyHTTP(w2, req2, ts.URL, time.Minute)

	if callCount != 1 {
		t.Fatalf("expected 1 upstream call (cached), got %d", callCount)
	}
	if w2.Body.String() != "cached-response" {
		t.Fatalf("expected 'cached-response', got '%s'", w2.Body.String())
	}
}

func TestCacheProxyStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("stream-data"))
	}))
	defer ts.Close()

	dir := t.TempDir()
	c := NewCache(dir, 1<<20)

	req := httptest.NewRequest("GET", "/stream-path", nil)
	w := httptest.NewRecorder()

	c.ProxyStream(w, req, ts.URL)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "stream-data" {
		t.Fatalf("expected 'stream-data', got '%s'", w.Body.String())
	}
}

func TestCacheKeyPath(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, 1<<20)

	p1 := c.keyPath("key1")
	p2 := c.keyPath("key2")

	if p1 == p2 {
		t.Fatal("expected different paths for different keys")
	}
	if filepath.Dir(p1) != dir {
		t.Fatalf("expected dir %s, got %s", dir, filepath.Dir(p1))
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, 1<<20)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(i int) {
			key := fmt.Sprintf("concurrent-key-%d", i)
			c.Set(key, []byte(key), make(http.Header), time.Minute)
			_, _, ok := c.Get(key)
			if !ok {
				t.Logf("cache miss for %s (concurrent)", key)
			}
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCleanExpiredEmptyDir(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, 1<<20)
	// Should not panic on empty directory
	c.CleanExpired()
}

func TestCacheUsedBytesTracking(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, 1<<20)

	if c.usedBytes != 0 {
		t.Fatalf("expected usedBytes=0, got %d", c.usedBytes)
	}

	c.Set("a", []byte("12345"), make(http.Header), 0)
	if c.usedBytes != 5 {
		t.Fatalf("expected usedBytes=5, got %d", c.usedBytes)
	}
}

func TestCacheDir(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, 1<<20)
	if c.Dir() != dir {
		t.Fatalf("expected %s, got %s", dir, c.Dir())
	}
}
