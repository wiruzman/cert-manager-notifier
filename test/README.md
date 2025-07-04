# Testing

This directory contains Go-based tests that replace the previous shell scripts.

## Prerequisites

- Go 1.24+
- Docker
- kubectl
- Helm
- **kind** (for local E2E testing with isolated clusters)

## Test Types

### Unit Tests
Located in the main project directories, these test individual functions and components.

```bash
go test -v -race ./...
```

### Integration Tests
Tests webhook functionality, application building, and Helm chart validation.

```bash
cd test
go test -v ./integration
```

### E2E Tests
Full end-to-end tests that deploy the application to a Kubernetes cluster.

```bash
# With existing cluster
cd test
go test -v ./e2e -timeout 20m

# With kind cluster (recommended for local development)
cd test
USE_KIND_CLUSTER=true go test -v ./e2e -timeout 20m
```

## Test Runner

Use the test runner for comprehensive testing:

```bash
cd test/cmd
go run test-runner.go -type all

# Run specific test types
go run test-runner.go -type unit
go run test-runner.go -type integration
go run test-runner.go -type e2e

# Use kind cluster for E2E tests
go run test-runner.go -type e2e -use-kind

# Skip E2E tests
go run test-runner.go -skip-e2e

# Skip cleanup for debugging
go run test-runner.go -skip-cleanup

# Verbose output
go run test-runner.go -v
```

## Environment Variables

- `SKIP_E2E_TESTS=true` - Skip E2E tests
- `SKIP_CLEANUP=true` - Skip cleanup after tests
- `USE_KIND_CLUSTER=true` - Use kind cluster instead of existing cluster
- `CI=true` - Automatically set in CI environments, enables kind cluster usage
- `KUBECONFIG` - Path to kubeconfig file

## Kind Cluster Support

The E2E tests support creating isolated kind clusters for testing:

### Automatic Detection
- In CI environments (`CI=true`), kind cluster is used automatically
- Set `USE_KIND_CLUSTER=true` for local development
- **Single Source of Truth**: The E2E test suite manages kind cluster lifecycle

### Benefits of Using Kind
- **Isolation** - Each test run gets a fresh cluster
- **Consistency** - Same environment as CI
- **No Conflicts** - Won't interfere with existing clusters
- **Easy Cleanup** - Entire cluster is deleted after tests
- **Full Control** - Test suite manages cluster creation, configuration, and teardown

### Image Handling
- **Local Development**: Builds Docker image from source and loads into kind
- **CI Environment**: Uses pre-built image from GitHub Actions workflow
- **Automatic Detection**: Switches behavior based on environment

### Examples

```bash
# Run E2E tests with kind cluster
USE_KIND_CLUSTER=true go test -v ./e2e

# Run all tests with kind and preserve environment for debugging
USE_KIND_CLUSTER=true SKIP_CLEANUP=true go test -v ./e2e

# Use test runner with kind
cd test/cmd
go run test-runner.go -type e2e -use-kind
```

## Benefits of Go-based Tests

1. **Type Safety** - Compile-time error checking
2. **Better IDE Support** - IntelliSense, debugging, refactoring
3. **Unified Language** - Same language as the main application
4. **Better Error Handling** - Structured error handling vs shell script error handling
5. **Cross-platform** - Works on Windows, macOS, and Linux
6. **Better Maintenance** - Easier to read, modify, and extend
7. **Integration** - Can import and test actual application code
8. **Parallel Testing** - Go's testing framework supports parallel execution
9. **Cluster Management** - Automated kind cluster creation and cleanup
