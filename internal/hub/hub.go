// Package hub handles updating systems and serving the web UI.
package hub

import (
	"beszel"
	"beszel/internal/alerts"
	"beszel/internal/entities/system"
	"beszel/internal/hub/config"
	"beszel/internal/hub/systems"
	"beszel/internal/records"
	"beszel/internal/users"
	"beszel/site"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type Hub struct {
	core.App
	*alerts.AlertManager
	um            *users.UserManager
	rm            *records.RecordManager
	sm            *systems.SystemManager
	configManager *ConfigurationManager // Optimized configuration management
	authKey       string                 // Base64 authentication key for agents
	appURL        string
}

// NewHub creates a new Hub instance with default configuration
func NewHub(app core.App) *Hub {
	hub := &Hub{}
	hub.App = app

	hub.AlertManager = alerts.NewAlertManager(hub)
	hub.um = users.NewUserManager(hub)
	hub.rm = records.NewRecordManager(hub)
	hub.sm = systems.NewSystemManager(hub)
	hub.configManager = NewConfigurationManager(hub) // Initialize configuration manager
	hub.appURL, _ = GetEnv("APP_URL")

	// Generate base64 authentication key for agents
	hub.generateAuthKey()

	return hub
}

// generateAuthKey creates a random base64 key for agent authentication
func (h *Hub) generateAuthKey() {
	// Try to load existing key from disk first
	if h.loadAuthKeyFromDisk() {
		slog.Info("Loaded existing auth key from disk")
		return
	}

	slog.Info("No existing auth key found, generating new one")

	// Generate new key if none exists
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		// Fallback to a deterministic key if random generation fails
		keyBytes = []byte("default-auth-key-for-beszel-hub")
	}

	// Encode to base64
	h.authKey = "base64:" + base64.StdEncoding.EncodeToString(keyBytes)

	// Save the new key to disk
	h.saveAuthKeyToDisk()
}

// loadAuthKeyFromDisk loads the authentication key from disk
func (h *Hub) loadAuthKeyFromDisk() bool {
	keyPath := filepath.Join(h.DataDir(), "auth_key")
	slog.Debug("Trying to load auth key from", "path", keyPath)
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		slog.Debug("Failed to load auth key from disk", "err", err)
		return false
	}

	h.authKey = string(keyData)
	slog.Debug("Successfully loaded auth key from disk")
	return true
}

// saveAuthKeyToDisk saves the authentication key to disk
func (h *Hub) saveAuthKeyToDisk() {
	keyPath := filepath.Join(h.DataDir(), "auth_key")
	slog.Debug("Saving auth key to disk", "path", keyPath)
	err := os.WriteFile(keyPath, []byte(h.authKey), 0600)
	if err != nil {
		slog.Error("Failed to save auth key to disk", "err", err)
	} else {
		slog.Info("Successfully saved auth key to disk")
	}
}

// GetAuthKey returns the base64 authentication key for agents
func (h *Hub) GetAuthKey() string {
	return h.authKey
}

// GetEnv retrieves an environment variable with a "BESZEL_HUB_" prefix, or falls back to the unprefixed key.
func GetEnv(key string) (value string, exists bool) {
	if value, exists = os.LookupEnv("BESZEL_HUB_" + key); exists {
		return value, exists
	}
	// Fallback to the old unprefixed key
	return os.LookupEnv(key)
}

func (h *Hub) StartHub() error {
	// Add shutdown hook for configuration manager
	h.App.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		if h.configManager != nil {
			h.configManager.Stop()
		}
		return e.Next()
	})

	h.App.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// initialize settings / collections
		if err := h.initialize(e); err != nil {
			return err
		}
		// sync systems with config
		if err := config.SyncSystems(e); err != nil {
			return err
		}
		// register api routes
		if err := h.registerApiRoutes(e); err != nil {
			return err
		}
		// register cron jobs
		if err := h.registerCronJobs(e); err != nil {
			return err
		}
		// start server
		if err := h.startServer(e); err != nil {
			return err
		}
		// start system updates
		if err := h.sm.Initialize(); err != nil {
			return err
		}
		return e.Next()
	})

	// TODO: move to users package
	// handle default values for user / user_settings creation
	h.App.OnRecordCreate("users").BindFunc(h.um.InitializeUserRole)
	h.App.OnRecordCreate("user_settings").BindFunc(h.um.InitializeUserSettings)

	// handle system record updates (for initial config sending on startup)
	h.App.OnRecordAfterUpdateSuccess("systems").BindFunc(h.onSystemRecordUpdate)
	// handle monitoring configuration changes
	h.App.OnRecordAfterUpdateSuccess("monitoring_config").BindFunc(h.onMonitoringConfigUpdate)
	h.App.OnRecordAfterCreateSuccess("monitoring_config").BindFunc(h.onMonitoringConfigUpdate)
	h.App.OnRecordAfterDeleteSuccess("monitoring_config").BindFunc(h.onMonitoringConfigDelete)

	if pb, ok := h.App.(*pocketbase.PocketBase); ok {
		// log.Println("Starting pocketbase")
		err := pb.Start()
		if err != nil {
			return err
		}
	}

	return nil
}

// initialize sets up initial configuration (collections, settings, etc.)
func (h *Hub) initialize(e *core.ServeEvent) error {
	// set general settings
	settings := e.App.Settings()
	// batch requests (for global alerts)
	settings.Batch.Enabled = true
	// set URL if BASE_URL env is set
	if h.appURL != "" {
		settings.Meta.AppURL = h.appURL
	}
	if err := e.App.Save(settings); err != nil {
		return err
	}
	// set auth settings
	usersCollection, err := e.App.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	// disable email auth if DISABLE_PASSWORD_AUTH env var is set
	disablePasswordAuth, _ := GetEnv("DISABLE_PASSWORD_AUTH")
	usersCollection.PasswordAuth.Enabled = disablePasswordAuth != "true"
	usersCollection.PasswordAuth.IdentityFields = []string{"email"}
	// disable oauth if no providers are configured (todo: remove this in post 0.9.0 release)
	if usersCollection.OAuth2.Enabled {
		usersCollection.OAuth2.Enabled = len(usersCollection.OAuth2.Providers) > 0
	}
	// allow oauth user creation if USER_CREATION is set
	if userCreation, _ := GetEnv("USER_CREATION"); userCreation == "true" {
		cr := "@request.context = 'oauth2'"
		usersCollection.CreateRule = &cr
	} else {
		usersCollection.CreateRule = nil
	}
	if err := e.App.Save(usersCollection); err != nil {
		return err
	}
	// allow all users to access systems if SHARE_ALL_SYSTEMS is set
	systemsCollection, err := e.App.FindCachedCollectionByNameOrId("systems")
	if err != nil {
		return err
	}
	// Role-based access control: admins can add/modify, users can view
	systemsReadRule := "@request.auth.id != \"\""
	// Only admins can create, update, and delete systems
	systemsCreateRule := "@request.auth.id != \"\" && @request.auth.role = \"admin\""
	systemsUpdateRule := "@request.auth.id != \"\" && @request.auth.role = \"admin\""
	systemsDeleteRule := "@request.auth.id != \"\" && @request.auth.role = \"admin\""

	systemsCollection.ListRule = &systemsReadRule
	systemsCollection.ViewRule = &systemsReadRule
	systemsCollection.CreateRule = &systemsCreateRule
	systemsCollection.UpdateRule = &systemsUpdateRule
	systemsCollection.DeleteRule = &systemsDeleteRule
	if err := e.App.Save(systemsCollection); err != nil {
		return err
	}

	// Set alerts collection access rules - role-based access control
	alertsCollection, err := e.App.FindCachedCollectionByNameOrId("alerts")
	if err != nil {
		return err
	}
	// Alerts: admins can manage, users can view
	alertsReadRule := "@request.auth.id != \"\""
	alertsCreateRule := "@request.auth.id != \"\" && @request.auth.role = \"admin\""
	alertsUpdateRule := "@request.auth.id != \"\" && @request.auth.role = \"admin\""
	alertsDeleteRule := "@request.auth.id != \"\" && @request.auth.role = \"admin\""

	alertsCollection.ListRule = &alertsReadRule
	alertsCollection.ViewRule = &alertsReadRule
	alertsCollection.CreateRule = &alertsCreateRule
	alertsCollection.UpdateRule = &alertsUpdateRule
	alertsCollection.DeleteRule = &alertsDeleteRule
	if err := e.App.Save(alertsCollection); err != nil {
		return err
	}

	// Set monitoring_config collection access rules - only admins can manage
	monitoringConfigCollection, err := e.App.FindCachedCollectionByNameOrId("monitoring_config")
	if err != nil {
		return err
	}
	// Monitoring config: all users can read, only admins can manage
	monitoringConfigReadRule := "@request.auth.id != \"\""
	monitoringConfigCreateRule := "@request.auth.id != \"\" && @request.auth.role = \"admin\""
	monitoringConfigUpdateRule := "@request.auth.id != \"\" && @request.auth.role = \"admin\""
	monitoringConfigDeleteRule := "@request.auth.id != \"\" && @request.auth.role = \"admin\""

	monitoringConfigCollection.ListRule = &monitoringConfigReadRule
	monitoringConfigCollection.ViewRule = &monitoringConfigReadRule
	monitoringConfigCollection.CreateRule = &monitoringConfigCreateRule
	monitoringConfigCollection.UpdateRule = &monitoringConfigUpdateRule
	monitoringConfigCollection.DeleteRule = &monitoringConfigDeleteRule
	if err := e.App.Save(monitoringConfigCollection); err != nil {
		return err
	}
	return nil
}

// startServer sets up the server for Beszel
func (h *Hub) startServer(se *core.ServeEvent) error {
	// TODO: exclude dev server from production binary
	switch h.IsDev() {
	case true:
		proxy := httputil.NewSingleHostReverseProxy(&url.URL{
			Scheme: "http",
			Host:   "localhost:5173",
		})
		se.Router.GET("/{path...}", func(e *core.RequestEvent) error {
			proxy.ServeHTTP(e.Response, e.Request)
			return nil
		})
	default:
		// parse app url
		parsedURL, err := url.Parse(h.appURL)
		if err != nil {
			return err
		}
		// fix base paths in html if using subpath
		basePath := strings.TrimSuffix(parsedURL.Path, "/") + "/"
		indexFile, _ := fs.ReadFile(site.DistDirFS, "index.html")
		indexContent := strings.ReplaceAll(string(indexFile), "./", basePath)
		indexContent = strings.Replace(indexContent, "{{V}}", beszel.Version, 1)
		indexContent = strings.Replace(indexContent, "{{HUB_URL}}", h.appURL, 1)
		// set up static asset serving
		staticPaths := [2]string{"/static/", "/assets/"}
		serveStatic := apis.Static(site.DistDirFS, false)
		// get CSP configuration
		csp, cspExists := GetEnv("CSP")
		// add route
		se.Router.GET("/{path...}", func(e *core.RequestEvent) error {
			// serve static assets if path is in staticPaths
			for i := range staticPaths {
				if strings.Contains(e.Request.URL.Path, staticPaths[i]) {
					e.Response.Header().Set("Cache-Control", "public, max-age=2592000")
					return serveStatic(e)
				}
			}
			if cspExists {
				e.Response.Header().Del("X-Frame-Options")
				e.Response.Header().Set("Content-Security-Policy", csp)
			}
			return e.HTML(http.StatusOK, indexContent)
		})
	}
	return nil
}

// registerCronJobs sets up scheduled tasks
func (h *Hub) registerCronJobs(_ *core.ServeEvent) error {
	// delete old records based on retention policy once every hour
	h.Cron().MustAdd("delete old records", "8 * * * *", h.rm.DeleteOldRecords)
	// calculate system averages every 5 minutes
	h.Cron().MustAdd("calculate system averages", "*/5 * * * *", func() {
		if err := h.calculateSystemAverages(); err != nil {
			h.Logger().Error("Failed to calculate system averages", "err", err)
		}
	})

	return nil
}

// custom api routes
func (h *Hub) registerApiRoutes(se *core.ServeEvent) error {
	// returns auth key and version
	se.Router.GET("/api/beszel/getkey", func(e *core.RequestEvent) error {
		info, _ := e.RequestInfo()
		if info.Auth == nil {
			return apis.NewForbiddenError("Forbidden", nil)
		}

		return e.JSON(http.StatusOK, map[string]string{"key": h.GetAuthKey(), "v": beszel.Version})
	})
	// check if first time setup on login page
	se.Router.GET("/api/beszel/first-run", func(e *core.RequestEvent) error {
		total, err := h.CountRecords("users")
		return e.JSON(http.StatusOK, map[string]bool{"firstRun": err == nil && total == 0})
	})
	// send test notification
	se.Router.GET("/api/beszel/send-test-notification", h.SendTestNotification)
	// manually trigger average calculation for testing
	se.Router.GET("/api/beszel/calculate-averages", func(e *core.RequestEvent) error {
		if err := h.calculateSystemAverages(); err != nil {
			return e.JSON(500, map[string]string{"error": err.Error()})
		}
		return e.JSON(200, map[string]string{"status": "averages calculated"})
	})
	// API endpoint to get config.yml content
	se.Router.GET("/api/beszel/config-yaml", config.GetYamlConfig)
	// Configuration management endpoints
	se.Router.GET("/api/beszel/config/stats", h.getConfigurationStats)
	se.Router.POST("/api/beszel/config/sync-all", h.syncConfigurationToAllAgents)
	se.Router.POST("/api/beszel/config/sync/{id}", h.syncConfigurationToAgent)
	// handle agent websocket connection
	se.Router.GET("/api/beszel/agent-connect", h.handleAgentConnect)
	// get or create universal tokens
	se.Router.GET("/api/beszel/universal-token", h.getUniversalToken)
	// create first user endpoint only needed if no users exist
	if totalUsers, _ := h.CountRecords("users"); totalUsers == 0 {
		se.Router.POST("/api/beszel/create-user", h.um.CreateFirstUser)
	}
	return nil
}

// Handler for universal token API endpoint (create, read, delete)
func (h *Hub) getUniversalToken(e *core.RequestEvent) error {
	info, err := e.RequestInfo()
	if err != nil || info.Auth == nil {
		return apis.NewForbiddenError("Forbidden", nil)
	}

	// The JWT manager is no longer used, so we return a placeholder
	tokenMap := universalTokenMap.GetMap()
	userID := info.Auth.Id
	query := e.Request.URL.Query()
	token := query.Get("token")
	tokenSet := token != ""

	if !tokenSet {
		// return existing token if it exists
		if token, _, ok := tokenMap.GetByValue(userID); ok {
			return e.JSON(http.StatusOK, map[string]any{"token": token, "active": true})
		}
		// if no token is provided, generate a new one
		token = uuid.New().String()
	}
	response := map[string]any{"token": token}

	switch query.Get("enable") {
	case "1":
		tokenMap.Set(token, userID, time.Hour)
	case "0":
		tokenMap.RemovebyValue(userID)
	}
	_, response["active"] = tokenMap.GetOk(token)
	return e.JSON(http.StatusOK, response)
}

// MakeLink formats a link with the app URL and path segments.
// Only path segments should be provided.
func (h *Hub) MakeLink(parts ...string) string {
	base := strings.TrimSuffix(h.Settings().Meta.AppURL, "/")
	for _, part := range parts {
		if part == "" {
			continue
		}
		base = fmt.Sprintf("%s/%s", base, url.PathEscape(part))
	}
	return base
}

// getConfigurationStats returns statistics about the configuration manager
func (h *Hub) getConfigurationStats(e *core.RequestEvent) error {
	info, _ := e.RequestInfo()
	if info.Auth == nil {
		return apis.NewForbiddenError("Forbidden", nil)
	}

	stats := map[string]interface{}{
		"config_manager_initialized": h.configManager != nil,
	}

	if h.configManager != nil {
		configStats := h.configManager.GetConfigurationStats()
		for key, value := range configStats {
			stats[key] = value
		}
	}

	return e.JSON(http.StatusOK, stats)
}

// syncConfigurationToAllAgents triggers configuration sync to all connected agents
func (h *Hub) syncConfigurationToAllAgents(e *core.RequestEvent) error {
	info, _ := e.RequestInfo()
	if info.Auth == nil || info.Auth.GetString("role") != "admin" {
		return apis.NewForbiddenError("Admin access required", nil)
	}

	if h.configManager == nil {
		return e.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Configuration manager not initialized",
		})
	}

	err := h.configManager.SendConfigurationToAllAgents()
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return e.JSON(http.StatusOK, map[string]string{
		"status": "Configuration sync queued for all agents",
	})
}

// syncConfigurationToAgent triggers configuration sync to a specific agent
func (h *Hub) syncConfigurationToAgent(e *core.RequestEvent) error {
	info, _ := e.RequestInfo()
	if info.Auth == nil || info.Auth.GetString("role") != "admin" {
		return apis.NewForbiddenError("Admin access required", nil)
	}

	if h.configManager == nil {
		return e.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Configuration manager not initialized",
		})
	}

	systemID := e.Request.PathValue("id")
	if systemID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "System ID required",
		})
	}

	err := h.configManager.SendConfigurationToAgent(systemID, 1) // High priority
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return e.JSON(http.StatusOK, map[string]string{
		"status": "Configuration sync triggered for system " + systemID,
	})
}

// onMonitoringConfigUpdate handles monitoring configuration updates
func (h *Hub) onMonitoringConfigUpdate(e *core.RecordEvent) error {
	systemID := e.Record.GetString("system")
	if systemID == "" {
		return e.Next()
	}

	h.Logger().Debug("Monitoring configuration updated", "system", systemID)

	// Use configuration manager to queue the update if available
	if h.configManager != nil {
		// Clear cache for this system to force reload
		h.configManager.cache.Delete(systemID)
		
		// Queue configuration update with normal priority
		if config, err := h.configManager.GetConfiguration(systemID); err == nil {
			h.configManager.QueueConfigurationUpdate(systemID, config.Config, 2)
		}
	} else {
		// Fallback to direct config sending
		if systemRecord, err := h.FindRecordById("systems", systemID); err == nil {
			go h.SendMonitoringConfigToAgent(systemRecord)
		}
	}

	return e.Next()
}

// onMonitoringConfigDelete handles monitoring configuration deletions
func (h *Hub) onMonitoringConfigDelete(e *core.RecordEvent) error {
	systemID := e.Record.GetString("system")
	if systemID == "" {
		return e.Next()
	}

	h.Logger().Debug("Monitoring configuration deleted", "system", systemID)

	// Use configuration manager to send empty config if available
	if h.configManager != nil {
		// Clear cache for this system
		h.configManager.cache.Delete(systemID)
		
		// Queue empty configuration update with high priority
		emptyConfig := system.MonitoringConfig{}
		h.configManager.QueueConfigurationUpdate(systemID, emptyConfig, 1)
	} else {
		// Fallback to direct config sending with empty config
		go h.sendMonitoringConfigToSystem(systemID, system.MonitoringConfig{})
	}

	return e.Next()
}
