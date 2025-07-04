package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Set test environment variables
	os.Setenv("WEBHOOK_URLS", "https://example.com/webhook1,https://example.com/webhook2")
	os.Setenv("CHECK_INTERVAL", "1h")
	os.Setenv("EXPIRATION_THRESHOLD", "168h") // 7 days
	os.Setenv("NAMESPACE", "test-namespace")
	os.Setenv("HEALTH_PORT", "9090")
	os.Setenv("LOG_LEVEL", "debug")

	defer func() {
		// Clean up
		os.Unsetenv("WEBHOOK_URLS")
		os.Unsetenv("CHECK_INTERVAL")
		os.Unsetenv("EXPIRATION_THRESHOLD")
		os.Unsetenv("NAMESPACE")
		os.Unsetenv("HEALTH_PORT")
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check webhook configuration
	if len(cfg.Webhooks) != 2 {
		t.Errorf("Expected 2 webhooks, got %d", len(cfg.Webhooks))
	}

	if cfg.Webhooks[0].URL != "https://example.com/webhook1" {
		t.Errorf("Expected webhook URL 'https://example.com/webhook1', got '%s'", cfg.Webhooks[0].URL)
	}

	// Check other configurations
	if cfg.CheckInterval != time.Hour {
		t.Errorf("Expected check interval '1h', got '%v'", cfg.CheckInterval)
	}

	if cfg.ExpirationThreshold != 168*time.Hour {
		t.Errorf("Expected expiration threshold '168h', got '%v'", cfg.ExpirationThreshold)
	}

	if cfg.Namespace != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got '%s'", cfg.Namespace)
	}

	if cfg.HealthPort != 9090 {
		t.Errorf("Expected health port 9090, got %d", cfg.HealthPort)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", cfg.LogLevel)
	}
}

func TestLoad_NoWebhooks(t *testing.T) {
	// Unset webhook URLs
	os.Unsetenv("WEBHOOK_URLS")

	_, err := Load()
	if err == nil {
		t.Error("Expected error when no webhooks configured, got nil")
	}
}
