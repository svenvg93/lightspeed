package agent

import (
	"testing"
)

func TestGetAllNetworkInterfaces(t *testing.T) {
	agent := &Agent{}

	interfaces := agent.getAllNetworkInterfaces()

	if interfaces == nil {
		t.Skip("No network interfaces found or failed to get network info")
	}

	// Verify we can get interface information
	if len(interfaces) == 0 {
		t.Log("No network interfaces found")
		return
	}

	// Check that we have at least one interface with valid data
	foundValidInterface := false
	for _, iface := range interfaces {
		if iface.Name != "" {
			foundValidInterface = true
			t.Logf("Found interface: %s (speed: %d Mbps, virtual: %v)",
				iface.Name, iface.Speed, iface.IsVirtual)
		}
	}

	if !foundValidInterface {
		t.Error("No valid network interfaces found")
	}
}
