.PHONY: build test test-integration test-e2e test-deploy clean docker-build docker-push helm-package helm-install helm-uninstall

# Variables
IMAGE_NAME := cert-manager-notifier
IMAGE_TAG := latest
REGISTRY := 
NAMESPACE := default

# Build the application
build:
	go build -o bin/cert-manager-notifier cmd/main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Build Docker image
docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Push Docker image
docker-push:
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

# Load Docker image into Docker Desktop Kubernetes
docker-load:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	docker save $(IMAGE_NAME):$(IMAGE_TAG) | docker load

# Package Helm chart
helm-package:
	helm package helm/cert-manager-notifier

# Install Helm chart
helm-install:
	helm install cert-manager-notifier helm/cert-manager-notifier \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--set image.repository=$(IMAGE_NAME) \
		--set image.tag=$(IMAGE_TAG) \
		--set config.webhookUrls="https://httpbin.org/post"

# Upgrade Helm chart
helm-upgrade:
	helm upgrade cert-manager-notifier helm/cert-manager-notifier \
		--namespace $(NAMESPACE) \
		--set image.repository=$(IMAGE_NAME) \
		--set image.tag=$(IMAGE_TAG) \
		--set config.webhookUrls="https://httpbin.org/post"

# Uninstall Helm chart
helm-uninstall:
	helm uninstall cert-manager-notifier --namespace $(NAMESPACE)

# Deploy to local Kubernetes (Docker Desktop)
deploy-local: docker-build helm-upgrade

# Full deployment pipeline
deploy: docker-build helm-install

# Show Helm chart values
helm-values:
	helm show values helm/cert-manager-notifier

# Lint Helm chart
helm-lint:
	helm lint helm/cert-manager-notifier

# Template Helm chart
helm-template:
	helm template cert-manager-notifier helm/cert-manager-notifier \
		--set config.webhookUrls="https://httpbin.org/post"

# Run locally for development
run-local:
	export WEBHOOK_URLS="https://httpbin.org/post" && \
	export CHECK_INTERVAL="30s" && \
	export LOG_LEVEL="debug" && \
	go run cmd/main.go

# Run integration tests  
test-integration:
	cd test && go test -v ./integration

# Run end-to-end tests
test-e2e:
	cd test && go test -v ./e2e -timeout 20m

# Test deployment to local Kubernetes
test-deploy:
	@echo "Testing deployment to local Kubernetes..."
	make docker-build
	make helm-install
	@echo "Waiting for deployment to be ready..."
	kubectl wait --for=condition=available --timeout=300s deployment/cert-manager-notifier -n $(NAMESPACE)
	@echo "Deployment test completed successfully!"

# Show help
help:
	@echo "Available targets:"
	@echo "  build              - Build the application"
	@echo "  test               - Run unit tests"
	@echo "  test-coverage      - Run tests with coverage"
	@echo "  test-integration   - Run integration tests"
	@echo "  test-e2e           - Run end-to-end tests"
	@echo "  test-deploy        - Test deployment to local Kubernetes"
	@echo "  clean              - Clean build artifacts"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-push        - Push Docker image"
	@echo "  docker-load        - Load Docker image into Docker Desktop"
	@echo "  helm-package       - Package Helm chart"
	@echo "  helm-install       - Install Helm chart"
	@echo "  helm-upgrade       - Upgrade Helm chart"
	@echo "  helm-uninstall     - Uninstall Helm chart"
	@echo "  deploy-local       - Deploy to local Kubernetes (Docker Desktop)"
	@echo "  deploy             - Full deployment pipeline"
	@echo "  helm-values        - Show Helm chart values"
	@echo "  helm-lint          - Lint Helm chart"
	@echo "  helm-template      - Template Helm chart"
	@echo "  run-local          - Run locally for development"
	@echo "  help               - Show this help"
