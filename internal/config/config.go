package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig            `yaml:"server"`
	Mirrors   map[string]MirrorConfig `yaml:"mirrors"`
	GitProxy  GitProxyConfig          `yaml:"gitproxy"`
	Cache     CacheConfig             `yaml:"cache"`
	Logging   LoggingConfig           `yaml:"logging"`
	RateLimit RateLimitConfig         `yaml:"rate_limit"`
}

type ServerConfig struct {
	Port      int    `yaml:"port"`
	AuthToken string `yaml:"auth_token"`
	PublicURL string `yaml:"public_url"`
}

type MirrorConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Upstream  string `yaml:"upstream"`
	CacheTTL  string `yaml:"cache_ttl"`
	CacheTTLd time.Duration
}

type GitProxyConfig struct {
	Enabled        bool   `yaml:"enabled"`
	GithubUpstream string `yaml:"github_upstream"`
	GitlabUpstream string `yaml:"gitlab_upstream"`
	CacheTTL       string `yaml:"cache_ttl"`
	CacheTTLd      time.Duration
}

type CacheConfig struct {
	Dir     string `yaml:"dir"`
	MaxSize string `yaml:"max_size"`
	MaxSizeBytes int64
}

type LoggingConfig struct {
	Level         string `yaml:"level"`
	AccessLog     bool   `yaml:"access_log"`
	RetentionDays int    `yaml:"retention_days"`
}

type RateLimitConfig struct {
	Enabled      bool          `yaml:"enabled"`
	Rate         int           `yaml:"rate"`       // max requests per interval
	Interval     string        `yaml:"interval"`   // e.g. "3h"
	IntervalDur  time.Duration // parsed
	Whitelist    []string      `yaml:"whitelist"`  // IPs exempt from rate limiting
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyDefaults(cfg)
	applyEnvOverrides(cfg)

	if err := parseDurations(cfg); err != nil {
		return nil, err
	}
	if err := parseCacheMaxSize(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Cache.Dir == "" {
		cfg.Cache.Dir = "/data/cache"
	}
	if cfg.Cache.MaxSize == "" {
		cfg.Cache.MaxSize = "5GB"
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.RetentionDays == 0 {
		cfg.Logging.RetentionDays = 30
	}
	if cfg.RateLimit.Interval == "" {
		cfg.RateLimit.Interval = "3h"
	}
	if cfg.RateLimit.Rate == 0 {
		cfg.RateLimit.Rate = 500
	}

	defaultMirrors := map[string]MirrorConfig{
		"npm":    {Enabled: true, Upstream: "https://registry.npmjs.org", CacheTTL: "7d"},
		"pypi":   {Enabled: true, Upstream: "https://pypi.org/simple", CacheTTL: "30d"},
		"docker": {Enabled: true, Upstream: "https://registry-1.docker.io", CacheTTL: "0"},
		"golang": {Enabled: true, Upstream: "https://proxy.golang.org", CacheTTL: "0"},
		"cran":   {Enabled: true, Upstream: "https://cran.r-project.org", CacheTTL: "30d"},
		"ghcr":   {Enabled: true, Upstream: "https://ghcr.io", CacheTTL: "0"},
		"quay":   {Enabled: true, Upstream: "https://quay.io", CacheTTL: "0"},
		"mcr":    {Enabled: true, Upstream: "https://mcr.microsoft.com", CacheTTL: "0"},
		"ghapi":  {Enabled: true, Upstream: "https://api.github.com", CacheTTL: "0"},
			"hf":     {Enabled: true, Upstream: "https://huggingface.co", CacheTTL: "7d"},
	}
	for name, def := range defaultMirrors {
		if _, ok := cfg.Mirrors[name]; !ok {
			cfg.Mirrors[name] = def
		}
	}

	if cfg.GitProxy.GithubUpstream == "" {
		cfg.GitProxy.GithubUpstream = "https://github.com"
	}
	if cfg.GitProxy.GitlabUpstream == "" {
		cfg.GitProxy.GitlabUpstream = "https://gitlab.com"
	}
	if cfg.GitProxy.CacheTTL == "" {
		cfg.GitProxy.CacheTTL = "7d"
	}
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("DEVBOX_SERVER_PORT"); v != "" {
		cfg.Server.Port = mustInt(v)
	}
	if v := os.Getenv("DEVBOX_AUTH_TOKEN"); v != "" {
		cfg.Server.AuthToken = v
	}
	if v := os.Getenv("DEVBOX_PUBLIC_URL"); v != "" {
		cfg.Server.PublicURL = v
	}
	if v := os.Getenv("DEVBOX_CACHE_DIR"); v != "" {
		cfg.Cache.Dir = v
	}
	if v := os.Getenv("DEVBOX_CACHE_MAX_SIZE"); v != "" {
		cfg.Cache.MaxSize = v
	}
	if v := os.Getenv("DEVBOX_LOGGING_RETENTION_DAYS"); v != "" {
		cfg.Logging.RetentionDays = mustInt(v)
	}
	if v := os.Getenv("DEVBOX_RATE_LIMIT_ENABLED"); v != "" {
		cfg.RateLimit.Enabled = mustBool(v)
	}
	if v := os.Getenv("DEVBOX_RATE_LIMIT_RATE"); v != "" {
		cfg.RateLimit.Rate = mustInt(v)
	}
	if v := os.Getenv("DEVBOX_RATE_LIMIT_INTERVAL"); v != "" {
		cfg.RateLimit.Interval = v
	}

	for name := range cfg.Mirrors {
		upstream := os.Getenv(fmt.Sprintf("DEVBOX_MIRROR_%s_UPSTREAM", strings.ToUpper(name)))
		if upstream != "" {
			m := cfg.Mirrors[name]
			m.Upstream = upstream
			cfg.Mirrors[name] = m
		}
		enabled := os.Getenv(fmt.Sprintf("DEVBOX_MIRROR_%s_ENABLED", strings.ToUpper(name)))
		if enabled != "" {
			m := cfg.Mirrors[name]
			m.Enabled = mustBool(enabled)
			cfg.Mirrors[name] = m
		}
	}
}

func parseDurations(cfg *Config) error {
	for name, m := range cfg.Mirrors {
		d, err := parseDuration(m.CacheTTL)
		if err != nil {
			return fmt.Errorf("mirror %s cache_ttl: %w", name, err)
		}
		m.CacheTTLd = d
		cfg.Mirrors[name] = m
	}

	d, err := parseDuration(cfg.GitProxy.CacheTTL)
	if err != nil {
		return fmt.Errorf("gitproxy cache_ttl: %w", err)
	}
	cfg.GitProxy.CacheTTLd = d

	d, err = parseDuration(cfg.RateLimit.Interval)
	if err != nil {
		return fmt.Errorf("rate_limit interval: %w", err)
	}
	cfg.RateLimit.IntervalDur = d
	return nil
}

func parseDuration(s string) (time.Duration, error) {
	if s == "0" {
		return 0, nil // never expire
	}
	// support "7d", "30d" etc
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid days: %s", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func parseCacheMaxSize(cfg *Config) error {
	s := cfg.Cache.MaxSize
	// Order matters: check longer suffixes first so "5GB" doesn't match "B"
	suffixes := []struct {
		suffix string
		mul    int64
	}{
		{"GB", 1 << 30}, {"MB", 1 << 20}, {"KB", 1 << 10}, {"B", 1},
	}
	for _, sf := range suffixes {
		if strings.HasSuffix(s, sf.suffix) {
			val, err := strconv.ParseInt(strings.TrimSuffix(s, sf.suffix), 10, 64)
			if err != nil {
				return fmt.Errorf("invalid cache max_size: %s", s)
			}
			cfg.Cache.MaxSizeBytes = val * sf.mul
			return nil
		}
	}
	return fmt.Errorf("invalid cache max_size suffix: %s", s)
}

func mustInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func mustBool(s string) bool {
	v, _ := strconv.ParseBool(s)
	return v
}