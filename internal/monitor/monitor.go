package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagerclient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/wiruzman/cert-manager-notifier/internal/config"
	"github.com/wiruzman/cert-manager-notifier/internal/webhook"
)

// CertificateMonitor monitors cert-manager certificates
type CertificateMonitor struct {
	client        certmanagerclient.Interface
	config        *config.Config
	notifier      *webhook.Notifier
	logger        *logrus.Entry
	notifiedCerts map[string]time.Time
	notifiedMutex sync.RWMutex
}

// NewCertificateMonitor creates a new certificate monitor
func NewCertificateMonitor(k8sConfig *rest.Config, cfg *config.Config, notifier *webhook.Notifier, logger *logrus.Entry) (*CertificateMonitor, error) {
	client, err := certmanagerclient.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert-manager client: %w", err)
	}

	return &CertificateMonitor{
		client:        client,
		config:        cfg,
		notifier:      notifier,
		logger:        logger.WithField("component", "cert-monitor"),
		notifiedCerts: make(map[string]time.Time),
	}, nil
}

// Run starts the certificate monitoring loop
func (m *CertificateMonitor) Run(ctx context.Context) error {
	m.logger.Info("Starting certificate monitor")

	// Initial check
	if err := m.checkCertificates(ctx); err != nil {
		m.logger.WithError(err).Error("Initial certificate check failed")
	}

	// Start periodic checks
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Certificate monitor stopped")
			return nil
		case <-ticker.C:
			if err := m.checkCertificates(ctx); err != nil {
				m.logger.WithError(err).Error("Certificate check failed")
			}
		}
	}
}

// checkCertificates checks all certificates for expiration
func (m *CertificateMonitor) checkCertificates(ctx context.Context) error {
	m.logger.Info("Checking certificates")

	// Get all certificates
	certificates, err := m.getCertificates(ctx)
	if err != nil {
		return fmt.Errorf("failed to get certificates: %w", err)
	}

	m.logger.WithField("count", len(certificates.Items)).Info("Found certificates")

	now := time.Now()
	expiredCount := 0
	expiringCount := 0

	for i := range certificates.Items {
		cert := &certificates.Items[i]
		if err := m.checkCertificate(ctx, cert, now); err != nil {
			m.logger.WithError(err).WithField("certificate", cert.Name).Error("Failed to check certificate")
			continue
		}

		if m.isCertificateExpired(cert, now) {
			expiredCount++
		} else if m.isCertificateExpiring(cert, now) {
			expiringCount++
		}
	}

	m.logger.WithField("expired", expiredCount).WithField("expiring", expiringCount).Info("Certificate check completed")
	return nil
}

// getCertificates retrieves all certificates from the specified namespace
func (m *CertificateMonitor) getCertificates(ctx context.Context) (*certmanagerv1.CertificateList, error) {
	listOptions := metav1.ListOptions{}

	if m.config.Namespace != "" {
		return m.client.CertmanagerV1().Certificates(m.config.Namespace).List(ctx, listOptions)
	}

	// Get certificates from all namespaces
	return m.client.CertmanagerV1().Certificates(metav1.NamespaceAll).List(ctx, listOptions)
}

// checkCertificate checks a single certificate for expiration
func (m *CertificateMonitor) checkCertificate(ctx context.Context, cert *certmanagerv1.Certificate, now time.Time) error {
	// Get certificate status
	if cert.Status.NotAfter == nil {
		m.logger.WithField("certificate", cert.Name).Debug("Certificate has no expiration date")
		return nil
	}

	expirationTime := cert.Status.NotAfter.Time
	certKey := fmt.Sprintf("%s/%s", cert.Namespace, cert.Name)

	// Check if certificate is expired
	if m.isCertificateExpired(cert, now) {
		// Check if we've already notified about this expired certificate today
		if !m.shouldNotifyExpired(certKey, now) {
			return nil
		}

		m.logger.WithField("certificate", cert.Name).WithField("expires_at", expirationTime).Warn("Certificate is expired")

		if err := m.notifier.SendExpiredNotification(ctx, cert.Name, cert.Namespace, m.getIssuerName(cert), cert.Spec.DNSNames, expirationTime); err != nil {
			return fmt.Errorf("failed to send expired notification: %w", err)
		}

		m.markNotified(certKey, now)
		return nil
	}

	// Check if certificate is expiring soon
	if m.isCertificateExpiring(cert, now) {
		// Check if we've already notified about this expiring certificate today
		if !m.shouldNotifyExpiring(certKey, now) {
			return nil
		}

		daysUntilExpiry := int(time.Until(expirationTime).Hours() / 24)
		m.logger.WithField("certificate", cert.Name).WithField("days_until_expiry", daysUntilExpiry).Info("Certificate is expiring soon")

		if err := m.notifier.SendExpiringNotification(ctx, cert.Name, cert.Namespace, m.getIssuerName(cert), cert.Spec.DNSNames, expirationTime); err != nil {
			return fmt.Errorf("failed to send expiring notification: %w", err)
		}

		m.markNotified(certKey, now)
		return nil
	}

	return nil
}

// isCertificateExpired checks if a certificate is expired
func (m *CertificateMonitor) isCertificateExpired(cert *certmanagerv1.Certificate, now time.Time) bool {
	if cert.Status.NotAfter == nil {
		return false
	}
	return now.After(cert.Status.NotAfter.Time)
}

// isCertificateExpiring checks if a certificate is expiring within the threshold
func (m *CertificateMonitor) isCertificateExpiring(cert *certmanagerv1.Certificate, now time.Time) bool {
	if cert.Status.NotAfter == nil {
		return false
	}
	return now.Add(m.config.ExpirationThreshold).After(cert.Status.NotAfter.Time)
}

// shouldNotifyExpired checks if we should send an expired notification
func (m *CertificateMonitor) shouldNotifyExpired(certKey string, now time.Time) bool {
	m.notifiedMutex.RLock()
	defer m.notifiedMutex.RUnlock()

	lastNotified, exists := m.notifiedCerts[certKey]
	if !exists {
		return true
	}

	// Notify about expired certificates once per day
	return now.Sub(lastNotified) >= 24*time.Hour
}

// shouldNotifyExpiring checks if we should send an expiring notification
func (m *CertificateMonitor) shouldNotifyExpiring(certKey string, now time.Time) bool {
	m.notifiedMutex.RLock()
	defer m.notifiedMutex.RUnlock()

	lastNotified, exists := m.notifiedCerts[certKey]
	if !exists {
		return true
	}

	// Notify about expiring certificates once per day
	return now.Sub(lastNotified) >= 24*time.Hour
}

// markNotified marks a certificate as having been notified
func (m *CertificateMonitor) markNotified(certKey string, now time.Time) {
	m.notifiedMutex.Lock()
	defer m.notifiedMutex.Unlock()
	m.notifiedCerts[certKey] = now
}

// getIssuerName extracts the issuer name from the certificate
func (m *CertificateMonitor) getIssuerName(cert *certmanagerv1.Certificate) string {
	if cert.Spec.IssuerRef.Name != "" {
		return cert.Spec.IssuerRef.Name
	}
	return "unknown"
}
