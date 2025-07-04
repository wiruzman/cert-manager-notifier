# Contributing to cert-manager-notifier

Thank you for your interest in contributing to cert-manager-notifier! This document provides guidelines for contributing to the project.

## Code of Conduct

This project adheres to the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md). By participating, you are expected to uphold this code.

## How to Contribute

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When creating a bug report, include:

- A clear description of the problem
- Steps to reproduce the issue
- Expected vs actual behavior
- Environment details (Kubernetes version, cert-manager version, etc.)
- Relevant logs or error messages

### Suggesting Enhancements

Enhancement suggestions are welcome! Please provide:

- A clear description of the enhancement
- Use cases and benefits
- Any implementation considerations

### Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

#### Pull Request Guidelines

- Include a clear description of changes
- Add tests for new functionality
- Update documentation as needed
- Ensure all tests pass
- Follow Go coding standards
- Keep commits focused and atomic

## Development Setup

### Prerequisites

- Go 1.24 or later
- Docker
- Kubernetes cluster (for testing)
- kubectl
- Helm 3.x

### Building the Project

```bash
# Clone the repository
git clone https://github.com/your-username/cert-manager-notifier.git
cd cert-manager-notifier

# Build the binary
make build

# Run tests
make test

# Run integration tests  
make test-integration

# Run E2E tests
make test-e2e

# Build Docker image
make docker-build
```

### Testing

```bash
# Run unit tests
go test ./...

# Run tests with coverage
go test -v -cover ./...

# Run specific test
go test -v ./internal/monitor
```

### Code Style

This project follows standard Go conventions:

- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and small

### Commit Message Format

Use conventional commit format:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance tasks

Examples:
```
feat(monitor): add certificate status tracking
fix(webhook): handle timeout errors gracefully
docs(README): update installation instructions
```

## Project Structure

```
├── cmd/                    # Application entrypoints
├── internal/               # Private application code
│   ├── config/            # Configuration management
│   ├── health/            # Health check handlers
│   ├── metrics/           # Prometheus metrics
│   ├── monitor/           # Certificate monitoring logic
│   └── webhook/           # Webhook notification handling
├── helm/                  # Helm chart
├── .github/               # GitHub workflows
├── docs/                  # Documentation
└── README.md
```

## Documentation

- Update README.md for user-facing changes
- Update code comments for implementation changes
- Add examples for new features
- Update Helm chart documentation

## Release Process

Releases are handled by maintainers:

1. Create release branch
2. Update version numbers
3. Update CHANGELOG.md
4. Create GitHub release
5. CI/CD builds and publishes Docker images

## Getting Help

- Open an issue for bugs or feature requests
- Start a discussion for questions or ideas
- Check existing documentation and issues first

Thank you for contributing to cert-manager-notifier!
