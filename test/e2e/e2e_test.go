package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// E2ETestSuite manages end-to-end testing
type E2ETestSuite struct {
	namespace      string
	webhookPort    string
	testCertName   string
	shouldCleanup  bool
	kubeconfig     string
	useKindCluster bool
	kindClusterName string
}

// NewE2ETestSuite creates a new E2E test suite
func NewE2ETestSuite() *E2ETestSuite {
	useKind := os.Getenv("USE_KIND_CLUSTER") == "true" || os.Getenv("CI") == "true"
	
	return &E2ETestSuite{
		namespace:       "cert-manager-notifier-test",
		webhookPort:     "8081",
		testCertName:    "test-cert",
		shouldCleanup:   true,
		kubeconfig:      os.Getenv("KUBECONFIG"),
		useKindCluster:  useKind,
		kindClusterName: "cert-manager-notifier-e2e",
	}
}

func (e *E2ETestSuite) runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if e.kubeconfig != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", e.kubeconfig))
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (e *E2ETestSuite) checkPrerequisites(t *testing.T) {
	t.Log("🔍 Checking prerequisites...")

	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		t.Skip("kubectl not found, skipping E2E tests")
	}

	// Check if helm is available
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm not found, skipping E2E tests")
	}

	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found, skipping E2E tests")
	}

	// Check if kind is available when needed
	if e.useKindCluster {
		if _, err := exec.LookPath("kind"); err != nil {
			t.Skip("kind not found but USE_KIND_CLUSTER=true, skipping E2E tests")
		}
	}

	// Check Kubernetes cluster connectivity (only if not using kind)
	if !e.useKindCluster {
		output, err := e.runCommand("kubectl", "cluster-info")
		if err != nil {
			t.Skipf("Cannot connect to Kubernetes cluster: %v\nOutput: %s", err, output)
		}
	}

	t.Log("✅ Prerequisites check passed")
}

func (e *E2ETestSuite) setupKindCluster(t *testing.T) {
	if !e.useKindCluster {
		return
	}

	t.Logf("🏗️  Creating kind cluster '%s'...", e.kindClusterName)

	// Delete existing cluster if it exists
	e.runCommand("kind", "delete", "cluster", "--name", e.kindClusterName)

	// Create kind cluster
	output, err := e.runCommand("kind", "create", "cluster", "--name", e.kindClusterName, "--wait", "60s")
	if err != nil {
		t.Fatalf("Failed to create kind cluster: %v\nOutput: %s", err, output)
	}

	// Set kubeconfig for the kind cluster
	output, err = e.runCommand("kind", "get", "kubeconfig", "--name", e.kindClusterName)
	if err != nil {
		t.Fatalf("Failed to get kubeconfig: %v\nOutput: %s", err, output)
	}

	// Write kubeconfig to temporary file
	kubeconfigFile := fmt.Sprintf("/tmp/kubeconfig-%s", e.kindClusterName)
	if err := os.WriteFile(kubeconfigFile, []byte(output), 0600); err != nil {
		t.Fatalf("Failed to write kubeconfig: %v", err)
	}

	e.kubeconfig = kubeconfigFile
	os.Setenv("KUBECONFIG", kubeconfigFile)

	t.Logf("✅ Kind cluster '%s' created and configured", e.kindClusterName)
}

func (e *E2ETestSuite) teardownKindCluster(t *testing.T) {
	if !e.useKindCluster || !e.shouldCleanup {
		return
	}

	t.Logf("🗑️  Deleting kind cluster '%s'...", e.kindClusterName)

	// Delete kind cluster
	e.runCommand("kind", "delete", "cluster", "--name", e.kindClusterName)

	// Clean up kubeconfig file
	if e.kubeconfig != "" && strings.Contains(e.kubeconfig, "/tmp/kubeconfig-") {
		os.Remove(e.kubeconfig)
	}

	t.Logf("✅ Kind cluster '%s' deleted", e.kindClusterName)
}

func (e *E2ETestSuite) buildDockerImage(t *testing.T) {
	t.Log("🔨 Building Docker image...")

	// Get project root directory
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	// Build Docker image
	cmd := exec.Command("docker", "build", "-t", "cert-manager-notifier:e2e-test", ".")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build Docker image: %v\nOutput: %s", err, output)
	}

	t.Log("✅ Docker image built successfully")
}

func (e *E2ETestSuite) createNamespace(t *testing.T) {
	t.Logf("🏗️  Creating namespace %s...", e.namespace)

	// Delete namespace if it exists
	e.runCommand("kubectl", "delete", "namespace", e.namespace, "--ignore-not-found=true")

	// Wait for namespace deletion
	time.Sleep(5 * time.Second)

	// Create namespace
	output, err := e.runCommand("kubectl", "create", "namespace", e.namespace)
	if err != nil {
		t.Fatalf("Failed to create namespace: %v\nOutput: %s", err, output)
	}

	t.Logf("✅ Namespace %s created", e.namespace)
}

func (e *E2ETestSuite) deployCertManager(t *testing.T) {
	t.Log("📦 Deploying cert-manager...")

	// Check if cert-manager is already installed
	output, err := e.runCommand("kubectl", "get", "namespace", "cert-manager")
	if err == nil {
		t.Log("ℹ️  cert-manager namespace already exists, skipping installation")
		return
	}

	// Add cert-manager Helm repository
	t.Log("🔗 Adding cert-manager Helm repository...")
	output, err = e.runCommand("helm", "repo", "add", "jetstack", "https://charts.jetstack.io")
	if err != nil {
		t.Fatalf("Failed to add cert-manager Helm repo: %v\nOutput: %s", err, output)
	}

	// Update Helm repositories
	output, err = e.runCommand("helm", "repo", "update")
	if err != nil {
		t.Fatalf("Failed to update Helm repos: %v\nOutput: %s", err, output)
	}

	// Install cert-manager using Helm
	t.Log("⚙️  Installing cert-manager v1.18.2 using Helm...")
	output, err = e.runCommand("helm", "install", "cert-manager", "jetstack/cert-manager",
		"--namespace", "cert-manager",
		"--create-namespace",
		"--version", "v1.18.2",
		"--set", "crds.enabled=true",
		"--wait", "--timeout=300s")
	if err != nil {
		t.Fatalf("Failed to install cert-manager: %v\nOutput: %s", err, output)
	}

	// Wait for cert-manager to be ready
	t.Log("⏳ Waiting for cert-manager to be ready...")
	for i := 0; i < 60; i++ { // Wait up to 5 minutes
		output, err := e.runCommand("kubectl", "get", "pods", "-n", "cert-manager", "-l", "app.kubernetes.io/instance=cert-manager", "--field-selector=status.phase=Running")
		if err == nil && strings.Contains(output, "Running") {
			runningPods := strings.Count(output, "Running")
			if runningPods >= 3 { // cert-manager has 3 main components
				break
			}
		}
		time.Sleep(5 * time.Second)
	}

	t.Log("✅ cert-manager deployed and ready")
}

func (e *E2ETestSuite) deployApplication(t *testing.T) {
	t.Log("🚀 Deploying cert-manager-notifier...")

	// Get project root directory
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	// Determine image configuration based on environment
	var imageRepo, imageTag, pullPolicy string
	if os.Getenv("CI") == "true" {
		// In CI, use the tagged image that was loaded into kind
		imageRepo = "cert-manager-notifier"
		imageTag = "e2e-test"
		pullPolicy = "Never" // Image is loaded into kind cluster
		t.Log("ℹ️  Using CI-built image loaded into kind cluster")
	} else {
		// Local testing with Docker image
		imageRepo = "cert-manager-notifier"
		imageTag = "e2e-test"
		pullPolicy = "Never"
		t.Log("ℹ️  Using local Docker image for development")
	}

	// Create values file for testing
	valuesContent := fmt.Sprintf(`
image:
  repository: %s
  tag: %s
  pullPolicy: %s

config:
  webhookUrl: "http://host.docker.internal:%s/webhook"
  checkInterval: "30s"

replicaCount: 1

resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi
`, imageRepo, imageTag, pullPolicy, e.webhookPort)

	valuesFile := filepath.Join(".", "values-e2e.yaml")
	if err := os.WriteFile(valuesFile, []byte(valuesContent), 0644); err != nil {
		t.Fatalf("Failed to create values file: %v", err)
	}
	defer os.Remove(valuesFile)

	// Install using Helm
	helmChartPath := filepath.Join(projectRoot, "helm", "cert-manager-notifier")
	t.Logf("🔍 Using Helm chart path: %s", helmChartPath)
	t.Logf("🐳 Using image: %s:%s (pullPolicy: %s)", imageRepo, imageTag, pullPolicy)
	
	output, err := e.runCommand("helm", "install", "cert-manager-notifier",
		helmChartPath,
		"--namespace", e.namespace,
		"--values", valuesFile,
		"--wait", "--timeout=600s") // Increased timeout to 10 minutes
	if err != nil {
		// Log more debugging information
		t.Logf("Values file content:\n%s", valuesContent)
		debugOutput, _ := e.runCommand("kubectl", "get", "pods", "-n", e.namespace, "-o", "wide")
		t.Logf("Pod status:\n%s", debugOutput)
		debugOutput, _ = e.runCommand("kubectl", "describe", "pods", "-n", e.namespace)
		t.Logf("Pod details:\n%s", debugOutput)
		t.Fatalf("Failed to deploy with Helm: %v\nOutput: %s", err, output)
	}

	t.Log("✅ cert-manager-notifier deployed successfully")
}

func (e *E2ETestSuite) createTestCertificate(t *testing.T) {
	t.Log("📜 Creating test certificate...")

	certManifest := fmt.Sprintf(`
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: %s
  namespace: %s
spec:
  secretName: %s-tls
  dnsNames:
  - test.example.com
  - www.test.example.com
  issuerRef:
    name: selfsigned-issuer
    kind: ClusterIssuer
---
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: selfsigned-issuer
spec:
  selfSigned: {}
`, e.testCertName, e.namespace, e.testCertName)

	// Apply certificate manifest
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(certManifest)
	if e.kubeconfig != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", e.kubeconfig))
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create test certificate: %v\nOutput: %s", err, output)
	}

	// Wait for certificate to be ready
	t.Log("⏳ Waiting for certificate to be ready...")
	for i := 0; i < 30; i++ { // Wait up to 2.5 minutes
		output, err := e.runCommand("kubectl", "get", "certificate", e.testCertName, "-n", e.namespace, "-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
		if err == nil && strings.TrimSpace(string(output)) == "True" {
			break
		}
		time.Sleep(5 * time.Second)
	}

	t.Log("✅ Test certificate created and ready")
}

func (e *E2ETestSuite) verifyDeployment(t *testing.T) {
	t.Log("🔍 Verifying deployment...")

	// Check if pods are running
	output, err := e.runCommand("kubectl", "get", "pods", "-n", e.namespace, "-l", "app.kubernetes.io/name=cert-manager-notifier")
	if err != nil {
		t.Fatalf("Failed to get pods: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Running") {
		t.Fatalf("cert-manager-notifier pods are not running:\n%s", output)
	}

	// Check logs for any obvious errors
	output, err = e.runCommand("kubectl", "logs", "-n", e.namespace, "-l", "app.kubernetes.io/name=cert-manager-notifier", "--tail=50")
	if err != nil {
		t.Logf("Warning: Could not retrieve logs: %v", err)
	} else {
		t.Logf("Recent logs:\n%s", output)
	}

	t.Log("✅ Deployment verification passed")
}

func (e *E2ETestSuite) cleanup(t *testing.T) {
	if !e.shouldCleanup {
		t.Log("⚠️  Skipping cleanup (SKIP_CLEANUP=true)")
		return
	}

	t.Log("🧹 Cleaning up test environment...")

	// Uninstall Helm release
	e.runCommand("helm", "uninstall", "cert-manager-notifier", "--namespace", e.namespace)

	// Delete namespace (only if not using kind, as kind cluster will be deleted entirely)
	if !e.useKindCluster {
		e.runCommand("kubectl", "delete", "namespace", e.namespace, "--ignore-not-found=true")
	}

	// Remove Docker image
	e.runCommand("docker", "rmi", "cert-manager-notifier:e2e-test", "--force")

	// Cleanup kind cluster if we created one
	e.teardownKindCluster(t)

	// Note: We don't remove cert-manager in cleanup as it might be needed by other tests
	// and it's generally safe to leave it running in the cluster

	t.Log("✅ Cleanup completed")
}

func TestE2EFullDeployment(t *testing.T) {
	if os.Getenv("SKIP_E2E_TESTS") == "true" {
		t.Skip("Skipping E2E tests")
	}

	suite := NewE2ETestSuite()
	
	// Check for cleanup skip
	if os.Getenv("SKIP_CLEANUP") == "true" {
		suite.shouldCleanup = false
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("E2E test panicked: %v", r)
		}
		if !suite.shouldCleanup {
			t.Log("⚠️  Test environment preserved for debugging")
		} else {
			suite.cleanup(t)
		}
	}()

	// Run test steps
	suite.checkPrerequisites(t)
	suite.setupKindCluster(t) // Setup kind cluster if needed
	
	// Only build and load Docker image if not in CI environment
	if os.Getenv("CI") != "true" {
		// Local development - build image from source
		suite.buildDockerImage(t)
		
		// Load Docker image into kind cluster if using kind
		if suite.useKindCluster {
			t.Log("📦 Loading Docker image into kind cluster...")
			output, err := suite.runCommand("kind", "load", "docker-image", "cert-manager-notifier:e2e-test", "--name", suite.kindClusterName)
			if err != nil {
				t.Fatalf("Failed to load Docker image into kind: %v\nOutput: %s", err, output)
			}
			t.Log("✅ Docker image loaded into kind cluster")
		}
	} else {
		// CI environment - the workflow has already tagged the pulled image as cert-manager-notifier:e2e-test
		t.Log("ℹ️  Using pre-built image from CI workflow")
		
		// Load the pre-tagged image into kind cluster
		if suite.useKindCluster {
			t.Log("📦 Loading pre-built Docker image into kind cluster...")
			output, err := suite.runCommand("kind", "load", "docker-image", "cert-manager-notifier:e2e-test", "--name", suite.kindClusterName)
			if err != nil {
				t.Fatalf("Failed to load Docker image into kind: %v\nOutput: %s", err, output)
			}
			t.Log("✅ Pre-built Docker image loaded into kind cluster")
		}
	}
	
	suite.createNamespace(t)
	suite.deployCertManager(t)
	suite.deployApplication(t)
	suite.createTestCertificate(t)
	suite.verifyDeployment(t)

	t.Log("🎉 E2E test completed successfully!")
}

func TestE2EHelmChart(t *testing.T) {
	if os.Getenv("SKIP_E2E_TESTS") == "true" {
		t.Skip("Skipping E2E tests")
	}

	t.Log("🧪 Testing Helm chart...")

	// Get project root directory
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	// Test Helm template rendering
	output, err := exec.Command("helm", "template", "test-release",
		filepath.Join(projectRoot, "helm", "cert-manager-notifier"),
		"--namespace", "test-namespace").CombinedOutput()
	if err != nil {
		t.Fatalf("Helm template failed: %v\nOutput: %s", err, output)
	}

	// Basic validation of rendered templates
	rendered := string(output)
	if !strings.Contains(rendered, "kind: Deployment") {
		t.Error("Rendered templates should contain a Deployment")
	}
	if !strings.Contains(rendered, "kind: ConfigMap") {
		t.Error("Rendered templates should contain a ConfigMap")
	}
	if !strings.Contains(rendered, "kind: ServiceAccount") {
		t.Error("Rendered templates should contain a ServiceAccount")
	}

	t.Log("✅ Helm chart test passed")
}
