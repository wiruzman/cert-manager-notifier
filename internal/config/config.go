package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	// Webhook configuration
	Webhooks []WebhookConfig `json:"webhooks"`

	// Monitoring configuration
	CheckInterval       time.Duration `json:"check_interval"`
	ExpirationThreshold time.Duration `json:"expiration_threshold"`

	// Kubernetes configuration
	Namespace string `json:"namespace"`

	// Health check configuration
	HealthPort int `json:"health_port"`

	// Logging configuration
	LogLevel string `json:"log_level"`
}

// WebhookConfig holds webhook configuration
type WebhookConfig struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Timeout time.Duration     `json:"timeout"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		CheckInterval:       24 * time.Hour,      // Check daily
		ExpirationThreshold: 30 * 24 * time.Hour, // 30 days
		Namespace:           "",                  // All namespaces
		HealthPort:          8080,
		LogLevel:            "info",
	}

	// Load webhook configurations
	webhooks, err := loadWebhooks()
	if err != nil {
		return nil, fmt.Errorf("failed to load webhooks: %w", err)
	}
	cfg.Webhooks = webhooks

	// Load optional configurations
	if val := os.Getenv("CHECK_INTERVAL"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			cfg.CheckInterval = duration
		}
	}

	if val := os.Getenv("EXPIRATION_THRESHOLD"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			cfg.ExpirationThreshold = duration
		}
	}

	if val := os.Getenv("NAMESPACE"); val != "" {
		cfg.Namespace = val
	}

	if val := os.Getenv("HEALTH_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			cfg.HealthPort = port
		}
	}

	if val := os.Getenv("LOG_LEVEL"); val != "" {
		cfg.LogLevel = val
	}

	return cfg, nil
}

// loadWebhooks loads webhook configurations from environment variables
func loadWebhooks() ([]WebhookConfig, error) {
	var webhooks []WebhookConfig

	// Support multiple webhooks via WEBHOOK_URLS (comma-separated)
	urls := os.Getenv("WEBHOOK_URLS")
	if urls == "" {
		return nil, fmt.Errorf("WEBHOOK_URLS environment variable is required")
	}

	urlList := strings.Split(urls, ",")
	for i, url := range urlList {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		webhook := WebhookConfig{
			Name:    fmt.Sprintf("webhook-%d", i+1),
			URL:     url,
			Headers: make(map[string]string),
			Timeout: 30 * time.Second,
		}

		// Load headers for this webhook
		headersKey := fmt.Sprintf("WEBHOOK_%d_HEADERS", i+1)
		if headers := os.Getenv(headersKey); headers != "" {
			headerPairs := strings.Split(headers, ",")
			for _, pair := range headerPairs {
				if kv := strings.SplitN(pair, ":", 2); len(kv) == 2 {
					webhook.Headers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
				}
			}
		}

		// Load timeout for this webhook
		timeoutKey := fmt.Sprintf("WEBHOOK_%d_TIMEOUT", i+1)
		if timeout := os.Getenv(timeoutKey); timeout != "" {
			if duration, err := time.ParseDuration(timeout); err == nil {
				webhook.Timeout = duration
			}
		}

		webhooks = append(webhooks, webhook)
	}

	if len(webhooks) == 0 {
		return nil, fmt.Errorf("no valid webhooks configured")
	}

	return webhooks, nil
}
