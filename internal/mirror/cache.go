package mirror

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

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
		c.misses.Add(1)
		return nil, nil, false
	}
	hdrPath := path + ".hdr"
	hdrData, _ := os.ReadFile(hdrPath)
	hdr := make(http.Header)
	if hdrData != nil {
		lines := string(hdrData)
		for _, line := range splitLines(lines) {
			if idx := indexOf(line, ':'); idx > 0 {
				hdr.Set(line[:idx], strings.TrimSpace(line[idx+1:]))
			}
		}
	}
	if c.maxBytes > 0 && info.Size() > c.maxBytes {
		os.Remove(path)
		os.Remove(hdrPath)
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

	// Subtract old file size if overwriting
	if info, err := os.Stat(path); err == nil {
		c.usedBytes -= info.Size()
	}

	os.MkdirAll(filepath.Dir(path), 0755)

	f, err := os.Create(path)
	if err != nil {
		return
	}
	f.Write(data)
	f.Close()

	hdrPath := path + ".hdr"
	hf, err := os.Create(hdrPath)
	if err != nil {
		return
	}
	for k, vv := range hdr {
		for _, v := range vv {
			fmt.Fprintf(hf, "%s:%s\n", k, v)
		}
	}
	hf.Close()

	if ttl > 0 {
		expPath := path + ".exp"
		ef, err := os.Create(expPath)
		if err != nil {
			return
		}
		fmt.Fprintf(ef, "%d", time.Now().Add(ttl).Unix())
		ef.Close()
	}

	c.usedBytes += int64(len(data))

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

	filepath.WalkDir(c.dir, func(path string, d fs.DirEntry, err error) error {
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
		if filepath.Ext(name) == ".exp" || filepath.Ext(name) == ".hdr" {
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
	var expiry int64
	fmt.Sscanf(string(data), "%d", &expiry)
	return time.Now().Unix() > expiry
}

func (c *Cache) ProxyHTTP(w http.ResponseWriter, r *http.Request, upstream string, ttl time.Duration) {
	key := upstream + r.URL.Path + "?" + r.URL.RawQuery

	if !c.IsExpired(key) {
		data, hdr, ok := c.Get(key)
		if ok {
			for k, vv := range hdr {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(http.StatusOK)
			w.Write(data)
			return
		}
	}

	target := upstream + r.URL.Path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	resp, err := http.Get(target)
	if err != nil {
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	respHdr := resp.Header.Clone()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.Set(key, body, respHdr, ttl)
	}

	for k, vv := range respHdr {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func (c *Cache) ProxyStream(w http.ResponseWriter, r *http.Request, upstream string) {
	target := upstream + r.URL.Path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	resp, err := http.Get(target)
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
	io.Copy(w, resp.Body)
}

func (c *Cache) Hits() int64   { return c.hits.Load() }
func (c *Cache) Misses() int64 { return c.misses.Load() }

func (c *Cache) CleanExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	cleanDirExpired(c.dir, time.Now().Unix())
}

func cleanDirExpired(dir string, now int64) {
	files, _ := os.ReadDir(dir)
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".exp" {
			data, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				continue
			}
			var expiry int64
			fmt.Sscanf(string(data), "%d", &expiry)
			if now > expiry {
				base := filepath.Join(dir, stringsTrimSuffix(f.Name(), ".exp"))
				os.Remove(base)
				os.Remove(base + ".hdr")
				os.Remove(base + ".exp")
			}
		}
	}
}

func (c *Cache) keyPath(key string) string {
	hash := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, hex.EncodeToString(hash[:]))
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(s) && s[start:] != "" {
		lines = append(lines, s[start:])
	}
	return lines
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func stringsTrimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}
