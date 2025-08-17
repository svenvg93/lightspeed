package hub

import (
	"beszel/internal/entities/system"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// SendMonitoringConfigToAgent sends unified monitoring configuration to an agent via WebSocket
func (h *Hub) SendMonitoringConfigToAgent(systemRecord *core.Record) error {
	// Get monitoring config from the monitoring_config collection
	monitoringConfigRecord, err := h.FindFirstRecordByFilter("monitoring_config", "system = {:system}", map[string]any{"system": systemRecord.Id})

	if err != nil {
		h.Logger().Debug("No monitoring config found for system, sending empty configuration", "system", systemRecord.Id, "err", err)
		// No monitoring config, send empty configuration
		return h.sendMonitoringConfigToSystem(systemRecord.Id, system.MonitoringConfig{})
	}

	// Build the monitoring configuration from the record fields
	monitoringConfig := system.MonitoringConfig{
		Enabled: struct {
			Ping      bool `json:"ping"`
			Dns       bool `json:"dns"`
			Http      bool `json:"http,omitempty"`
			Speedtest bool `json:"speedtest,omitempty"`
		}{
			Ping:      monitoringConfigRecord.Get("ping") != nil,
			Dns:       monitoringConfigRecord.Get("dns") != nil,
			Http:      monitoringConfigRecord.Get("http") != nil,
			Speedtest: monitoringConfigRecord.Get("speedtest") != nil,
		},
	}

	// Parse individual monitoring configurations
	if pingData := monitoringConfigRecord.Get("ping"); pingData != nil {
		if err := json.Unmarshal([]byte(fmt.Sprintf("%v", pingData)), &monitoringConfig.Ping); err != nil {
			h.Logger().Error("Failed to parse ping config", "system", systemRecord.Id, "err", err)
		}
	}

	if dnsData := monitoringConfigRecord.Get("dns"); dnsData != nil {
		if err := json.Unmarshal([]byte(fmt.Sprintf("%v", dnsData)), &monitoringConfig.Dns); err != nil {
			h.Logger().Error("Failed to parse DNS config", "system", systemRecord.Id, "err", err)
		}
	}

	if httpData := monitoringConfigRecord.Get("http"); httpData != nil {
		if err := json.Unmarshal([]byte(fmt.Sprintf("%v", httpData)), &monitoringConfig.Http); err != nil {
			h.Logger().Error("Failed to parse HTTP config", "system", systemRecord.Id, "err", err)
		}
	}

	if speedtestData := monitoringConfigRecord.Get("speedtest"); speedtestData != nil {
		if err := json.Unmarshal([]byte(fmt.Sprintf("%v", speedtestData)), &monitoringConfig.Speedtest); err != nil {
			h.Logger().Error("Failed to parse speedtest config", "system", systemRecord.Id, "err", err)
		}
	}

	return h.sendMonitoringConfigToSystem(systemRecord.Id, monitoringConfig)
}

// sendMonitoringConfigToSystem sends monitoring configuration to a specific system
func (h *Hub) sendMonitoringConfigToSystem(systemId string, config system.MonitoringConfig) error {
	// Find the system in the system manager
	if h.sm == nil {
		slog.Debug("System manager is nil", "system", systemId)
		return nil
	}

	// Get the system from the store
	system, exists := h.sm.GetSystem(systemId)
	if !exists || system == nil {
		slog.Debug("System not found in manager", "system", systemId)
		return nil
	}

	// Send config via WebSocket if available
	if system.WsConn != nil && system.WsConn.IsConnected() {
		// Create versioned configuration structure
		versionedConfig := map[string]interface{}{
			"config":  config,
			"version": h.getNextConfigVersion(systemId),
		}

		err := system.WsConn.SendMonitoringConfig(versionedConfig)
		if err != nil {
			slog.Error("Failed to send monitoring config via WebSocket", "system", systemId, "err", err)
		} else {
			slog.Debug("Successfully sent monitoring config via WebSocket", "system", systemId, "version", versionedConfig["version"])
		}
		return err
	}

	return nil
}

// getNextConfigVersion generates the next configuration version for a system
func (h *Hub) getNextConfigVersion(systemId string) int64 {
	// Use Unix timestamp (seconds) for more reasonable version numbers
	// In a production environment, you might want to use a more sophisticated versioning system
	return time.Now().Unix()
}

// onSystemRecordUpdate handles system record updates to detect monitoring config changes
func (h *Hub) onSystemRecordUpdate(e *core.RecordEvent) error {
	h.Logger().Debug("System record update detected", "system", e.Record.Id)

	// Only send configuration on startup (first time)
	if !h.sm.HasConfigBeenSent(e.Record.Id) {
		h.Logger().Debug("Sending monitoring config on startup", "system", e.Record.Id)

		if err := h.SendMonitoringConfigToAgent(e.Record); err != nil {
			h.Logger().Error("Failed to send monitoring config on startup", "system", e.Record.Id, "err", err)
		} else {
			h.Logger().Debug("Successfully sent monitoring config on startup", "system", e.Record.Id)
			// Mark that we've sent the configuration to this system
			h.sm.MarkConfigAsSent(e.Record.Id)
		}
	} else {
		h.Logger().Debug("Monitoring config already sent, skipping (agent restart required for changes)", "system", e.Record.Id)
	}

	return e.Next()
}
