// Package agent provides configuration caching for improved performance
package agent

import (
	"beszel/internal/entities/system"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// ConfigCache provides thread-safe caching of monitoring configurations
type ConfigCache struct {
	configs    map[string]*CachedConfig
	lastUpdate map[string]time.Time
	ttl        time.Duration
	mutex      sync.RWMutex
}

// CachedConfig wraps a monitoring configuration with metadata
type CachedConfig struct {
	Config      *system.MonitoringConfig `json:"config"`
	Version     int64                    `json:"version"`
	Hash        string                   `json:"hash"`
	LastUpdated time.Time                `json:"last_updated"`
}

// NewConfigCache creates a new configuration cache with the specified TTL
func NewConfigCache(ttl time.Duration) *ConfigCache {
	return &ConfigCache{
		configs:    make(map[string]*CachedConfig),
		lastUpdate: make(map[string]time.Time),
		ttl:        ttl,
	}
}

// Get retrieves a cached configuration if it exists and hasn't expired
func (cc *ConfigCache) Get(systemID string) (*CachedConfig, bool) {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	if cached, exists := cc.configs[systemID]; exists {
		if time.Since(cc.lastUpdate[systemID]) < cc.ttl {
			return cached, true
		}
		// Config has expired, remove it
		delete(cc.configs, systemID)
		delete(cc.lastUpdate, systemID)
	}
	return nil, false
}

// Set stores a configuration in the cache
func (cc *ConfigCache) Set(systemID string, config *system.MonitoringConfig, version int64) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	hash := cc.calculateConfigHash(config)
	cachedConfig := &CachedConfig{
		Config:      config,
		Version:     version,
		Hash:        hash,
		LastUpdated: time.Now(),
	}

	cc.configs[systemID] = cachedConfig
	cc.lastUpdate[systemID] = time.Now()

	slog.Debug("Cached configuration", "system", systemID, "version", version, "hash", hash)
}

// Remove removes a configuration from the cache
func (cc *ConfigCache) Remove(systemID string) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	delete(cc.configs, systemID)
	delete(cc.lastUpdate, systemID)

	slog.Debug("Removed cached configuration", "system", systemID)
}

// Clear removes all cached configurations
func (cc *ConfigCache) Clear() {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	cc.configs = make(map[string]*CachedConfig)
	cc.lastUpdate = make(map[string]time.Time)

	slog.Debug("Cleared all cached configurations")
}

// GetStats returns cache statistics
func (cc *ConfigCache) GetStats() map[string]interface{} {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	now := time.Now()
	activeCount := 0
	expiredCount := 0

	for _, lastUpdate := range cc.lastUpdate {
		if now.Sub(lastUpdate) < cc.ttl {
			activeCount++
		} else {
			expiredCount++
		}
	}

	return map[string]interface{}{
		"total_configs":   len(cc.configs),
		"active_configs":  activeCount,
		"expired_configs": expiredCount,
		"cache_ttl":       cc.ttl.String(),
	}
}

// calculateConfigHash generates a hash of the configuration for change detection
func (cc *ConfigCache) calculateConfigHash(config *system.MonitoringConfig) string {
	// Create a deterministic representation of the config
	configData := map[string]interface{}{
		"enabled": map[string]bool{
			"ping":      config.Enabled.Ping,
			"dns":       config.Enabled.Dns,
			"http":      config.Enabled.Http,
			"speedtest": config.Enabled.Speedtest,
		},
		"global_interval": config.GlobalInterval,
		"ping": map[string]interface{}{
			"targets":  config.Ping.Targets,
			"interval": config.Ping.Interval,
		},
		"dns": map[string]interface{}{
			"targets":  config.Dns.Targets,
			"interval": config.Dns.Interval,
		},
		"http": map[string]interface{}{
			"targets":  config.Http.Targets,
			"interval": config.Http.Interval,
		},
		"speedtest": map[string]interface{}{
			"targets":  config.Speedtest.Targets,
			"interval": config.Speedtest.Interval,
		},
	}

	// Marshal to JSON for consistent hashing
	jsonData, err := json.Marshal(configData)
	if err != nil {
		// Fallback to simple string representation
		jsonData = []byte(fmt.Sprintf("%+v", config))
	}

	// Generate SHA256 hash
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes for shorter hash
}

// ConfigValidator validates monitoring configurations
type ConfigValidator struct {
	maxTargets     int
	maxInterval    time.Duration
	allowedDomains []string
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator(maxTargets int, maxInterval time.Duration, allowedDomains []string) *ConfigValidator {
	return &ConfigValidator{
		maxTargets:     maxTargets,
		maxInterval:    maxInterval,
		allowedDomains: allowedDomains,
	}
}

// ValidateConfig validates a monitoring configuration
func (cv *ConfigValidator) ValidateConfig(config *system.MonitoringConfig) error {
	var errors []string

	// Validate ping targets
	if len(config.Ping.Targets) > cv.maxTargets {
		errors = append(errors, fmt.Sprintf("too many ping targets: %d > %d", len(config.Ping.Targets), cv.maxTargets))
	}

	// Validate DNS targets
	for _, target := range config.Dns.Targets {
		if !cv.isAllowedDomain(target.Domain) {
			errors = append(errors, fmt.Sprintf("domain not allowed: %s", target.Domain))
		}
	}

	// Validate global interval (could be cron expression or duration)
	if config.GlobalInterval != "" {
		// Try to parse as duration first
		if _, err := time.ParseDuration(config.GlobalInterval); err != nil {
			// If not a duration, check if it's a valid cron expression
			if !cv.isValidCronExpression(config.GlobalInterval) {
				errors = append(errors, fmt.Sprintf("invalid global interval: %s", config.GlobalInterval))
			}
		}
	}

	// Validate individual service intervals (cron expressions)
	if config.Ping.Interval != "" {
		if !cv.isValidCronExpression(config.Ping.Interval) {
			errors = append(errors, fmt.Sprintf("invalid ping interval: %s", config.Ping.Interval))
		}
	}

	if config.Dns.Interval != "" {
		if !cv.isValidCronExpression(config.Dns.Interval) {
			errors = append(errors, fmt.Sprintf("invalid DNS interval: %s", config.Dns.Interval))
		}
	}

	if config.Http.Interval != "" {
		if !cv.isValidCronExpression(config.Http.Interval) {
			errors = append(errors, fmt.Sprintf("invalid HTTP interval: %s", config.Http.Interval))
		}
	}

	if config.Speedtest.Interval != "" {
		if !cv.isValidCronExpression(config.Speedtest.Interval) {
			errors = append(errors, fmt.Sprintf("invalid speedtest interval: %s", config.Speedtest.Interval))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// isAllowedDomain checks if a domain is in the allowed list
func (cv *ConfigValidator) isAllowedDomain(domain string) bool {
	if len(cv.allowedDomains) == 0 {
		return true // No restrictions if no domains specified
	}

	for _, allowed := range cv.allowedDomains {
		if domain == allowed {
			return true
		}
	}
	return false
}

// isValidCronExpression checks if a string is a valid cron expression
func (cv *ConfigValidator) isValidCronExpression(expression string) bool {
	// Basic cron expression validation
	// Cron expressions have 5 or 6 fields: minute hour day month weekday [year]
	parts := strings.Fields(expression)
	if len(parts) != 5 && len(parts) != 6 {
		return false
	}

	// Simple validation - check if it looks like a cron expression
	// This is a basic check, in production you might want more sophisticated validation
	for _, part := range parts {
		if part == "" {
			return false
		}
		// Check for common cron patterns: *, /, -, numbers
		if !strings.ContainsAny(part, "*/0123456789-,") {
			return false
		}
	}

	return true
}

// ConfigurationVersion tracks configuration changes
type ConfigurationVersion struct {
	Version     int64     `json:"version"`
	LastUpdated time.Time `json:"last_updated"`
	Hash        string    `json:"hash"`
}

// OptimizedConfigManager provides efficient configuration management
type OptimizedConfigManager struct {
	cache     *ConfigCache
	validator *ConfigValidator
	mutex     sync.RWMutex
}

// NewOptimizedConfigManager creates a new optimized configuration manager
func NewOptimizedConfigManager(cacheTTL time.Duration, maxTargets int, maxInterval time.Duration, allowedDomains []string) *OptimizedConfigManager {
	return &OptimizedConfigManager{
		cache:     NewConfigCache(cacheTTL),
		validator: NewConfigValidator(maxTargets, maxInterval, allowedDomains),
	}
}

// GetConfig retrieves a configuration, checking cache first
func (ocm *OptimizedConfigManager) GetConfig(systemID string) (*CachedConfig, bool) {
	return ocm.cache.Get(systemID)
}

// SetConfig validates and caches a configuration
func (ocm *OptimizedConfigManager) SetConfig(systemID string, config *system.MonitoringConfig, version int64) error {
	// Validate configuration
	if err := ocm.validator.ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration for system %s: %w", systemID, err)
	}

	// Cache the configuration
	ocm.cache.Set(systemID, config, version)

	return nil
}

// HasChanged checks if a configuration has changed since the last version
func (ocm *OptimizedConfigManager) HasChanged(systemID string, newConfig *system.MonitoringConfig, newVersion int64) bool {
	cached, exists := ocm.cache.Get(systemID)
	if !exists {
		return true // No cached config, consider it changed
	}

	if cached.Version >= newVersion {
		return false // Version hasn't increased
	}

	// Check if the actual configuration content has changed
	newHash := ocm.cache.calculateConfigHash(newConfig)
	return cached.Hash != newHash
}

// GetCacheStats returns cache statistics
func (ocm *OptimizedConfigManager) GetCacheStats() map[string]interface{} {
	return ocm.cache.GetStats()
}
