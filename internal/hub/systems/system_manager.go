package systems

import (
	"beszel/internal/entities/system"
	"beszel/internal/hub/ws"
	"errors"
	"fmt"
	"time"

	"github.com/blang/semver"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/store"
)

// System status constants
const (
	up      string = "up"      // System is online and responding
	down    string = "down"    // System is offline or not responding
	paused  string = "paused"  // System monitoring is paused
	pending string = "pending" // System is waiting on initial connection result

	// interval is the default update interval in milliseconds (60 seconds)
	interval int = 60_000
	// interval int = 10_000 // Debug interval for faster updates
)

var (
	// errSystemExists is returned when attempting to add a system that already exists
	errSystemExists = errors.New("system exists")
)

// SystemManager manages a collection of monitored systems and their connections.
// It handles system lifecycle, status updates, and maintains WebSocket connections.
type SystemManager struct {
	hub        hubLike                       // Hub interface for database and alert operations
	systems    *store.Store[string, *System] // Thread-safe store of active systems
	configSent map[string]bool               // Track which systems have received monitoring config
}

// hubLike defines the interface requirements for the hub dependency.
// It extends core.App with system-specific functionality.
type hubLike interface {
	core.App
	HandleSystemAlerts(systemRecord *core.Record, data *system.CombinedData) error
	HandleStatusAlerts(status string, systemRecord *core.Record) error
	SendMonitoringConfigToAgent(systemRecord *core.Record) error
}

// NewSystemManager creates a new SystemManager instance with the provided hub.
func NewSystemManager(hub hubLike) *SystemManager {
	sm := &SystemManager{
		hub:        hub,
		systems:    store.New(map[string]*System{}),
		configSent: make(map[string]bool),
	}
	sm.bindEventHooks()
	return sm
}

// Initialize sets up the system manager by binding event hooks and starting existing systems.
// It begins monitoring all non-paused systems from the database.
// Systems are started with staggered delays to prevent overwhelming the hub during startup.
func (sm *SystemManager) Initialize() error {
	sm.bindEventHooks()

	// Load existing systems from database (excluding paused ones)
	var systems []*System
	err := sm.hub.DB().NewQuery("SELECT id, host, status FROM systems WHERE status != 'paused'").All(&systems)
	if err != nil || len(systems) == 0 {
		return err
	}

	// Start systems in background with staggered timing
	go func() {
		// Calculate staggered delay between system starts (max 2 seconds per system)
		delta := interval / max(1, len(systems))
		delta = min(delta, 2_000)
		sleepTime := time.Duration(delta) * time.Millisecond

		for _, system := range systems {
			time.Sleep(sleepTime)
			_ = sm.AddSystem(system)
		}
	}()
	return nil
}

// bindEventHooks registers event handlers for system and fingerprint record changes.
// These hooks ensure the system manager stays synchronized with database changes.
func (sm *SystemManager) bindEventHooks() {
	sm.hub.OnRecordCreate("systems").BindFunc(sm.onRecordCreate)
	sm.hub.OnRecordAfterCreateSuccess("systems").BindFunc(sm.onRecordAfterCreateSuccess)
	sm.hub.OnRecordUpdate("systems").BindFunc(sm.onRecordUpdate)
	sm.hub.OnRecordAfterUpdateSuccess("systems").BindFunc(sm.onRecordAfterUpdateSuccess)
	sm.hub.OnRecordAfterDeleteSuccess("systems").BindFunc(sm.onRecordAfterDeleteSuccess)
	sm.hub.OnRecordAfterUpdateSuccess("fingerprints").BindFunc(sm.onTokenRotated)
}

// onTokenRotated handles fingerprint token rotation events.
// When a system's authentication token is rotated, any existing WebSocket connection
// must be closed to force re-authentication with the new token.
func (sm *SystemManager) onTokenRotated(e *core.RecordEvent) error {
	systemID := e.Record.GetString("system")
	system, ok := sm.systems.GetOk(systemID)
	if !ok {
		return e.Next()
	}
	// No need to close connection if not connected via websocket
	if system.WsConn == nil {
		return e.Next()
	}
	system.setDown(nil)
	sm.RemoveSystem(systemID)
	return e.Next()
}

// onRecordCreate is called before a new system record is committed to the database.
// It initializes the record with default values: empty info and pending status.
func (sm *SystemManager) onRecordCreate(e *core.RecordEvent) error {
	e.Record.Set("info", system.Info{})
	e.Record.Set("status", pending)
	return e.Next()
}

// onRecordAfterCreateSuccess is called after a new system record is successfully created.
// It adds the new system to the manager to begin monitoring.
func (sm *SystemManager) onRecordAfterCreateSuccess(e *core.RecordEvent) error {
	if err := sm.AddRecord(e.Record, nil); err != nil {
		e.App.Logger().Error("Error adding record", "err", err)
	}
	return e.Next()
}

// onRecordUpdate is called before a system record is updated in the database.
// It clears system info when the status is changed to paused.
func (sm *SystemManager) onRecordUpdate(e *core.RecordEvent) error {
	if e.Record.GetString("status") == paused {
		e.Record.Set("info", system.Info{})
	}
	return e.Next()
}

// onRecordAfterUpdateSuccess handles system record updates after they're committed to the database.
// It manages system lifecycle based on status changes and triggers appropriate alerts.
// Status transitions are handled as follows:
// - paused: Closes connection and deactivates alerts
// - pending: Starts monitoring (reuses WebSocket if available)
// - up: Triggers system alerts
// - down: Triggers status change alerts
func (sm *SystemManager) onRecordAfterUpdateSuccess(e *core.RecordEvent) error {
	newStatus := e.Record.GetString("status")
	prevStatus := pending
	system, ok := sm.systems.GetOk(e.Record.Id)
	if ok {
		prevStatus = system.Status
		system.Status = newStatus
	}

	switch newStatus {
	case paused:
		if ok {
			// Pause monitoring but keep system in manager for potential resume
		}
		_ = deactivateAlerts(e.App, e.Record.Id)
		return e.Next()
	case pending:
		// Resume monitoring, preferring existing WebSocket connection
		if ok && system.WsConn != nil {
			go system.update()
			return e.Next()
		}
		// Start new monitoring session
		if err := sm.AddRecord(e.Record, nil); err != nil {
			e.App.Logger().Error("Error adding record", "err", err)
		}
		_ = deactivateAlerts(e.App, e.Record.Id)
		return e.Next()
	}

	// Handle systems not in manager
	if !ok {
		return sm.AddRecord(e.Record, nil)
	}

	// Trigger system alerts when system comes online
	if newStatus == up {
		if err := sm.hub.HandleSystemAlerts(e.Record, system.data); err != nil {
			e.App.Logger().Error("Error handling system alerts", "err", err)
		}
	}

	// Trigger status change alerts for up/down transitions
	if (newStatus == down && prevStatus == up) || (newStatus == up && prevStatus == down) {
		if err := sm.hub.HandleStatusAlerts(newStatus, e.Record); err != nil {
			e.App.Logger().Error("Error handling status alerts", "err", err)
		}
	}
	return e.Next()
}

// onRecordAfterDeleteSuccess is called after a system record is successfully deleted.
// It removes the system from the manager and cleans up all associated resources.
func (sm *SystemManager) onRecordAfterDeleteSuccess(e *core.RecordEvent) error {
	sm.RemoveSystem(e.Record.Id)
	return e.Next()
}

// AddSystem adds a system to the manager and starts monitoring it.
// It validates required fields, initializes the system context, and starts the update goroutine.
// Returns error if a system with the same ID already exists.
func (sm *SystemManager) AddSystem(sys *System) error {
	if sm.systems.Has(sys.Id) {
		return errSystemExists
	}
	if sys.Id == "" || sys.Host == "" {
		return errors.New("system missing required fields")
	}

	// Initialize system for monitoring
	sys.manager = sm
	sys.ctx, sys.cancel = sys.getContext()
	sys.data = &system.CombinedData{}
	sm.systems.Set(sys.Id, sys)

	// Start monitoring in background
	go sys.StartUpdater()
	return nil
}

// RemoveSystem removes a system from the manager and cleans up all associated resources.
// It cancels the system's context, closes all connections, and removes it from the store.
// Returns an error if the system is not found.
func (sm *SystemManager) RemoveSystem(systemID string) error {
	system, ok := sm.systems.GetOk(systemID)
	if !ok {
		return errors.New("system not found")
	}

	// Stop the update goroutine
	if system.cancel != nil {
		system.cancel()
	}

	// Clean up WebSocket connection
	system.closeWebSocketConnection()
	sm.systems.Remove(systemID)
	sm.ClearConfigSent(systemID)
	return nil
}

// AddRecord creates a System instance from a database record and adds it to the manager.
// If a system with the same ID already exists, it's removed first to ensure clean state.
// If no system instance is provided, a new one is created.
// This method is typically called when systems are created or their status changes to pending.
func (sm *SystemManager) AddRecord(record *core.Record, system *System) (err error) {
	// Remove existing system to ensure clean state
	if sm.systems.Has(record.Id) {
		_ = sm.RemoveSystem(record.Id)
	}

	// Create new system if none provided
	if system == nil {
		system = sm.NewSystem(record.Id)
	}

	// Populate system from record
	system.Status = record.GetString("status")
	system.Host = record.GetString("host")

	return sm.AddSystem(system)
}

// AddWebSocketSystem creates and adds a system with an established WebSocket connection.
// This method is called when an agent connects via WebSocket with valid authentication.
// The system is immediately added to monitoring with the provided connection and version info.
func (sm *SystemManager) AddWebSocketSystem(systemId string, agentVersion semver.Version, wsConn *ws.WsConn) error {
	systemRecord, err := sm.hub.FindRecordById("systems", systemId)
	if err != nil {
		return err
	}

	system := sm.NewSystem(systemId)
	system.WsConn = wsConn
	system.agentVersion = agentVersion

	if err := sm.AddRecord(systemRecord, system); err != nil {
		return err
	}

	// Send unified monitoring configuration to the newly connected agent (startup only)
	go func() {
		sm.hub.Logger().Debug("Sending monitoring config to newly connected agent at startup", "system", systemId)

		if hubWithMonitoring, ok := sm.hub.(interface{ SendMonitoringConfigToAgent(*core.Record) error }); ok {
			sm.hub.Logger().Debug("Hub interface cast successful, sending monitoring config", "system", systemId)
			if err := hubWithMonitoring.SendMonitoringConfigToAgent(systemRecord); err != nil {
				sm.hub.Logger().Error("Failed to send monitoring config to newly connected agent", "system", systemId, "err", err)
			} else {
				sm.hub.Logger().Debug("Successfully sent monitoring config to newly connected agent", "system", systemId)
				// Mark that we've sent the configuration to this system
				sm.MarkConfigAsSent(systemId)
			}
		} else {
			sm.hub.Logger().Debug("Hub interface cast failed - monitoring config not available", "system", systemId)
		}
	}()

	return nil
}

// GetSystem returns a system by ID from the store
func (sm *SystemManager) GetSystem(systemID string) (*System, bool) {
	return sm.systems.GetOk(systemID)
}

// HasConfigBeenSent checks if monitoring config has been sent to a system
func (sm *SystemManager) HasConfigBeenSent(systemID string) bool {
	return sm.configSent[systemID]
}

// MarkConfigAsSent marks that monitoring config has been sent to a system
func (sm *SystemManager) MarkConfigAsSent(systemID string) {
	sm.configSent[systemID] = true
}

// ClearConfigSent clears the config sent status for a system
func (sm *SystemManager) ClearConfigSent(systemID string) {
	delete(sm.configSent, systemID)
}

// deactivateAlerts finds all triggered alerts for a system and sets them to inactive.
// This is called when a system is paused or goes offline to prevent continued alerts.
func deactivateAlerts(app core.App, systemID string) error {
	// Note: Direct SQL updates don't trigger SSE, so we use the PocketBase API
	// _, err := app.DB().NewQuery(fmt.Sprintf("UPDATE alerts SET triggered = false WHERE system = '%s'", systemID)).Execute()

	alerts, err := app.FindRecordsByFilter("alerts", fmt.Sprintf("system = '%s' && triggered = 1", systemID), "", -1, 0)
	if err != nil {
		return err
	}

	for _, alert := range alerts {
		alert.Set("triggered", false)
		if err := app.SaveNoValidate(alert); err != nil {
			return err
		}
	}
	return nil
}
