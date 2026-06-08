package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Provider interface {
	Send(subject, body string) error
}

type WebhookProvider struct {
	url    string
	client *http.Client
}

func NewWebhook(url string) *WebhookProvider {
	return &WebhookProvider{
		url:    url,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type webhookPayload struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Time    string `json:"time"`
}

func (w *WebhookProvider) Send(subject, body string) error {
	payload := webhookPayload{
		Subject: subject,
		Body:    body,
		Time:    time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("alert marshal: %w", err)
	}
	resp, err := w.client.Post(w.url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("alert webhook: %w", err)
	}
	resp.Body.Close()
	return nil
}

type LogProvider struct{}

func NewLog() *LogProvider { return &LogProvider{} }

func (l *LogProvider) Send(subject, body string) error {
	slog.Warn("alert triggered", "subject", subject, "body", body)
	return nil
}

type Engine struct {
	mu        sync.Mutex
	providers []Provider
	cooldown  time.Duration
	lastAlert map[string]time.Time
}

func NewEngine(cooldown time.Duration) *Engine {
	return &Engine{
		cooldown:  cooldown,
		lastAlert: make(map[string]time.Time),
	}
}

func (e *Engine) AddProvider(p Provider) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.providers = append(e.providers, p)
}

func (e *Engine) Trigger(key, subject, body string) {
	e.mu.Lock()
	last, ok := e.lastAlert[key]
	now := time.Now()
	if ok && now.Sub(last) < e.cooldown {
		e.mu.Unlock()
		return
	}
	e.lastAlert[key] = now
	providers := make([]Provider, len(e.providers))
	copy(providers, e.providers)
	e.mu.Unlock()

	for _, p := range providers {
		if err := p.Send(subject, body); err != nil {
			slog.Error("alert send failed", "key", key, "error", err)
		}
	}
}
