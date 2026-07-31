package config

import (
	"fmt"
	"log/slog"
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
	Alerts    AlertConfig             `yaml:"alerts"`

	// AuthTokenOrig holds the auth_token value from the yaml file so that
	// Save() can avoid persisting an env-injected DEVBOX_AUTH_TOKEN.
	AuthTokenOrig string `yaml:"-"`
}

type AlertConfig struct {
	WebhookURL string        `yaml:"webhook_url"`
	Cooldown   string        `yaml:"cooldown"`
	CooldownD  time.Duration `yaml:"-"`
}

type ServerConfig struct {
	Port      int    `yaml:"port"`
	AuthToken string `yaml:"auth_token"`
	PublicURL string `yaml:"public_url"`
}

type MirrorConfig struct {
	Enabled   bool          `yaml:"enabled"`
	Upstream  string        `yaml:"upstream"`
	CacheTTL  string        `yaml:"cache_ttl"`
	CacheTTLd time.Duration `yaml:"-"`
}

type GitProxyConfig struct {
	Enabled        bool          `yaml:"enabled"`
	GithubUpstream string        `yaml:"github_upstream"`
	GitlabUpstream string        `yaml:"gitlab_upstream"`
	RawUpstream    string        `yaml:"raw_upstream"`
	CacheTTL       string        `yaml:"cache_ttl"`
	CacheTTLd      time.Duration `yaml:"-"`
}

type CacheConfig struct {
	Dir          string `yaml:"dir"`
	MaxSize      string `yaml:"max_size"`
	MaxSizeBytes int64  `yaml:"-"`
}

type LoggingConfig struct {
	Level         string     `yaml:"level"`
	Format        string     `yaml:"format"`
	AccessLog     bool       `yaml:"access_log"`
	RetentionDays int        `yaml:"retention_days"`
	LogLevel      slog.Level `yaml:"-"` // parsed
}

type RateLimitConfig struct {
	Enabled        bool          `yaml:"enabled"`
	Rate           int           `yaml:"rate"`            // max requests per rolling window
	Interval       string        `yaml:"interval"`        // rolling window length, e.g. "3h"
	IntervalDur    time.Duration `yaml:"-"`               // parsed
	Whitelist      []string      `yaml:"whitelist"`       // IPs exempt from rate limiting
	Blacklist      []string      `yaml:"blacklist"`       // IPs always blocked
	TrustedProxies []string      `yaml:"trusted_proxies"` // CIDRs whose X-Real-IP / X-Forwarded-For are trusted
}

func (cfg *Config) Save(path string) error {
	// Never persist an env-injected auth token: DEVBOX_AUTH_TOKEN must stay
	// out of the config file. Save the file value (or empty) instead.
	orig := cfg.Server.AuthToken
	cfg.Server.AuthToken = cfg.AuthTokenOrig
	data, err := yaml.Marshal(cfg)
	cfg.Server.AuthToken = orig
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return os.Rename(tmp, path)
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

	// Remember the file value before env overrides replace it.
	cfg.AuthTokenOrig = cfg.Server.AuthToken

	applyDefaults(cfg)
	applyEnvOverrides(cfg)

	if err := ParseDurations(cfg); err != nil {
		return nil, err
	}
	if err := parseCacheMaxSize(cfg); err != nil {
		return nil, err
	}
	parseLogLevel(cfg)

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Mirrors == nil {
		cfg.Mirrors = make(map[string]MirrorConfig)
	}
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
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "text"
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
	if len(cfg.RateLimit.TrustedProxies) == 0 {
		// Loopback only by default: proxy headers are trusted solely from a
		// reverse proxy running on the same host (e.g. nginx + 127.0.0.1 bind).
		cfg.RateLimit.TrustedProxies = []string{"127.0.0.1/8", "::1/128"}
	}

	defaultMirrors := map[string]MirrorConfig{
		"npm":      {Enabled: true, Upstream: "https://registry.npmjs.org", CacheTTL: "7d"},
		"pypi":     {Enabled: true, Upstream: "https://pypi.org/simple", CacheTTL: "30d"},
		"docker":   {Enabled: true, Upstream: "https://registry-1.docker.io", CacheTTL: "0"},
		"golang":   {Enabled: true, Upstream: "https://proxy.golang.org", CacheTTL: "0"},
		"cran":     {Enabled: true, Upstream: "https://cran.r-project.org", CacheTTL: "30d"},
		"ghcr":     {Enabled: true, Upstream: "https://ghcr.io", CacheTTL: "0"},
		"quay":     {Enabled: true, Upstream: "https://quay.io", CacheTTL: "0"},
		"mcr":      {Enabled: true, Upstream: "https://mcr.microsoft.com", CacheTTL: "0"},
		"ghapi":    {Enabled: true, Upstream: "https://api.github.com", CacheTTL: "0"},
		"hf":       {Enabled: true, Upstream: "https://huggingface.co", CacheTTL: "0"},
		"conda":    {Enabled: true, Upstream: "https://repo.anaconda.com", CacheTTL: "30d"},
		"rubygems": {Enabled: true, Upstream: "https://rubygems.org", CacheTTL: "7d"},
		"cargo":    {Enabled: true, Upstream: "https://static.crates.io/crates", CacheTTL: "7d"},
		"nuget":    {Enabled: true, Upstream: "https://api.nuget.org/v3", CacheTTL: "7d"},
		"apt":      {Enabled: true, Upstream: "https://deb.debian.org/debian", CacheTTL: "0"},
		"alpine":   {Enabled: true, Upstream: "https://dl-cdn.alpinelinux.org/alpine", CacheTTL: "0"},
		"homebrew": {Enabled: true, Upstream: "https://ghcr.io/v2/homebrew/core", CacheTTL: "0"},
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
	if cfg.GitProxy.RawUpstream == "" {
		cfg.GitProxy.RawUpstream = "https://raw.githubusercontent.com"
	}
	if cfg.GitProxy.CacheTTL == "" {
		cfg.GitProxy.CacheTTL = "7d"
	}

	if cfg.Alerts.Cooldown == "" {
		cfg.Alerts.Cooldown = "5m"
	}
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("DEVBOX_SERVER_PORT"); v != "" {
		if n, err := mustInt(v); err == nil {
			cfg.Server.Port = n
		} else {
			slog.Error("invalid env DEVBOX_SERVER_PORT, keeping file value", "value", v)
		}
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
		if n, err := mustInt(v); err == nil {
			cfg.Logging.RetentionDays = n
		} else {
			slog.Error("invalid env DEVBOX_LOGGING_RETENTION_DAYS, keeping file value", "value", v)
		}
	}
	if v := os.Getenv("DEVBOX_RATE_LIMIT_ENABLED"); v != "" {
		if b, err := mustBool(v); err == nil {
			cfg.RateLimit.Enabled = b
		} else {
			slog.Error("invalid env DEVBOX_RATE_LIMIT_ENABLED, keeping file value", "value", v)
		}
	}
	if v := os.Getenv("DEVBOX_RATE_LIMIT_RATE"); v != "" {
		if n, err := mustInt(v); err == nil {
			cfg.RateLimit.Rate = n
		} else {
			slog.Error("invalid env DEVBOX_RATE_LIMIT_RATE, keeping file value", "value", v)
		}
	}
	if v := os.Getenv("DEVBOX_RATE_LIMIT_INTERVAL"); v != "" {
		cfg.RateLimit.Interval = v
	}
	if v := os.Getenv("DEVBOX_RATE_LIMIT_TRUSTED_PROXIES"); v != "" {
		cfg.RateLimit.TrustedProxies = strings.Split(v, ",")
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
			if b, err := mustBool(enabled); err == nil {
				m := cfg.Mirrors[name]
				m.Enabled = b
				cfg.Mirrors[name] = m
			} else {
				slog.Error("invalid env DEVBOX_MIRROR_"+strings.ToUpper(name)+"_ENABLED, keeping file value", "value", enabled)
			}
		}
	}
}

func ParseDurations(cfg *Config) error {
	for name, m := range cfg.Mirrors {
		d, err := ParseDuration(m.CacheTTL)
		if err != nil {
			return fmt.Errorf("mirror %s cache_ttl: %w", name, err)
		}
		m.CacheTTLd = d
		cfg.Mirrors[name] = m
	}

	d, err := ParseDuration(cfg.GitProxy.CacheTTL)
	if err != nil {
		return fmt.Errorf("gitproxy cache_ttl: %w", err)
	}
	cfg.GitProxy.CacheTTLd = d

	d, err = ParseDuration(cfg.RateLimit.Interval)
	if err != nil {
		return fmt.Errorf("rate_limit interval: %w", err)
	}
	cfg.RateLimit.IntervalDur = d

	cd, err := ParseDuration(cfg.Alerts.Cooldown)
	if err != nil {
		return fmt.Errorf("alerts cooldown: %w", err)
	}
	cfg.Alerts.CooldownD = cd
	return nil
}

func parseLogLevel(cfg *Config) {
	switch strings.ToLower(cfg.Logging.Level) {
	case "debug":
		cfg.Logging.LogLevel = slog.LevelDebug
	case "warn", "warning":
		cfg.Logging.LogLevel = slog.LevelWarn
	case "error":
		cfg.Logging.LogLevel = slog.LevelError
	default:
		cfg.Logging.LogLevel = slog.LevelInfo
	}
}

func ParseDuration(s string) (time.Duration, error) {
	if s == "0" {
		return 0, nil // never expire
	}
	// support "7d", "30d" etc
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid days: %s", s)
		}
		// Guard against int overflow when converting days to nanoseconds.
		if days > int((1<<63-1)/(24*int(time.Hour))) {
			return 0, fmt.Errorf("duration too large: %s", s)
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
			// Guard against multiplication overflow (e.g. huge values in GB).
			if sf.mul != 1 && val > (1<<63-1)/sf.mul {
				return fmt.Errorf("cache max_size too large: %s", s)
			}
			cfg.Cache.MaxSizeBytes = val * sf.mul
			return nil
		}
	}
	return fmt.Errorf("invalid cache max_size suffix: %s", s)
}

func mustInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func mustBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}
