package embed

import (
	"fmt"
	"os"

	"github.com/vnp-community/vnp-memory/apps/OpenViking/internal/config"
)

// SetEnvHelper sets an environment variable temporarily or updates the environment for the sub-services.
func SetEnvHelper(key, value string) error {
	return os.Setenv(key, value)
}

// PrepareServiceEnv injects port config into env before running the service main func
func PrepareServiceEnv(cfg *config.Config, serviceName string, port int) {
	// Usually services read from specific env vars like GRPC_PORT or PORT.
	// For demonstration, we just set a generic GRPC_PORT.
	os.Setenv("GRPC_PORT", fmt.Sprintf("%d", port))
}
