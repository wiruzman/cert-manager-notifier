package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	var (
		testType = flag.String("type", "all", "Type of tests to run: unit, integration, e2e, or all")
		skipE2E  = flag.Bool("skip-e2e", false, "Skip E2E tests")
		skipCleanup = flag.Bool("skip-cleanup", false, "Skip cleanup after tests")
		useKind  = flag.Bool("use-kind", false, "Use kind cluster for E2E tests")
		verbose  = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	// Set environment variables based on flags
	if *skipE2E {
		os.Setenv("SKIP_E2E_TESTS", "true")
	}
	if *skipCleanup {
		os.Setenv("SKIP_CLEANUP", "true")
	}
	if *useKind {
		os.Setenv("USE_KIND_CLUSTER", "true")
	}

	// Get project root
	projectRoot, err := filepath.Abs("..")
	if err != nil {
		log.Fatalf("Failed to get project root: %v", err)
	}

	fmt.Printf("🧪 Running tests in %s\n", projectRoot)

	// Change to project root
	if err := os.Chdir(projectRoot); err != nil {
		log.Fatalf("Failed to change to project root: %v", err)
	}

	success := true

	// Run unit tests
	if *testType == "all" || *testType == "unit" {
		fmt.Println("\n📋 Running unit tests...")
		if !runUnitTests(*verbose) {
			success = false
		}
	}

	// Run integration tests
	if *testType == "all" || *testType == "integration" {
		fmt.Println("\n🔗 Running integration tests...")
		if !runIntegrationTests(*verbose) {
			success = false
		}
	}

	// Run E2E tests
	if *testType == "all" || *testType == "e2e" {
		if !*skipE2E {
			fmt.Println("\n🚀 Running E2E tests...")
			if !runE2ETests(*verbose) {
				success = false
			}
		} else {
			fmt.Println("\n⏭️  Skipping E2E tests")
		}
	}

	if success {
		fmt.Println("\n🎉 All tests passed!")
		os.Exit(0)
	} else {
		fmt.Println("\n❌ Some tests failed!")
		os.Exit(1)
	}
}

func runUnitTests(verbose bool) bool {
	fmt.Println("Running Go unit tests...")
	
	args := []string{"test", "./..."}
	if verbose {
		args = append(args, "-v")
	}
	args = append(args, "-race", "-coverprofile=coverage.out")
	
	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	
	if verbose || err != nil {
		fmt.Printf("Output:\n%s\n", output)
	}
	
	if err != nil {
		fmt.Printf("❌ Unit tests failed: %v\n", err)
		return false
	}
	
	fmt.Println("✅ Unit tests passed")
	return true
}

func runIntegrationTests(verbose bool) bool {
	fmt.Println("Running integration tests...")
	
	// Change to test directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	if err := os.Chdir("test"); err != nil {
		fmt.Printf("❌ Failed to change to test directory: %v\n", err)
		return false
	}
	
	args := []string{"test", "-run", "TestWebhookIntegration|TestApplicationBuild|TestHelmLint"}
	if verbose {
		args = append(args, "-v")
	}
	
	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	
	if verbose || err != nil {
		fmt.Printf("Output:\n%s\n", output)
	}
	
	if err != nil {
		fmt.Printf("❌ Integration tests failed: %v\n", err)
		return false
	}
	
	fmt.Println("✅ Integration tests passed")
	return true
}

func runE2ETests(verbose bool) bool {
	fmt.Println("Running E2E tests...")
	
	// Check prerequisites
	if !checkPrerequisites() {
		fmt.Println("⏭️  Skipping E2E tests due to missing prerequisites")
		return true
	}
	
	// Change to test directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	if err := os.Chdir("test"); err != nil {
		fmt.Printf("❌ Failed to change to test directory: %v\n", err)
		return false
	}
	
	args := []string{"test", "-run", "TestE2E", "-timeout", "20m"}
	if verbose {
		args = append(args, "-v")
	}
	
	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	
	if verbose || err != nil {
		fmt.Printf("Output:\n%s\n", output)
	}
	
	if err != nil {
		fmt.Printf("❌ E2E tests failed: %v\n", err)
		return false
	}
	
	fmt.Println("✅ E2E tests passed")
	return true
}

func checkPrerequisites() bool {
	required := []string{"kubectl", "helm", "docker"}
	missing := []string{}
	
	// Add kind to required tools if USE_KIND_CLUSTER is set
	if os.Getenv("USE_KIND_CLUSTER") == "true" {
		required = append(required, "kind")
	}
	
	for _, tool := range required {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	
	if len(missing) > 0 {
		fmt.Printf("⚠️  Missing required tools: %s\n", strings.Join(missing, ", "))
		return false
	}
	
	// Check Kubernetes connectivity only if not using kind
	if os.Getenv("USE_KIND_CLUSTER") != "true" {
		cmd := exec.Command("kubectl", "cluster-info")
		if err := cmd.Run(); err != nil {
			fmt.Println("⚠️  Cannot connect to Kubernetes cluster")
			return false
		}
	}
	
	return true
}
