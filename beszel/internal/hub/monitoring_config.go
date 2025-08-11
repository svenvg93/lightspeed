package hub

import (
	"beszel/internal/entities/system"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// SendMonitoringConfigToAgent sends unified monitoring configuration to an agent via WebSocket
func (h *Hub) SendMonitoringConfigToAgent(systemRecord *core.Record) error {
	// Get monitoring config from system record
	monitoringConfigData := systemRecord.Get("monitoring_config")

	if monitoringConfigData == nil {
		// No monitoring config, send empty configuration
		return h.sendMonitoringConfigToSystem(systemRecord.Id, system.MonitoringConfig{})
	}

	var monitoringConfig system.MonitoringConfig

	// Handle different data types that PocketBase might return
	switch v := monitoringConfigData.(type) {
	case types.JSONRaw:
		if err := json.Unmarshal([]byte(v), &monitoringConfig); err != nil {
			slog.Error("Failed to unmarshal monitoring config from JSONRaw", "system", systemRecord.Id, "err", err)
			return err
		}
	case []byte:
		if err := json.Unmarshal(v, &monitoringConfig); err != nil {
			slog.Error("Failed to unmarshal monitoring config from []byte", "system", systemRecord.Id, "err", err)
			return err
		}
	case string:
		if err := json.Unmarshal([]byte(v), &monitoringConfig); err != nil {
			slog.Error("Failed to unmarshal monitoring config from string", "system", systemRecord.Id, "err", err)
			return err
		}
	case map[string]interface{}:
		// Re-marshal and unmarshal to convert to proper struct
		configBytes, err := json.Marshal(v)
		if err != nil {
			slog.Error("Failed to marshal monitoring config map", "system", systemRecord.Id, "err", err)
			return err
		}
		if err := json.Unmarshal(configBytes, &monitoringConfig); err != nil {
			slog.Error("Failed to unmarshal monitoring config from map", "system", systemRecord.Id, "err", err)
			return err
		}
	default:
		slog.Error("Invalid monitoring config type", "system", systemRecord.Id, "type", fmt.Sprintf("%T", v), "value", v)
		return nil
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
		err := system.WsConn.SendMonitoringConfig(config)
		if err != nil {
			slog.Error("Failed to send monitoring config via WebSocket", "system", systemId, "err", err)
		} else {
			slog.Debug("Successfully sent monitoring config via WebSocket", "system", systemId)
		}
		return err
	}

	return nil
}

// onSystemRecordUpdate handles system record updates to detect monitoring config changes
func (h *Hub) onSystemRecordUpdate(e *core.RecordEvent) error {
	h.Logger().Debug("System record update detected", "system", e.Record.Id)

	// Send monitoring configuration update
	if err := h.SendMonitoringConfigToAgent(e.Record); err != nil {
		h.Logger().Error("Failed to send monitoring config update", "system", e.Record.Id, "err", err)
	} else {
		h.Logger().Debug("Successfully sent monitoring config update", "system", e.Record.Id)
	}

	return nil
}
