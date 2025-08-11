package hub

import (
	"beszel/internal/entities/system"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// DnsConfig represents the DNS configuration for a system
type DnsConfig struct {
	Targets  []system.DnsTarget `json:"targets"`
	Interval string             `json:"interval"` // Cron expression (e.g., "*/30 * * * * *" for every 30 seconds)
}

// SendDnsConfigToAgent sends DNS configuration to an agent via WebSocket
func (h *Hub) SendDnsConfigToAgent(systemRecord *core.Record) error {
	slog.Debug("SendDnsConfigToAgent called", "system", systemRecord.Id)

	// Get DNS config from system record
	dnsConfigData := systemRecord.Get("dns_config")
	slog.Debug("DNS config data from record", "system", systemRecord.Id, "dns_config", dnsConfigData)

	if dnsConfigData == nil {
		slog.Debug("No DNS config found, sending empty configuration", "system", systemRecord.Id)
		// No DNS config, send empty configuration
		return h.sendDnsConfigToSystem(systemRecord.Id, DnsConfig{})
	}

	var dnsConfig DnsConfig

	// Handle different data types that PocketBase might return
	switch v := dnsConfigData.(type) {
	case types.JSONRaw:

		if err := json.Unmarshal([]byte(v), &dnsConfig); err != nil {
			slog.Error("Failed to unmarshal DNS config from JSONRaw", "system", systemRecord.Id, "err", err)
			return err
		}
	case []byte:

		if err := json.Unmarshal(v, &dnsConfig); err != nil {
			slog.Error("Failed to unmarshal DNS config from []byte", "system", systemRecord.Id, "err", err)
			return err
		}
	case string:

		if err := json.Unmarshal([]byte(v), &dnsConfig); err != nil {
			slog.Error("Failed to unmarshal DNS config from string", "system", systemRecord.Id, "err", err)
			return err
		}
	case map[string]interface{}:

		// Re-marshal and unmarshal to convert to proper struct
		configBytes, err := json.Marshal(v)
		if err != nil {
			slog.Error("Failed to marshal DNS config map", "system", systemRecord.Id, "err", err)
			return err
		}
		if err := json.Unmarshal(configBytes, &dnsConfig); err != nil {
			slog.Error("Failed to unmarshal DNS config from map", "system", systemRecord.Id, "err", err)
			return err
		}
	default:
		slog.Error("Invalid DNS config type", "system", systemRecord.Id, "type", fmt.Sprintf("%T", v), "value", v)
		return nil
	}

	slog.Debug("DNS config parsed successfully", "system", systemRecord.Id, "targets", len(dnsConfig.Targets), "interval", dnsConfig.Interval)
	return h.sendDnsConfigToSystem(systemRecord.Id, dnsConfig)
}

// sendDnsConfigToSystem sends DNS configuration to a specific system
func (h *Hub) sendDnsConfigToSystem(systemId string, config DnsConfig) error {
	slog.Debug("sendDnsConfigToSystem called", "system", systemId, "targets", len(config.Targets))

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
		slog.Debug("WebSocket connection available, sending DNS config", "system", systemId)
		err := system.WsConn.SendDnsConfig(config)
		if err != nil {
			slog.Error("Failed to send DNS config via WebSocket", "system", systemId, "err", err)
		} else {
			slog.Debug("DNS config sent successfully", "system", systemId, "targets", len(config.Targets))
		}
		return err
	} else {
		slog.Debug("WebSocket connection not available", "system", systemId, "ws_conn_nil", system.WsConn == nil, "connected", system.WsConn != nil && system.WsConn.IsConnected())
	}

	return nil
}
