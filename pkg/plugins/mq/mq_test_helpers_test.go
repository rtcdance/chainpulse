package mq

import (
	"os"
	"testing"
)

const mqIntegrationEnv = "CHAINPULSE_RUN_MQ_INTEGRATION"

func requireMQIntegration(t testing.TB) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping MQ integration test in short mode")
	}
	if os.Getenv(mqIntegrationEnv) != "1" {
		t.Skip("Skipping MQ integration test; set CHAINPULSE_RUN_MQ_INTEGRATION=1 to enable")
	}
}
