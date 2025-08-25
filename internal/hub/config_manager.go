package hub

import (
	"beszel/internal/entities/system"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ConfigurationManager provides optimized configuration management for the hub
type ConfigurationManager struct {
	hub             *Hub
	cache           sync.Map                    // Cache for configuration data by system ID
	versions        sync.Map                    // Track configuration versions by system ID
	pendingUpdates  sync.Map                    // Track pending configuration updates
	batchCh         chan ConfigurationUpdate    // Channel for batching configuration updates
	updateTicker    *time.Ticker               // Ticker for periodic batch processing
	mutex           sync.RWMutex               // Mutex for configuration operations
	
	// Configuration settings
	batchSize       int           // Maximum batch size for configuration updates
	batchTimeout    time.Duration // Timeout for batch processing
	cacheTimeout    time.Duration // Cache expiration timeout
}

// ConfigurationUpdate represents a pending configuration update
type ConfigurationUpdate struct {
	SystemID    string                    `json:"system_id"`
	Config      system.MonitoringConfig   `json:"config"`
	Version     int64                     `json:"version"`
	Hash        string                    `json:"hash"`
	Timestamp   time.Time                 `json:"timestamp"`
	Priority    int                       `json:"priority"` // 1=high, 2=normal, 3=low
}

// CachedConfiguration represents a cached configuration with metadata
type CachedConfiguration struct {
	Config      system.MonitoringConfig `json:"config"`
	Version     int64                   `json:"version"`
	Hash        string                  `json:"hash"`
	Timestamp   time.Time               `json:"timestamp"`
	SendCount   int                     `json:"send_count"`   // Track how many times sent
	LastSent    time.Time               `json:"last_sent"`    // Last time sent to agent
}

// NewConfigurationManager creates a new optimized configuration manager
func NewConfigurationManager(hub *Hub) *ConfigurationManager {
	cm := &ConfigurationManager{
		hub:          hub,
		batchSize:    50,                  // Process up to 50 updates per batch
		batchTimeout: 30 * time.Second,    // Process batches every 30 seconds
		cacheTimeout: 10 * time.Minute,    // Cache configurations for 10 minutes
		batchCh:      make(chan ConfigurationUpdate, 1000), // Buffer for 1000 updates
		updateTicker: time.NewTicker(30 * time.Second),
	}

	// Start batch processing goroutine
	go cm.processBatchUpdates()
	
	return cm
}

// GetConfiguration retrieves a cached configuration or loads it from database
func (cm *ConfigurationManager) GetConfiguration(systemID string) (*CachedConfiguration, error) {
	// Check cache first
	if cached, ok := cm.cache.Load(systemID); ok {
		cachedConfig := cached.(*CachedConfiguration)
		
		// Check if cache is still valid
		if time.Since(cachedConfig.Timestamp) < cm.cacheTimeout {
			return cachedConfig, nil
		}
		
		// Remove expired cache entry
		cm.cache.Delete(systemID)
	}

	// Load from database
	config, err := cm.loadConfigurationFromDatabase(systemID)
	if err != nil {
		return nil, err
	}

	// Cache the configuration
	cm.cache.Store(systemID, config)
	
	return config, nil
}

// loadConfigurationFromDatabase loads configuration from the monitoring_config collection
func (cm *ConfigurationManager) loadConfigurationFromDatabase(systemID string) (*CachedConfiguration, error) {
	monitoringConfigRecord, err := cm.hub.FindFirstRecordByFilter("monitoring_config", "system = {:system}", map[string]any{"system": systemID})
	
	var config system.MonitoringConfig
	
	if err != nil {
		// No monitoring config found, use empty configuration
		config = system.MonitoringConfig{}
	} else {
		// Build the monitoring configuration from the record fields
		config = system.MonitoringConfig{
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
			if err := json.Unmarshal([]byte(fmt.Sprintf("%v", pingData)), &config.Ping); err != nil {
				slog.Error("Failed to parse ping config", "system", systemID, "err", err)
			}
		}

		if dnsData := monitoringConfigRecord.Get("dns"); dnsData != nil {
			if err := json.Unmarshal([]byte(fmt.Sprintf("%v", dnsData)), &config.Dns); err != nil {
				slog.Error("Failed to parse DNS config", "system", systemID, "err", err)
			}
		}

		if httpData := monitoringConfigRecord.Get("http"); httpData != nil {
			if err := json.Unmarshal([]byte(fmt.Sprintf("%v", httpData)), &config.Http); err != nil {
				slog.Error("Failed to parse HTTP config", "system", systemID, "err", err)
			}
		}

		if speedtestData := monitoringConfigRecord.Get("speedtest"); speedtestData != nil {
			if err := json.Unmarshal([]byte(fmt.Sprintf("%v", speedtestData)), &config.Speedtest); err != nil {
				slog.Error("Failed to parse speedtest config", "system", systemID, "err", err)
			}
		}
	}

	version := cm.getNextConfigVersion(systemID)
	hash := cm.calculateConfigHash(config)

	return &CachedConfiguration{
		Config:    config,
		Version:   version,
		Hash:      hash,
		Timestamp: time.Now(),
		SendCount: 0,
	}, nil
}

// QueueConfigurationUpdate queues a configuration update for batch processing
func (cm *ConfigurationManager) QueueConfigurationUpdate(systemID string, config system.MonitoringConfig, priority int) {
	version := cm.getNextConfigVersion(systemID)
	hash := cm.calculateConfigHash(config)

	update := ConfigurationUpdate{
		SystemID:  systemID,
		Config:    config,
		Version:   version,
		Hash:      hash,
		Timestamp: time.Now(),
		Priority:  priority,
	}

	// Try to send to channel without blocking
	select {
	case cm.batchCh <- update:
		slog.Debug("Configuration update queued", "system", systemID, "version", version, "priority", priority)
	default:
		// Channel is full, process immediately for high priority updates
		if priority == 1 {
			go cm.processImmediateUpdate(update)
			slog.Warn("Configuration channel full, processing high priority update immediately", "system", systemID)
		} else {
			slog.Warn("Configuration update dropped - queue full", "system", systemID)
		}
	}
}

// SendConfigurationToAgent sends configuration to a specific agent immediately
func (cm *ConfigurationManager) SendConfigurationToAgent(systemID string, priority int) error {
	config, err := cm.GetConfiguration(systemID)
	if err != nil {
		return fmt.Errorf("failed to get configuration for system %s: %w", systemID, err)
	}

	return cm.sendConfigToSystem(systemID, config)
}

// SendConfigurationToAllAgents sends configuration updates to all connected agents
func (cm *ConfigurationManager) SendConfigurationToAllAgents() error {
	if cm.hub.sm == nil {
		return fmt.Errorf("system manager not initialized")
	}

	// Get all systems and queue configuration updates
	systems := cm.getAllConnectedSystems()
	
	for _, systemID := range systems {
		config, err := cm.GetConfiguration(systemID)
		if err != nil {
			slog.Error("Failed to get configuration for bulk update", "system", systemID, "err", err)
			continue
		}

		cm.QueueConfigurationUpdate(systemID, config.Config, 2) // Normal priority
	}

	return nil
}

// processBatchUpdates processes queued configuration updates in batches
func (cm *ConfigurationManager) processBatchUpdates() {
	updates := make([]ConfigurationUpdate, 0, cm.batchSize)
	
	for {
		select {
		case update := <-cm.batchCh:
			updates = append(updates, update)
			
			// Process batch when full or after timeout
			if len(updates) >= cm.batchSize {
				cm.processBatch(updates)
				updates = updates[:0]
			}
			
		case <-cm.updateTicker.C:
			// Process any pending updates on timer
			if len(updates) > 0 {
				cm.processBatch(updates)
				updates = updates[:0]
			}
		}
	}
}

// processBatch processes a batch of configuration updates
func (cm *ConfigurationManager) processBatch(updates []ConfigurationUpdate) {
	if len(updates) == 0 {
		return
	}

	// Sort by priority (high priority first)
	cm.sortUpdatesByPriority(updates)

	successful := 0
	failed := 0

	for _, update := range updates {
		// Update cache
		cachedConfig := &CachedConfiguration{
			Config:    update.Config,
			Version:   update.Version,
			Hash:      update.Hash,
			Timestamp: update.Timestamp,
			SendCount: 0,
		}

		// Check if configuration has actually changed
		if cm.hasConfigurationChanged(update.SystemID, cachedConfig) {
			cm.cache.Store(update.SystemID, cachedConfig)
			
			if err := cm.sendConfigToSystem(update.SystemID, cachedConfig); err != nil {
				slog.Error("Failed to send configuration in batch", "system", update.SystemID, "err", err)
				failed++
			} else {
				successful++
			}
		}
	}

	slog.Info("Processed configuration batch", "total", len(updates), "successful", successful, "failed", failed)
}

// processImmediateUpdate processes a high-priority update immediately
func (cm *ConfigurationManager) processImmediateUpdate(update ConfigurationUpdate) {
	cachedConfig := &CachedConfiguration{
		Config:    update.Config,
		Version:   update.Version,
		Hash:      update.Hash,
		Timestamp: update.Timestamp,
		SendCount: 0,
	}

	cm.cache.Store(update.SystemID, cachedConfig)
	
	if err := cm.sendConfigToSystem(update.SystemID, cachedConfig); err != nil {
		slog.Error("Failed to send immediate configuration update", "system", update.SystemID, "err", err)
	}
}

// sendConfigToSystem sends configuration to a specific system
func (cm *ConfigurationManager) sendConfigToSystem(systemID string, config *CachedConfiguration) error {
	// Find the system in the system manager
	if cm.hub.sm == nil {
		return fmt.Errorf("system manager not initialized")
	}

	system, exists := cm.hub.sm.GetSystem(systemID)
	if !exists || system == nil {
		return fmt.Errorf("system not found: %s", systemID)
	}

	// Send config via WebSocket if available
	if system.WsConn != nil && system.WsConn.IsConnected() {
		versionedConfig := map[string]interface{}{
			"config":  config.Config,
			"version": config.Version,
		}

		err := system.WsConn.SendMonitoringConfig(versionedConfig)
		if err != nil {
			return fmt.Errorf("failed to send config via WebSocket: %w", err)
		}

		// Update send statistics
		config.SendCount++
		config.LastSent = time.Now()
		cm.cache.Store(systemID, config)

		slog.Debug("Configuration sent via WebSocket", "system", systemID, "version", config.Version)
		return nil
	}

	return fmt.Errorf("system %s not connected via WebSocket", systemID)
}

// hasConfigurationChanged checks if the configuration has actually changed
func (cm *ConfigurationManager) hasConfigurationChanged(systemID string, newConfig *CachedConfiguration) bool {
	if cached, ok := cm.cache.Load(systemID); ok {
		cachedConfig := cached.(*CachedConfiguration)
		return cachedConfig.Hash != newConfig.Hash
	}
	return true // No cached config means it's new
}

// calculateConfigHash generates a hash of the configuration for change detection
func (cm *ConfigurationManager) calculateConfigHash(config system.MonitoringConfig) string {
	configBytes, err := json.Marshal(config)
	if err != nil {
		// Fallback to string representation
		configBytes = []byte(fmt.Sprintf("%+v", config))
	}

	hash := sha256.Sum256(configBytes)
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes for shorter hash
}

// getNextConfigVersion generates the next configuration version for a system
func (cm *ConfigurationManager) getNextConfigVersion(systemID string) int64 {
	now := time.Now().Unix()
	
	// Ensure version always increases
	if stored, ok := cm.versions.Load(systemID); ok {
		if lastVersion := stored.(int64); now <= lastVersion {
			now = lastVersion + 1
		}
	}
	
	cm.versions.Store(systemID, now)
	return now
}

// getAllConnectedSystems returns all system IDs that are currently connected
func (cm *ConfigurationManager) getAllConnectedSystems() []string {
	var systems []string
	
	if cm.hub.sm == nil {
		return systems
	}

	// Query the database for all non-paused systems
	var systemRecords []struct {
		Id string `db:"id" json:"id"`
	}
	
	err := cm.hub.DB().NewQuery("SELECT id FROM systems WHERE status != 'paused'").All(&systemRecords)
	if err != nil {
		slog.Error("Failed to get connected systems", "err", err)
		return systems
	}

	for _, record := range systemRecords {
		// Check if system is actually connected via WebSocket
		if system, exists := cm.hub.sm.GetSystem(record.Id); exists && system.WsConn != nil && system.WsConn.IsConnected() {
			systems = append(systems, record.Id)
		}
	}
	
	return systems
}

// sortUpdatesByPriority sorts updates by priority (1=high, 2=normal, 3=low)
func (cm *ConfigurationManager) sortUpdatesByPriority(updates []ConfigurationUpdate) {
	// Simple bubble sort by priority
	n := len(updates)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if updates[j].Priority > updates[j+1].Priority {
				updates[j], updates[j+1] = updates[j+1], updates[j]
			}
		}
	}
}

// GetConfigurationStats returns statistics about the configuration manager
func (cm *ConfigurationManager) GetConfigurationStats() map[string]interface{} {
	stats := map[string]interface{}{
		"cached_configs": 0,
		"pending_updates": len(cm.batchCh),
		"batch_size": cm.batchSize,
		"batch_timeout": cm.batchTimeout.String(),
		"cache_timeout": cm.cacheTimeout.String(),
	}

	// Count cached configurations
	cachedCount := 0
	cm.cache.Range(func(key, value interface{}) bool {
		cachedCount++
		return true
	})
	stats["cached_configs"] = cachedCount

	return stats
}

// Stop gracefully shuts down the configuration manager
func (cm *ConfigurationManager) Stop() {
	if cm.updateTicker != nil {
		cm.updateTicker.Stop()
	}
	close(cm.batchCh)
}