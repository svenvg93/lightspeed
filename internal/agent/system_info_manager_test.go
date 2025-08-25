package agent

import (
	"os"
	"testing"
	"time"
)

func TestSystemInfoManager_NewSystemInfoManager(t *testing.T) {
	agent := &Agent{}
	manager := NewSystemInfoManager(agent)

	if manager == nil {
		t.Fatal("Expected non-nil SystemInfoManager")
	}

	if manager.agent != agent {
		t.Error("Expected agent to be set correctly")
	}

	if manager.interval == 0 {
		t.Error("Expected interval to be set")
	}
}

func TestGetSystemInfoRefreshInterval(t *testing.T) {
	// Test default interval
	interval := getSystemInfoRefreshInterval()
	expectedDefault := 6 * time.Hour

	if interval != expectedDefault {
		t.Errorf("Expected default interval %v, got %v", expectedDefault, interval)
	}
}

func TestGetSystemInfoRefreshInterval_WithEnvVar(t *testing.T) {
	// Save original environment variable
	originalEnv := os.Getenv("BESZEL_AGENT_SYSTEM_INFO_REFRESH_INTERVAL")
	defer func() {
		if originalEnv != "" {
			os.Setenv("BESZEL_AGENT_SYSTEM_INFO_REFRESH_INTERVAL", originalEnv)
		} else {
			os.Unsetenv("BESZEL_AGENT_SYSTEM_INFO_REFRESH_INTERVAL")
		}
	}()

	// Test with custom interval
	os.Setenv("BESZEL_AGENT_SYSTEM_INFO_REFRESH_INTERVAL", "1h")
	interval := getSystemInfoRefreshInterval()
	expectedInterval := 1 * time.Hour

	if interval != expectedInterval {
		t.Errorf("Expected interval %v, got %v", expectedInterval, interval)
	}

	// Test with invalid interval (should fall back to default)
	os.Setenv("BESZEL_AGENT_SYSTEM_INFO_REFRESH_INTERVAL", "invalid")
	interval = getSystemInfoRefreshInterval()
	expectedDefault := 6 * time.Hour

	if interval != expectedDefault {
		t.Errorf("Expected default interval %v for invalid config, got %v", expectedDefault, interval)
	}
}

func TestSystemInfoManager_StartStop(t *testing.T) {
	agent := &Agent{}
	manager := NewSystemInfoManager(agent)

	// Start the manager
	manager.Start()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop the manager
	manager.Stop()

	// Give it a moment to stop
	time.Sleep(100 * time.Millisecond)
}
