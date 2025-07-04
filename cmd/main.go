package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/wiruzman/cert-manager-notifier/internal/config"
	"github.com/wiruzman/cert-manager-notifier/internal/health"
	"github.com/wiruzman/cert-manager-notifier/internal/monitor"
	"github.com/wiruzman/cert-manager-notifier/internal/webhook"
)

func main() {
	// Setup logging
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	log := logrus.WithField("component", "main")
	log.Info("Starting cert-manager-notifier")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// Create Kubernetes client
	k8sConfig, err := getKubernetesConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to get Kubernetes config")
	}

	// Create webhook notifier
	webhookNotifier := webhook.NewNotifier(cfg.Webhooks, log)

	// Create certificate monitor
	certMonitor, err := monitor.NewCertificateMonitor(k8sConfig, cfg, webhookNotifier, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to create certificate monitor")
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start health check server
	healthServer := health.NewHealthServer(cfg.HealthPort)
	go func() {
		if err := healthServer.Start(); err != nil {
			log.WithError(err).Error("Health server failed")
		}
	}()

	// Set initial health status
	health.SetHealthy(true)

	// Start certificate monitor
	go func() {
		if err := certMonitor.Run(ctx); err != nil {
			log.WithError(err).Error("Certificate monitor failed")
			cancel()
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Info("Shutting down...")

	// Set health status to unhealthy
	health.SetHealthy(false)

	// Cancel context to stop all operations
	cancel()

	// Give some time for graceful shutdown
	time.Sleep(2 * time.Second)
	log.Info("Shutdown complete")
}

func getKubernetesConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("HOME") + "/.kube/config"
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}
