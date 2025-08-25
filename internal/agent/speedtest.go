package agent

import (
	"beszel/internal/entities/system"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type SpeedtestManager struct {
	sync.RWMutex
	targets         map[string]*speedtestTarget
	results         map[string]*system.SpeedtestResult
	lastResultsTime time.Time
	ctx             context.Context
	cancel          context.CancelFunc
	cronScheduler   *cron.Cron
	cronExpression  string
}

type speedtestTarget struct {
	ServerID  string
	Timeout   time.Duration
	lastCheck time.Time
}

// NewSpeedtestManager creates a new speedtest manager
func NewSpeedtestManager() (*SpeedtestManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	sm := &SpeedtestManager{
		targets:        make(map[string]*speedtestTarget),
		results:        make(map[string]*system.SpeedtestResult),
		ctx:            ctx,
		cancel:         cancel,
		cronScheduler:  cron.New(cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow))),
		cronExpression: "",
	}

	slog.Debug("Speedtest manager initialized")

	// Start the cron scheduler
	sm.cronScheduler.Start()

	// Schedule the speedtest job
	sm.scheduleSpeedtestJob()

	return sm, nil
}

// UpdateConfig updates the speedtest configuration with targets and cron expression
func (sm *SpeedtestManager) UpdateConfig(targets []system.SpeedtestTarget, cronExpression string) {
	sm.Lock()
	defer sm.Unlock()

	oldTargetsCount := len(sm.targets)
	oldResultsCount := len(sm.results)
	
	slog.Debug("UpdateConfig called", "old_targets", oldTargetsCount, "new_targets", len(targets), "cron_expression", cronExpression)

	// Use cron expression directly
	sm.cronExpression = cronExpression

	// Clear existing targets and results to prevent stale data
	sm.targets = make(map[string]*speedtestTarget)
	sm.results = make(map[string]*system.SpeedtestResult)
	
	if oldTargetsCount > 0 || oldResultsCount > 0 {
		slog.Info("Cleared old speedtest configuration", "old_targets", oldTargetsCount, "old_results", oldResultsCount)
	}

	// Add new targets
	for _, target := range targets {
		timeout := target.Timeout
		if timeout <= 0 {
			timeout = 60 * time.Second // Default 60 seconds for speedtest
		}

		sm.targets[target.ServerID] = &speedtestTarget{
			ServerID:  target.ServerID,
			Timeout:   time.Duration(timeout) * time.Second,
			lastCheck: time.Time{}, // Will trigger immediate check
		}
	}

	// Reschedule the speedtest job with new cron expression
	sm.scheduleSpeedtestJob()

	slog.Debug("Updated speedtest config", "targets", len(targets))
}

// GetResults returns the current speedtest results
func (sm *SpeedtestManager) GetResults() map[string]*system.SpeedtestResult {
	sm.Lock()
	defer sm.Unlock()

	slog.Debug("GetResults called", "current_results_count", len(sm.results))

	// If no results are available, return nil to indicate no speedtest tests have run
	if len(sm.results) == 0 {
		slog.Debug("No speedtest results available, returning nil")
		return nil
	}

	// Create a copy to avoid race conditions
	results := make(map[string]*system.SpeedtestResult)
	for serverID, result := range sm.results {
		results[serverID] = &system.SpeedtestResult{
			ServerURL:             result.ServerURL,
			Status:                result.Status,
			DownloadSpeed:         result.DownloadSpeed,
			UploadSpeed:           result.UploadSpeed,
			Latency:               result.Latency,
			ErrorCode:             result.ErrorCode,
			LastChecked:           result.LastChecked,
			PingJitter:            result.PingJitter,
			PingLow:               result.PingLow,
			PingHigh:              result.PingHigh,
			DownloadBytes:         result.DownloadBytes,
			DownloadElapsed:       result.DownloadElapsed,
			DownloadLatencyIQM:    result.DownloadLatencyIQM,
			DownloadLatencyLow:    result.DownloadLatencyLow,
			DownloadLatencyHigh:   result.DownloadLatencyHigh,
			DownloadLatencyJitter: result.DownloadLatencyJitter,
			UploadBytes:           result.UploadBytes,
			UploadElapsed:         result.UploadElapsed,
			UploadLatencyIQM:      result.UploadLatencyIQM,
			UploadLatencyLow:      result.UploadLatencyLow,
			UploadLatencyHigh:     result.UploadLatencyHigh,
			UploadLatencyJitter:   result.UploadLatencyJitter,
			PacketLoss:            result.PacketLoss,
			ISP:                   result.ISP,
			InterfaceExternalIP:   result.InterfaceExternalIP,
			ServerName:            result.ServerName,
			ServerLocation:        result.ServerLocation,
			ServerCountry:         result.ServerCountry,
			ServerHost:            result.ServerHost,
			ServerIP:              result.ServerIP,
		}
	}

	return results
}

// scheduleSpeedtestJob schedules the speedtest monitoring job
func (sm *SpeedtestManager) scheduleSpeedtestJob() {
	// Remove all existing jobs by creating a new scheduler
	sm.cronScheduler.Stop()
	sm.cronScheduler = cron.New(cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow))) // 5-field format
	sm.cronScheduler.Start()

	// Only schedule if we have a valid cron expression
	if sm.cronExpression != "" {
		_, err := sm.cronScheduler.AddFunc(sm.cronExpression, func() {
			slog.Debug("Running speedtest checks")
			sm.performSpeedtestChecks()
		})
		if err != nil {
			slog.Error("Failed to schedule speedtest job", "cron_expression", sm.cronExpression, "error", err)
		} else {
			slog.Debug("Speedtest job scheduled", "expression", sm.cronExpression)
		}
	} else {
		slog.Debug("No cron expression set, speedtest job not scheduled")
	}
}

// performSpeedtestChecks performs speedtest checks for all targets
func (sm *SpeedtestManager) performSpeedtestChecks() {
	sm.RLock()
	targets := make([]*speedtestTarget, 0, len(sm.targets))
	for _, target := range sm.targets {
		targets = append(targets, target)
	}
	sm.RUnlock()
	
	slog.Debug("Performing speedtest checks", "targets", len(targets))

	// Check targets sequentially (one after another)
	for _, target := range targets {
		result := sm.performSpeedtestCheck(target)

		sm.Lock()
		sm.results[target.ServerID] = result
		sm.lastResultsTime = time.Now()
		sm.Unlock()

		slog.Debug("Speedtest check completed",
			"server_id", target.ServerID,
			"status", result.Status,
			"download_speed", result.DownloadSpeed,
			"upload_speed", result.UploadSpeed,
			"latency", result.Latency)
	}
}

// SpeedtestCLIResult represents the JSON output from speedtest CLI
type SpeedtestCLIResult struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Ping      struct {
		Jitter  float64 `json:"jitter"`
		Latency float64 `json:"latency"`
		Low     float64 `json:"low"`
		High    float64 `json:"high"`
	} `json:"ping"`
	Download struct {
		Bandwidth int64 `json:"bandwidth"` // Bytes per second
		Bytes     int64 `json:"bytes"`
		Elapsed   int64 `json:"elapsed"`
		Latency   struct {
			IQM    float64 `json:"iqm"`
			Low    float64 `json:"low"`
			High   float64 `json:"high"`
			Jitter float64 `json:"jitter"`
		} `json:"latency"`
	} `json:"download"`
	Upload struct {
		Bandwidth int64 `json:"bandwidth"` // Bytes per second
		Bytes     int64 `json:"bytes"`
		Elapsed   int64 `json:"elapsed"`
		Latency   struct {
			IQM    float64 `json:"iqm"`
			Low    float64 `json:"low"`
			High   float64 `json:"high"`
			Jitter float64 `json:"jitter"`
		} `json:"latency"`
	} `json:"upload"`
	PacketLoss float64 `json:"packetLoss"`
	ISP        string  `json:"isp"`
	Interface  struct {
		InternalIP string `json:"internalIp"`
		Name       string `json:"name"`
		MacAddr    string `json:"macAddr"`
		IsVpn      bool   `json:"isVpn"`
		ExternalIP string `json:"externalIp"`
	} `json:"interface"`
	Server struct {
		ID       int    `json:"id"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Name     string `json:"name"`
		Location string `json:"location"`
		Country  string `json:"country"`
		IP       string `json:"ip"`
	} `json:"server"`
	Result struct {
		ID        string `json:"id"`
		URL       string `json:"url"`
		Persisted bool   `json:"persisted"`
	} `json:"result"`
}

// performSpeedtestCheck performs a single speedtest check
func (sm *SpeedtestManager) performSpeedtestCheck(target *speedtestTarget) *system.SpeedtestResult {
	// Build speedtest command
	args := []string{"-f", "json", "--accept-gdpr", "--accept-license"}
	if target.ServerID != "" {
		args = append(args, "--server-id", target.ServerID)
	}

	cmd := exec.Command("speedtest", args...)

	// Set timeout for the command
	ctx, cancel := context.WithTimeout(context.Background(), target.Timeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	// Execute speedtest
	output, err := cmd.CombinedOutput()

	if err != nil {
		return &system.SpeedtestResult{
			ServerURL:     target.ServerID,
			Status:        "error",
			DownloadSpeed: 0,
			UploadSpeed:   0,
			Latency:       0,
			ErrorCode:     fmt.Sprintf("speedtest_failed: %v", err),
			LastChecked:   time.Now(),
		}
	}

	// Parse JSON output
	var cliResult SpeedtestCLIResult
	if err := json.Unmarshal(output, &cliResult); err != nil {
		return &system.SpeedtestResult{
			ServerURL:     target.ServerID,
			Status:        "error",
			DownloadSpeed: 0,
			UploadSpeed:   0,
			Latency:       0,
			ErrorCode:     fmt.Sprintf("json_parse_error: %v", err),
			LastChecked:   time.Now(),
		}
	}

	// Convert bandwidth from bytes per second to Mbps
	downloadMbps := float64(cliResult.Download.Bandwidth) * 8 / 1000000 // Convert to Mbps
	uploadMbps := float64(cliResult.Upload.Bandwidth) * 8 / 1000000     // Convert to Mbps

	return &system.SpeedtestResult{
		ServerURL:     fmt.Sprintf("%d", cliResult.Server.ID), // Use server ID as URL for consistency
		Status:        "success",
		DownloadSpeed: downloadMbps,
		UploadSpeed:   uploadMbps,
		Latency:       cliResult.Ping.Latency,
		ErrorCode:     "",
		LastChecked:   time.Now(),
		// Additional detailed information
		PingJitter:            cliResult.Ping.Jitter,
		PingLow:               cliResult.Ping.Low,
		PingHigh:              cliResult.Ping.High,
		DownloadBytes:         cliResult.Download.Bytes,
		DownloadElapsed:       cliResult.Download.Elapsed,
		DownloadLatencyIQM:    cliResult.Download.Latency.IQM,
		DownloadLatencyLow:    cliResult.Download.Latency.Low,
		DownloadLatencyHigh:   cliResult.Download.Latency.High,
		DownloadLatencyJitter: cliResult.Download.Latency.Jitter,
		UploadBytes:           cliResult.Upload.Bytes,
		UploadElapsed:         cliResult.Upload.Elapsed,
		UploadLatencyIQM:      cliResult.Upload.Latency.IQM,
		UploadLatencyLow:      cliResult.Upload.Latency.Low,
		UploadLatencyHigh:     cliResult.Upload.Latency.High,
		UploadLatencyJitter:   cliResult.Upload.Latency.Jitter,
		PacketLoss:            int(cliResult.PacketLoss),
		ISP:                   cliResult.ISP,
		InterfaceExternalIP:   cliResult.Interface.ExternalIP,
		ServerName:            cliResult.Server.Name,
		ServerLocation:        cliResult.Server.Location,
		ServerCountry:         cliResult.Server.Country,
		ServerHost:            cliResult.Server.Host,
		ServerIP:              cliResult.Server.IP,
	}
}

// Stop stops the speedtest manager
func (sm *SpeedtestManager) Stop() {
	sm.cancel()
	if sm.cronScheduler != nil {
		sm.cronScheduler.Stop()
	}
	slog.Debug("Speedtest manager stopped")
}
