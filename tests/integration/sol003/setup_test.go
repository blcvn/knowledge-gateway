//go:build integration

package sol003

import (
	"log"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// Option 1: Assume docker-compose already up (CI mode)
	if os.Getenv("INTEGRATION_ASSUME_UP") == "true" {
		os.Exit(m.Run())
	}

	// Option 2: Start docker-compose in test
	cmd := exec.Command("docker", "compose",
		"-f", "../../docker-compose.consolidated.yml",
		"up", "-d", "--wait")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("failed to start docker compose, continuing anyway: %v", err)
	}

	// Wait for services to be healthy
	// In a real test, you'd poll the gateway health endpoint
	time.Sleep(5 * time.Second)

	code := m.Run()

	// Teardown
	exec.Command("docker", "compose",
		"-f", "../../docker-compose.consolidated.yml",
		"down").Run()

	os.Exit(code)
}
