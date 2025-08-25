package agent

import (
	"beszel/internal/entities/system"
	"log/slog"
	"time"
)

// Not thread safe since we only access from gatherStats which is already locked
type SessionCache struct {
	data           *system.CombinedData
	lastUpdate     time.Time
	primarySession string
	leaseTime      time.Duration
}

func NewSessionCache(leaseTime time.Duration) *SessionCache {
	return &SessionCache{
		leaseTime: leaseTime,
		data:      &system.CombinedData{},
	}
}

func (c *SessionCache) Get(sessionID string) (stats *system.CombinedData, isCached bool) {
	timeSinceUpdate := time.Since(c.lastUpdate)
	withinLeaseTime := timeSinceUpdate < c.leaseTime
	slog.Debug("SessionCache.Get called", 
		"session_id", sessionID, 
		"primary_session", c.primarySession, 
		"time_since_update", timeSinceUpdate.String(), 
		"lease_time", c.leaseTime.String(),
		"within_lease_time", withinLeaseTime,
		"is_primary", sessionID == c.primarySession)
	
	if sessionID != c.primarySession && withinLeaseTime {
		slog.Debug("SessionCache returning cached data", "speedtest_results_count", len(c.data.Stats.SpeedtestResults))
		for serverID := range c.data.Stats.SpeedtestResults {
			slog.Debug("Cached speedtest result being returned", "server_id", serverID)
		}
		return c.data, true
	}
	slog.Debug("SessionCache returning fresh data required")
	return c.data, false
}

func (c *SessionCache) Set(sessionID string, data *system.CombinedData) {
	if data != nil {
		*c.data = *data
		slog.Debug("SessionCache.Set called", 
			"session_id", sessionID, 
			"speedtest_results_count", len(data.Stats.SpeedtestResults))
		for serverID := range data.Stats.SpeedtestResults {
			slog.Debug("Caching speedtest result", "server_id", serverID)
		}
	} else {
		slog.Debug("SessionCache.Set called with nil data", "session_id", sessionID)
	}
	c.primarySession = sessionID
	c.lastUpdate = time.Now()
}

// Clear invalidates the cache by resetting the last update time
func (c *SessionCache) Clear() {
	slog.Debug("SessionCache.Clear called", "speedtest_results_before_clear", len(c.data.Stats.SpeedtestResults))
	for serverID := range c.data.Stats.SpeedtestResults {
		slog.Debug("Clearing cached speedtest result", "server_id", serverID)
	}
	c.lastUpdate = time.Time{} // Reset to zero time to force cache miss
	c.primarySession = ""
	slog.Debug("SessionCache cleared successfully")
}
