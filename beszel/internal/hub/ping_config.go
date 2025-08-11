package hub

import (
	"beszel/internal/entities/system"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// PingConfig represents the ping configuration for a system
type PingConfig struct {
	Targets  []system.PingTarget `json:"targets"`
	Interval string              `json:"interval"` // Cron expression (e.g., "*/30 * * * * *" for every 30 seconds)
}

// SendPingConfigToAgent sends ping configuration to an agent via WebSocket
func (h *Hub) SendPingConfigToAgent(systemRecord *core.Record) error {

	// Get ping config from system record
	pingConfigData := systemRecord.Get("ping_config")

	if pingConfigData == nil {

		// No ping config, send empty configuration
		return h.sendPingConfigToSystem(systemRecord.Id, PingConfig{})
	}

	var pingConfig PingConfig

	// Handle different data types that PocketBase might return
	switch v := pingConfigData.(type) {
	case types.JSONRaw:

		if err := json.Unmarshal([]byte(v), &pingConfig); err != nil {
			slog.Error("Failed to unmarshal ping config from JSONRaw", "system", systemRecord.Id, "err", err)
			return err
		}
	case []byte:

		if err := json.Unmarshal(v, &pingConfig); err != nil {
			slog.Error("Failed to unmarshal ping config from []byte", "system", systemRecord.Id, "err", err)
			return err
		}
	case string:

		if err := json.Unmarshal([]byte(v), &pingConfig); err != nil {
			slog.Error("Failed to unmarshal ping config from string", "system", systemRecord.Id, "err", err)
			return err
		}
	case map[string]interface{}:

		// Re-marshal and unmarshal to convert to proper struct
		configBytes, err := json.Marshal(v)
		if err != nil {
			slog.Error("Failed to marshal ping config map", "system", systemRecord.Id, "err", err)
			return err
		}
		if err := json.Unmarshal(configBytes, &pingConfig); err != nil {
			slog.Error("Failed to unmarshal ping config from map", "system", systemRecord.Id, "err", err)
			return err
		}
	default:
		slog.Error("Invalid ping config type", "system", systemRecord.Id, "type", fmt.Sprintf("%T", v), "value", v)
		return nil
	}

	return h.sendPingConfigToSystem(systemRecord.Id, pingConfig)
}

// sendPingConfigToSystem sends ping configuration to a specific system
func (h *Hub) sendPingConfigToSystem(systemId string, config PingConfig) error {

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

		err := system.WsConn.SendPingConfig(config)
		if err != nil {
			slog.Error("Failed to send ping config via WebSocket", "system", systemId, "err", err)
		} else {

		}
		return err
	}

	return nil
}

// onSystemRecordUpdate handles system record updates to detect ping config changes
// Note: We only send ping config at startup, not on every update
func (h *Hub) onSystemRecordUpdate(e *core.RecordEvent) error {

	return e.Next()
}
