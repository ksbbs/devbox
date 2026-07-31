package mirror

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// proxyClient is used for upstream requests with connection/header timeout only.
// We use http.Transport.ResponseHeaderTimeout instead of Client.Timeout to avoid
// cutting off large or slow mirror downloads during body reads.
var proxyClient = &http.Client{
	Transport: &http.Transport{
		ResponseHeaderTimeout: 60 * time.Second,
	},
}

type Cache struct {
	dir       string
	maxBytes  int64
	usedBytes int64
	mu        sync.Mutex
	hits      atomic.Int64
	misses    atomic.Int64
}

func NewCache(dir string, maxBytes int64) *Cache {
	return &Cache{dir: dir, maxBytes: maxBytes}
}

func (c *Cache) Dir() string { return c.dir }

func (c *Cache) Get(key string) ([]byte, http.Header, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	path := c.keyPath(key)
	info, err := os.Stat(path)
	if err != nil {
		c.misses.Add(1)
		return nil, nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("cache read error", "path", path, "error", err)
		c.misses.Add(1)
		return nil, nil, false
	}
	hdrPath := path + ".hdr"
	hdrData, _ := os.ReadFile(hdrPath)
	hdr := make(http.Header)
	if hdrData != nil {
		for _, line := range strings.Split(string(hdrData), "\n") {
			if line == "" {
				continue
			}
			if idx := strings.IndexByte(line, ':'); idx > 0 {
				hdr.Set(line[:idx], strings.TrimSpace(line[idx+1:]))
			}
		}
	}
	if c.maxBytes > 0 && info.Size() > c.maxBytes {
		os.Remove(path)
		os.Remove(hdrPath)
		os.Remove(path + ".exp")
		c.usedBytes -= info.Size()
		c.misses.Add(1)
		return nil, nil, false
	}
	c.hits.Add(1)
	return data, hdr, true
}

func (c *Cache) Set(key string, data []byte, hdr http.Header, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	path := c.keyPath(key)

	oldSize := int64(0)
	if info, err := os.Stat(path); err == nil {
		oldSize = info.Size()
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		slog.Warn("cache mkdir error", "dir", filepath.Dir(path), "error", err)
		return
	}

	f, err := os.Create(path)
	if err != nil {
		slog.Warn("cache write error", "path", path, "error", err)
		return
	}
	if _, err := f.Write(data); err != nil {
		slog.Warn("cache write error", "path", path, "error", err)
		f.Close()
		return
	}
	f.Close()

	hdrPath := path + ".hdr"
	hf, err := os.Create(hdrPath)
	if err != nil {
		slog.Warn("cache header write error", "path", hdrPath, "error", err)
		os.Remove(path)
		return
	}
	for k, vv := range hdr {
		for _, v := range vv {
			fmt.Fprintf(hf, "%s:%s\n", k, v)
		}
	}
	hf.Close()

	expPath := path + ".exp"
	if ttl > 0 {
		ef, err := os.Create(expPath)
		if err != nil {
			slog.Warn("cache expiry write error", "path", expPath, "error", err)
			os.Remove(path)
			os.Remove(hdrPath)
			return
		}
		fmt.Fprintf(ef, "%d", time.Now().Add(ttl).Unix())
		ef.Close()
	} else {
		// ttl<=0 means never expire: drop any stale expiry marker from an
		// earlier TTL so shorter/zero TTLs take effect immediately.
		os.Remove(expPath)
	}

	c.usedBytes += int64(len(data)) - oldSize

	if c.maxBytes > 0 && c.usedBytes > c.maxBytes {
		c.evictLRU()
	}
}

func (c *Cache) evictLRU() {
	target := int64(float64(c.maxBytes) * 0.8)
	if target <= 0 {
		target = c.maxBytes / 2
	}

	type entry struct {
		path  string
		size  int64
		mtime time.Time
	}

	var entries []entry

	_ = filepath.WalkDir(c.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if err != nil {
				return nil
			}
			if path == c.dir {
				return nil
			}
			return filepath.SkipDir
		}
		name := d.Name()
		if filepath.Ext(name) == ".exp" || filepath.Ext(name) == ".hdr" || strings.Contains(name, ".tmp") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		entries = append(entries, entry{
			path:  path,
			size:  info.Size(),
			mtime: info.ModTime(),
		})
		return nil
	})

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].mtime.Before(entries[j].mtime)
	})

	for _, e := range entries {
		if c.usedBytes <= target {
			break
		}
		os.Remove(e.path)
		os.Remove(e.path + ".hdr")
		os.Remove(e.path + ".exp")
		c.usedBytes -= e.size
	}
}

func (c *Cache) IsExpired(key string) bool {
	path := c.keyPath(key) + ".exp"
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	expiry, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Unix() > expiry
}

func (c *Cache) ProxyHTTP(w http.ResponseWriter, r *http.Request, upstream string, ttl time.Duration) {
	key := r.Method + "|" + upstream + r.URL.Path + "?" + r.URL.RawQuery

	// Authenticated requests must not read or write the shared cache:
	// the key has no identity component, so cached responses fetched with
	// one client's credentials would leak to everyone else.
	authenticated := r.Header.Get("Authorization") != ""

	if r.Method == http.MethodGet && !authenticated && !c.IsExpired(key) {
		data, hdr, ok := c.Get(key)
		if ok {
			for k, vv := range hdr {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return
		}
	}

	target := upstream + r.URL.Path
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

	resp, err := proxyClient.Do(newReq)
	if err != nil {
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respHdr := resp.Header.Clone()

	// Stream to client while spooling cacheable (200 only, so partial
	// 206 responses never poison the cache) responses to a temp file.
	cacheable := resp.StatusCode == http.StatusOK && ttl >= 0 && !authenticated && r.Method == http.MethodGet
	path := c.keyPath(key)
	var tmp *os.File
	if cacheable {
		// Unique temp name so concurrent misses on the same key can never
		// interleave writes into one file; only the rename publishes data.
		tmp, _ = os.CreateTemp(c.dir, filepath.Base(path)+".tmp-")
		if tmp != nil {
			defer os.Remove(tmp.Name())
		}
	}

	for k, vv := range respHdr {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	var src io.Reader = resp.Body
	if tmp != nil {
		src = io.TeeReader(resp.Body, tmp)
	}
	written, copyErr := io.Copy(w, src)
	if tmp == nil {
		return
	}
	if err := tmp.Close(); err != nil || copyErr != nil || written <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// Stat before the atomic rename: after it, path already points at the new
	// file whose size equals written, which would cancel out the accounting
	// below and keep usedBytes from ever growing past maxBytes.
	oldSize := int64(0)
	if info, err := os.Stat(path); err == nil {
		oldSize = info.Size()
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		slog.Warn("cache rename error", "path", path, "error", err)
		return
	}
	c.writeCacheMeta(path, respHdr, ttl)
	c.usedBytes += written - oldSize
	if c.maxBytes > 0 && c.usedBytes > c.maxBytes {
		c.evictLRU()
	}
}

func (c *Cache) ProxyStream(w http.ResponseWriter, r *http.Request, upstream string) {
	target := upstream + r.URL.Path
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

	resp, err := proxyClient.Do(newReq)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// copyRequestHeaders copies client headers onto the upstream request,
// skipping hop-by-hop headers (RFC 7230 §6.1) like Host, Connection and
// Proxy-Authorization.
func copyRequestHeaders(dst *http.Request, src *http.Request) {
	for k, vv := range src.Header {
		if isHopByHopHeader(k) {
			continue
		}
		for _, v := range vv {
			dst.Header.Add(k, v)
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

// writeCacheMeta persists response headers and (optionally) an expiry file
// for a cached entry. Callers must hold c.mu.
func (c *Cache) writeCacheMeta(path string, hdr http.Header, ttl time.Duration) {
	hdrPath := path + ".hdr"
	hf, err := os.Create(hdrPath)
	if err != nil {
		slog.Warn("cache header write error", "path", hdrPath, "error", err)
		os.Remove(path)
		return
	}
	for k, vv := range hdr {
		for _, v := range vv {
			fmt.Fprintf(hf, "%s:%s\n", k, v)
		}
	}
	hf.Close()

	expPath := path + ".exp"
	if ttl > 0 {
		ef, err := os.Create(expPath)
		if err != nil {
			slog.Warn("cache expiry write error", "path", expPath, "error", err)
			os.Remove(path)
			os.Remove(hdrPath)
			return
		}
		fmt.Fprintf(ef, "%d", time.Now().Add(ttl).Unix())
		ef.Close()
	} else {
		// Never-expire entries must not keep a stale expiry marker from a
		// previous TTL, otherwise shortening the TTL to 0 would not stick.
		os.Remove(expPath)
	}
}

func (c *Cache) Hits() int64   { return c.hits.Load() }
func (c *Cache) Misses() int64 { return c.misses.Load() }

func (c *Cache) CleanExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now().Unix()
	files, _ := os.ReadDir(c.dir)
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".exp" {
			data, err := os.ReadFile(filepath.Join(c.dir, f.Name()))
			if err != nil {
				continue
			}
			expiry, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
			if err != nil {
				continue
			}
			if now <= expiry {
				continue
			}
			base := filepath.Join(c.dir, strings.TrimSuffix(f.Name(), ".exp"))
			if info, err := os.Stat(base); err == nil {
				c.usedBytes -= info.Size()
			}
			os.Remove(base)
			os.Remove(base + ".hdr")
			os.Remove(base + ".exp")
		}
	}
}

// ScanSize walks the cache dir and initializes usedBytes from the files on
// disk, so size accounting survives process restarts.
func (c *Cache) ScanSize() {
	c.mu.Lock()
	defer c.mu.Unlock()
	var total int64
	_ = filepath.WalkDir(c.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if filepath.Ext(name) == ".exp" || filepath.Ext(name) == ".hdr" || strings.Contains(name, ".tmp") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	c.usedBytes = total
}

func (c *Cache) keyPath(key string) string {
	hash := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, hex.EncodeToString(hash[:]))
}
