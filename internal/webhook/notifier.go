package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/wiruzman/cert-manager-notifier/internal/config"
)

// NotificationPayload represents the webhook notification payload
type NotificationPayload struct {
	Type        string `json:"type"`
	Message     string `json:"message"`
	Certificate struct {
		Name      string    `json:"name"`
		Namespace string    `json:"namespace"`
		Issuer    string    `json:"issuer"`
		DNSNames  []string  `json:"dns_names"`
		ExpiresAt time.Time `json:"expires_at"`
	} `json:"certificate"`
	Timestamp time.Time `json:"timestamp"`
}

// Notifier handles webhook notifications
type Notifier struct {
	webhooks []config.WebhookConfig
	client   *http.Client
	logger   *logrus.Entry
}

// NewNotifier creates a new webhook notifier
func NewNotifier(webhooks []config.WebhookConfig, logger *logrus.Entry) *Notifier {
	return &Notifier{
		webhooks: webhooks,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.WithField("component", "webhook-notifier"),
	}
}

// SendExpiredNotification sends a notification for expired certificates
func (n *Notifier) SendExpiredNotification(ctx context.Context, certName, namespace, issuer string, dnsNames []string, expiresAt time.Time) error {
	payload := NotificationPayload{
		Type:      "expired",
		Message:   fmt.Sprintf("Certificate %s/%s has expired", namespace, certName),
		Timestamp: time.Now(),
	}

	payload.Certificate.Name = certName
	payload.Certificate.Namespace = namespace
	payload.Certificate.Issuer = issuer
	payload.Certificate.DNSNames = dnsNames
	payload.Certificate.ExpiresAt = expiresAt

	return n.sendNotification(ctx, payload)
}

// SendExpiringNotification sends a notification for certificates expiring soon
func (n *Notifier) SendExpiringNotification(ctx context.Context, certName, namespace, issuer string, dnsNames []string, expiresAt time.Time) error {
	daysUntilExpiry := int(time.Until(expiresAt).Hours() / 24)

	payload := NotificationPayload{
		Type:      "expiring",
		Message:   fmt.Sprintf("Certificate %s/%s expires in %d days", namespace, certName, daysUntilExpiry),
		Timestamp: time.Now(),
	}

	payload.Certificate.Name = certName
	payload.Certificate.Namespace = namespace
	payload.Certificate.Issuer = issuer
	payload.Certificate.DNSNames = dnsNames
	payload.Certificate.ExpiresAt = expiresAt

	return n.sendNotification(ctx, payload)
}

// sendNotification sends the notification to all configured webhooks
func (n *Notifier) sendNotification(ctx context.Context, payload NotificationPayload) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal notification payload: %w", err)
	}

	var lastError error
	successCount := 0

	for _, webhook := range n.webhooks {
		if err := n.sendToWebhook(ctx, webhook, jsonPayload); err != nil {
			n.logger.WithError(err).WithField("webhook", webhook.Name).Error("Failed to send notification")
			lastError = err
		} else {
			successCount++
			n.logger.WithField("webhook", webhook.Name).Info("Notification sent successfully")
		}
	}

	if successCount == 0 {
		return fmt.Errorf("failed to send notification to any webhook: %w", lastError)
	}

	if successCount < len(n.webhooks) {
		n.logger.WithField("success_count", successCount).WithField("total_webhooks", len(n.webhooks)).Warn("Some webhooks failed")
	}

	return nil
}

// sendToWebhook sends the notification to a specific webhook
func (n *Notifier) sendToWebhook(ctx context.Context, webhook config.WebhookConfig, payload []byte) error {
	// Create request with timeout context
	reqCtx, cancel := context.WithTimeout(ctx, webhook.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "POST", webhook.URL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "cert-manager-notifier/1.0")

	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status: %d", resp.StatusCode)
	}

	return nil
}
