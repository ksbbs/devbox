package mirror

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Mirror interface {
	Name() string
	Pattern() string
	Upstream() string
	SetUpstream(url string)
	ProxyHandler(cache *Cache) http.HandlerFunc
	HealthCheck() error
	IsEnabled() bool
	SetEnabled(enabled bool)
	CacheTTL() string
	SetCacheTTL(ttl string) error
}

var healthClient = &http.Client{Timeout: 5 * time.Second}

func HealthGet(url string) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return healthClient.Do(req)
}

var registry = struct {
	sync.RWMutex
	m map[string]Mirror
}{m: make(map[string]Mirror)}

func Register(m Mirror) {
	registry.Lock()
	registry.m[m.Name()] = m
	registry.Unlock()
}

func Get(name string) (Mirror, bool) {
	registry.RLock()
	defer registry.RUnlock()
	m, ok := registry.m[name]
	return m, ok
}

func All() []Mirror {
	registry.RLock()
	defer registry.RUnlock()
	mirrors := make([]Mirror, 0, len(registry.m))
	for _, m := range registry.m {
		mirrors = append(mirrors, m)
	}
	return mirrors
}
