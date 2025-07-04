# Copilot Instructions for cert-manager-notifier

<!-- Use this file to provide workspace-specific custom instructions to Copilot. For more details, visit https://code.visualstudio.com/docs/copilot/copilot-customization#_use-a-githubcopilotinstructionsmd-file -->

This is a Kubernetes certificate monitoring application that monitors cert-manager certificates and sends webhook notifications.

## Project Structure
- This is a Go application that uses the Kubernetes client-go library
- The application monitors cert-manager Certificate resources
- It sends webhook notifications for expired certificates and certificates expiring within 30 days
- The project includes a Helm chart for easy Kubernetes deployment

## Key Technologies
- Go 1.24+
- Kubernetes client-go
- cert-manager CRDs
- Helm charts
- Docker for containerization

## Development Guidelines
- Use structured logging with logrus or zap
- Implement proper error handling
- Follow Go best practices and conventions
- Use interfaces for testability
- Include comprehensive unit tests
- Use environment variables for configuration
- Implement graceful shutdown
- Use context for cancellation and timeouts

## Kubernetes Considerations
- The application runs as a Kubernetes deployment
- It needs RBAC permissions to read Certificate resources
- Configuration is provided via ConfigMaps and Secrets
- Health checks should be implemented for liveness and readiness probes
