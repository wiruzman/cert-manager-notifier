package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/wiruzman/cert-manager-notifier/internal/config"
)

func TestNotifier_SendExpiredNotification(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create notifier
	webhooks := []config.WebhookConfig{
		{
			Name:    "test-webhook",
			URL:     server.URL,
			Headers: map[string]string{},
			Timeout: 5 * time.Second,
		},
	}

	logger := logrus.NewEntry(logrus.New())
	notifier := NewNotifier(webhooks, logger)

	// Test expired notification
	ctx := context.Background()
	expiresAt := time.Now().Add(-24 * time.Hour) // Expired yesterday

	err := notifier.SendExpiredNotification(ctx, "test-cert", "default", "letsencrypt", []string{"example.com"}, expiresAt)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestNotifier_SendExpiringNotification(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create notifier
	webhooks := []config.WebhookConfig{
		{
			Name:    "test-webhook",
			URL:     server.URL,
			Headers: map[string]string{},
			Timeout: 5 * time.Second,
		},
	}

	logger := logrus.NewEntry(logrus.New())
	notifier := NewNotifier(webhooks, logger)

	// Test expiring notification
	ctx := context.Background()
	expiresAt := time.Now().Add(15 * 24 * time.Hour) // Expires in 15 days

	err := notifier.SendExpiringNotification(ctx, "test-cert", "default", "letsencrypt", []string{"example.com"}, expiresAt)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestNotifier_SendNotification_FailedWebhook(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create notifier
	webhooks := []config.WebhookConfig{
		{
			Name:    "test-webhook",
			URL:     server.URL,
			Headers: map[string]string{},
			Timeout: 5 * time.Second,
		},
	}

	logger := logrus.NewEntry(logrus.New())
	notifier := NewNotifier(webhooks, logger)

	// Test failed notification
	ctx := context.Background()
	expiresAt := time.Now().Add(-24 * time.Hour)

	err := notifier.SendExpiredNotification(ctx, "test-cert", "default", "letsencrypt", []string{"example.com"}, expiresAt)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}
