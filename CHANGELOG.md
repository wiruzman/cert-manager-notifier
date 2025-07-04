# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of cert-manager-notifier
- Certificate monitoring for cert-manager certificates
- Webhook notifications for expired certificates
- Webhook notifications for certificates expiring within configurable threshold
- Health check endpoints (liveness and readiness)
- Prometheus metrics support
- Structured logging with configurable levels
- Kubernetes deployment via Helm chart
- Graceful shutdown handling
- Comprehensive unit tests
- Docker multi-stage build for optimized image size
- CI/CD pipeline with GitHub Actions

### Features
- Monitor certificates across all namespaces
- Configurable check interval (default: 5 minutes)
- Configurable expiration threshold (default: 30 days)
- Support for custom webhook URLs
- SSL/TLS certificate validation for webhooks
- Request timeout configuration
- Exponential backoff for failed webhook calls
- Structured JSON webhook payloads
- Kubernetes-native configuration via ConfigMaps and Secrets

### Security
- Runs as non-root user
- Minimal attack surface with Alpine Linux base image
- RBAC permissions limited to reading Certificate resources
- Secret-based webhook URL configuration
