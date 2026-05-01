package mirror

import (
	"net/http"
	"sync"
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