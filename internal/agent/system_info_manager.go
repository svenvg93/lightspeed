package agent

import (
	"log/slog"
	"sync"
	"time"
)

type SystemInfoManager struct {
	agent        *Agent
	ticker       *time.Ticker
	stopChan     chan struct{}
	refreshMutex sync.RWMutex
	interval     time.Duration
}

func NewSystemInfoManager(agent *Agent) *SystemInfoManager {
	interval := getSystemInfoRefreshInterval()

	return &SystemInfoManager{
		agent:    agent,
		stopChan: make(chan struct{}),
		interval: interval,
	}
}

func (sim *SystemInfoManager) Start() {
	sim.ticker = time.NewTicker(sim.interval)

	go func() {
		slog.Info("System info refresh manager started", "interval", sim.interval)

		for {
			select {
			case <-sim.ticker.C:
				sim.refreshSystemInfo()
			case <-sim.stopChan:
				slog.Debug("System info refresh manager stopped")
				return
			}
		}
	}()
}

func (sim *SystemInfoManager) Stop() {
	if sim.ticker != nil {
		sim.ticker.Stop()
	}
	close(sim.stopChan)
}

func (sim *SystemInfoManager) refreshSystemInfo() {
	sim.refreshMutex.Lock()
	defer sim.refreshMutex.Unlock()

	oldIP := sim.agent.systemInfo.PublicIP
	oldISP := sim.agent.systemInfo.ISP
	oldASN := sim.agent.systemInfo.ASN

	slog.Debug("Starting system info refresh")

	// Refresh IP info
	sim.agent.getIPInfo()

	// Check if anything changed
	changed := oldIP != sim.agent.systemInfo.PublicIP ||
		oldISP != sim.agent.systemInfo.ISP ||
		oldASN != sim.agent.systemInfo.ASN

	if changed {
		slog.Info("System info updated",
			"ip_changed", oldIP != sim.agent.systemInfo.PublicIP,
			"isp_changed", oldISP != sim.agent.systemInfo.ISP,
			"asn_changed", oldASN != sim.agent.systemInfo.ASN,
			"new_ip", sim.agent.systemInfo.PublicIP,
			"new_isp", sim.agent.systemInfo.ISP)

		// TODO: Push updated system info to hub when push functionality is implemented
		sim.pushSystemInfo()
	} else {
		slog.Debug("System info refresh completed - no changes detected")
	}
}

func (sim *SystemInfoManager) pushSystemInfo() {
	// This will be implemented when we add push functionality
	// For now, just log that we would push
	slog.Debug("Would push updated system info to hub")
}

// getSystemInfoRefreshInterval returns the configured refresh interval
// Defaults to 6 hours if not configured
func getSystemInfoRefreshInterval() time.Duration {
	const defaultInterval = 6 * time.Hour

	if intervalStr, exists := GetEnv("SYSTEM_INFO_REFRESH_INTERVAL"); exists {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			slog.Debug("Using configured system info refresh interval", "interval", interval)
			return interval
		} else {
			slog.Warn("Invalid SYSTEM_INFO_REFRESH_INTERVAL, using default",
				"configured", intervalStr, "error", err, "default", defaultInterval)
		}
	}

	slog.Debug("Using default system info refresh interval", "interval", defaultInterval)
	return defaultInterval
}
