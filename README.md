# Cert-Manager Notifier

A Kubernetes application that monitors cert-manager certificates and sends webhook notifications for expired certificates and certificates expiring within 30 days.

## Features

- **Certificate Monitoring**: Monitors cert-manager Certificate resources across all namespaces or a specific namespace
- **Webhook Notifications**: Sends HTTP webhook notifications for:
  - Expired certificates (immediate notification)
  - Certificates expiring within 30 days (daily notifications)
- **Multiple Webhooks**: Support for multiple webhook endpoints
- **Kubernetes Native**: Designed to run in Kubernetes with proper RBAC
- **Helm Chart**: Easy deployment with Helm
- **Health Checks**: Built-in health and readiness probes
- **Configurable**: Flexible configuration via environment variables

## Quick Start

### Prerequisites

- Kubernetes cluster with cert-manager installed
- Helm 3.x
- Docker (for building custom images)

### Installation

1. **Install cert-manager (if not already installed)**:
   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml
   ```

2. **Build the Docker image**:
   ```bash
   make docker-build
   ```

3. **Deploy with Helm**:
   ```bash
   # Install with default webhook (httpbin for testing)
   make helm-install
   
   # Or install with custom webhook
   helm install cert-manager-notifier helm/cert-manager-notifier \
     --set config.webhookUrls="https://your-webhook-url.com/notify"
   ```

### Configuration

The application is configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `WEBHOOK_URLS` | Comma-separated list of webhook URLs | Required |
| `CHECK_INTERVAL` | How often to check certificates | `24h` |
| `EXPIRATION_THRESHOLD` | Notify when certificates expire within this period | `720h` (30 days) |
| `NAMESPACE` | Kubernetes namespace to monitor (empty = all namespaces) | `` |
| `HEALTH_PORT` | Port for health check server | `8080` |
| `LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |

### Webhook Configuration

#### Multiple Webhooks
```yaml
config:
  webhookUrls: "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK,https://discord.com/api/webhooks/YOUR/DISCORD/WEBHOOK"
```

#### Webhook Headers (for authentication)
```yaml
config:
  webhookUrls: "https://api.example.com/webhooks/notify"
env:
  - name: WEBHOOK_1_HEADERS
    value: "Authorization:Bearer your-token,X-Custom-Header:custom-value"
```

### Webhook Payload

The webhook receives a JSON payload with the following structure:

```json
{
  "type": "expired|expiring",
  "message": "Certificate default/example-cert has expired",
  "certificate": {
    "name": "example-cert",
    "namespace": "default",
    "issuer": "letsencrypt-prod",
    "dns_names": ["example.com", "www.example.com"],
    "expires_at": "2023-12-31T23:59:59Z"
  },
  "timestamp": "2023-12-01T10:00:00Z"
}
```

## Development

### Local Development

1. **Clone the repository**:
   ```bash
   git clone https://github.com/wiruzman/cert-manager-notifier.git
   cd cert-manager-notifier
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Run tests**:
   ```bash
   make test
   ```

4. **Run locally** (requires kubeconfig):
   ```bash
   make run-local
   ```

### Building and Testing

```bash
# Build the application
make build

# Run tests with coverage
make test-coverage

# Build Docker image
make docker-build

# Lint Helm chart
make helm-lint

# Template Helm chart
make helm-template
```

## Deployment

### Docker Desktop Kubernetes

For local testing with Docker Desktop:

```bash
# Build and deploy locally
make deploy-local

# Check deployment status
kubectl get pods -n default -l app.kubernetes.io/name=cert-manager-notifier

# View logs
kubectl logs -f deployment/cert-manager-notifier -n default
```

### Production Deployment

1. **Build and push Docker image**:
   ```bash
   export REGISTRY=your-registry.com
   make docker-build
   make docker-push
   ```

2. **Deploy with Helm**:
   ```bash
   helm install cert-manager-notifier helm/cert-manager-notifier \
     --namespace cert-manager-notifier \
     --create-namespace \
     --set image.repository=your-registry.com/cert-manager-notifier \
     --set image.tag=latest \
     --set config.webhookUrls="https://your-webhook-url.com/notify" \
     --set config.namespace="production"
   ```

## Monitoring

The application exposes health check endpoints:

- `/health` - Liveness probe
- `/ready` - Readiness probe

### Prometheus Metrics (Optional)

Enable ServiceMonitor for Prometheus scraping:

```yaml
monitoring:
  serviceMonitor:
    enabled: true
    interval: 30s
    labels:
      prometheus: kube-prometheus
```

## RBAC

The application requires the following Kubernetes permissions:

- `get`, `list`, `watch` on `certificates.cert-manager.io`
- `create` on `events` (for audit logging)

These permissions are automatically configured when using the Helm chart.

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Kubernetes    │    │ cert-manager-   │    │    Webhook      │
│   cert-manager  │───▶│   notifier      │───▶│   Endpoints     │
│   Certificates  │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │  Health Check   │
                       │   Endpoints     │
                       └─────────────────┘
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Troubleshooting

### Common Issues

1. **RBAC Permissions**: Ensure the service account has proper permissions to read Certificate resources
2. **Webhook Connectivity**: Verify webhook URLs are accessible from the cluster
3. **cert-manager CRDs**: Ensure cert-manager is properly installed and Certificate CRDs are available

### Debugging

Enable debug logging:
```bash
helm upgrade cert-manager-notifier helm/cert-manager-notifier \
  --set config.logLevel=debug
```

View logs:
```bash
kubectl logs -f deployment/cert-manager-notifier -n default
```

## Webhook Integration Examples

### Slack
```bash
# Create a Slack webhook URL and use it
helm install cert-manager-notifier helm/cert-manager-notifier \
  --set config.webhookUrls="https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
```

### Discord
```bash
# Create a Discord webhook URL and use it
helm install cert-manager-notifier helm/cert-manager-notifier \
  --set config.webhookUrls="https://discord.com/api/webhooks/YOUR/DISCORD/WEBHOOK"
```

### Microsoft Teams
```bash
# Create a Teams webhook URL and use it
helm install cert-manager-notifier helm/cert-manager-notifier \
  --set config.webhookUrls="https://outlook.office.com/webhook/YOUR/TEAMS/WEBHOOK"
```

### Custom Webhook Server
```bash
# Use your own webhook server
helm install cert-manager-notifier helm/cert-manager-notifier \
  --set config.webhookUrls="https://your-api.com/webhooks/cert-notifications"
```
