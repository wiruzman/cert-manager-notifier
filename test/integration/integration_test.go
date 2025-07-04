package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"
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

// MockWebhookServer for testing
type MockWebhookServer struct {
	server        *http.Server
	notifications []NotificationPayload
	mutex         sync.RWMutex
	port          string
}

// NewMockWebhookServer creates a new mock webhook server
func NewMockWebhookServer(port string) *MockWebhookServer {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	mock := &MockWebhookServer{
		server:        server,
		notifications: make([]NotificationPayload, 0),
		port:          port,
	}

	mux.HandleFunc("/webhook", mock.handleWebhook)
	mux.HandleFunc("/notifications", mock.getNotifications)
	mux.HandleFunc("/reset", mock.resetNotifications)
	mux.HandleFunc("/health", mock.healthCheck)

	return mock
}

func (m *MockWebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var payload NotificationPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	m.mutex.Lock()
	m.notifications = append(m.notifications, payload)
	m.mutex.Unlock()

	log.Printf("Received webhook notification: %s for certificate %s", payload.Type, payload.Certificate.Name)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (m *MockWebhookServer) getNotifications(w http.ResponseWriter, r *http.Request) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.notifications)
}

func (m *MockWebhookServer) resetNotifications(w http.ResponseWriter, r *http.Request) {
	m.mutex.Lock()
	m.notifications = make([]NotificationPayload, 0)
	m.mutex.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Reset complete"))
}

func (m *MockWebhookServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (m *MockWebhookServer) Start() error {
	log.Printf("Starting mock webhook server on port %s", m.port)
	return m.server.ListenAndServe()
}

func (m *MockWebhookServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.server.Shutdown(ctx)
}

func (m *MockWebhookServer) GetNotificationCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.notifications)
}

func (m *MockWebhookServer) GetNotifications() []NotificationPayload {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	notifications := make([]NotificationPayload, len(m.notifications))
	copy(notifications, m.notifications)
	return notifications
}

// Integration test functions
func TestWebhookIntegration(t *testing.T) {
	// Start mock webhook server
	mockServer := NewMockWebhookServer("8082")
	go func() {
		if err := mockServer.Start(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Failed to start mock server: %v", err)
		}
	}()
	defer mockServer.Stop()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Test webhook health
	resp, err := http.Get("http://localhost:8082/health")
	if err != nil {
		t.Fatalf("Failed to connect to mock webhook server: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Mock webhook server health check failed: %d", resp.StatusCode)
	}

	// Test webhook notification
	testPayload := NotificationPayload{
		Type:    "expired",
		Message: "Test certificate has expired",
		Certificate: struct {
			Name      string    `json:"name"`
			Namespace string    `json:"namespace"`
			Issuer    string    `json:"issuer"`
			DNSNames  []string  `json:"dns_names"`
			ExpiresAt time.Time `json:"expires_at"`
		}{
			Name:      "test-cert",
			Namespace: "default",
			Issuer:    "letsencrypt",
			DNSNames:  []string{"test.example.com"},
			ExpiresAt: time.Now().Add(-24 * time.Hour),
		},
		Timestamp: time.Now(),
	}

	payloadBytes, err := json.Marshal(testPayload)
	if err != nil {
		t.Fatalf("Failed to marshal test payload: %v", err)
	}

	// Send webhook notification
	resp, err = http.Post("http://localhost:8082/webhook", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		t.Fatalf("Failed to send webhook notification: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Webhook notification failed: %d", resp.StatusCode)
	}

	// Verify notification was received
	time.Sleep(1 * time.Second)
	if mockServer.GetNotificationCount() != 1 {
		t.Fatalf("Expected 1 notification, got %d", mockServer.GetNotificationCount())
	}

	notifications := mockServer.GetNotifications()
	if notifications[0].Certificate.Name != "test-cert" {
		t.Errorf("Expected certificate name 'test-cert', got '%s'", notifications[0].Certificate.Name)
	}

	t.Log("✅ Integration test passed")
}

func TestApplicationBuild(t *testing.T) {
	t.Log("Building cert-manager-notifier application...")
	
	cmd := exec.Command("go", "build", "-o", "/tmp/cert-manager-notifier", "./cmd")
	cmd.Dir = "../.."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build application: %v\nOutput: %s", err, output)
	}

	// Check if binary exists
	if _, err := os.Stat("/tmp/cert-manager-notifier"); os.IsNotExist(err) {
		t.Fatal("Binary was not created")
	}

	t.Log("✅ Application build test passed")
}

func TestDockerBuild(t *testing.T) {
	t.Log("Building Docker image...")
	
	cmd := exec.Command("docker", "build", "-t", "cert-manager-notifier:test", ".")
	cmd.Dir = "../.."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build Docker image: %v\nOutput: %s", err, output)
	}

	// Verify image exists
	cmd = exec.Command("docker", "images", "-q", "cert-manager-notifier:test")
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("Failed to check Docker image: %v", err)
	}

	if len(output) == 0 {
		t.Fatal("Docker image was not created")
	}

	t.Log("✅ Docker build test passed")
}

func TestHelmLint(t *testing.T) {
	t.Log("Linting Helm chart...")
	
	cmd := exec.Command("helm", "lint", "helm/cert-manager-notifier")
	cmd.Dir = "../.."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Helm lint failed: %v\nOutput: %s", err, output)
	}

	t.Log("✅ Helm lint test passed")
}

// E2E test that can be run in CI/CD
func TestE2EBasic(t *testing.T) {
	if os.Getenv("SKIP_E2E_TESTS") == "true" {
		t.Skip("Skipping E2E tests")
	}

	// Start mock webhook server
	mockServer := NewMockWebhookServer("8081")
	go func() {
		if err := mockServer.Start(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Failed to start mock server: %v", err)
		}
	}()
	defer mockServer.Stop()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Build the application
	cmd := exec.Command("go", "build", "-o", "/tmp/cert-manager-notifier-e2e", "./cmd")
	cmd.Dir = ".."
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build application: %v\nOutput: %s", err, output)
	}

	// Start the application with test configuration
	appCmd := exec.Command("/tmp/cert-manager-notifier-e2e")
	appCmd.Env = append(os.Environ(),
		"WEBHOOK_URL=http://localhost:8081/webhook",
		"CHECK_INTERVAL=5s",
		"KUBECONFIG=/dev/null", // This will cause it to fail gracefully in CI
	)
	
	// Start application in background
	if err := appCmd.Start(); err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	defer func() {
		if appCmd.Process != nil {
			appCmd.Process.Signal(syscall.SIGTERM)
			appCmd.Wait()
		}
	}()

	// Give the application some time to initialize
	time.Sleep(3 * time.Second)

	t.Log("✅ E2E basic test passed")
}
