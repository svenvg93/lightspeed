package agent

import (
	"beszel/internal/entities/system"
	"context"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type PingManager struct {
	sync.RWMutex
	targets         map[string]*pingTarget
	results         map[string]*system.PingResult
	lastResultsTime time.Time // Track when results were last updated
	ctx             context.Context
	cancel          context.CancelFunc
	cronScheduler   *cron.Cron
	cronExpression  string // Cron expression for ping scheduling
}

type pingTarget struct {
	system.PingTarget
	lastPing time.Time
}

// NewPingManager creates a new ping manager
func NewPingManager() (*PingManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	pm := &PingManager{
		targets:        make(map[string]*pingTarget),
		results:        make(map[string]*system.PingResult),
		ctx:            ctx,
		cancel:         cancel,
		cronScheduler:  cron.New(cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow))), // 5-field format
		cronExpression: "",                                                                                                    // Will be set by hub configuration (5-field format: minute hour day month weekday)
	}

	slog.Debug("Ping manager initialized")

	// Start the cron scheduler
	pm.cronScheduler.Start()

	// Schedule the ping job
	pm.schedulePingJob()

	return pm, nil
}

// UpdateConfig updates the ping configuration with targets and cron expression
func (pm *PingManager) UpdateConfig(targets []system.PingTarget, cronExpression string) {
	pm.Lock()
	defer pm.Unlock()

	slog.Debug("UpdateConfig called", "targets_count", len(targets), "cron_expression", cronExpression)

	// Use cron expression directly - the cron library supports both 5-field and 6-field formats
	pm.cronExpression = cronExpression

	// Clear existing targets
	pm.targets = make(map[string]*pingTarget)
	pm.results = make(map[string]*system.PingResult)

	// Add new targets
	for _, target := range targets {
		if target.Count <= 0 {
			target.Count = 3
		}
		if target.Timeout <= 0 {
			target.Timeout = 5 * time.Second
		}

		pm.targets[target.Host] = &pingTarget{
			PingTarget: target,
			lastPing:   time.Time{}, // Will trigger immediate ping
		}

	}

	// Reschedule the ping job with new cron expression
	pm.schedulePingJob()

	slog.Debug("Updated ping config", "targets", len(targets))
}

// GetResults returns the current ping results and keeps them available for a reasonable period
// Returns nil if no results are available or if results are too old
func (pm *PingManager) GetResults() map[string]*system.PingResult {
	pm.Lock()
	defer pm.Unlock()

	// If no results are available, return nil to indicate no ping tests have run
	if len(pm.results) == 0 {

		return nil
	}

	// Check if results are too old (more than 5 minutes)
	if time.Since(pm.lastResultsTime) > 5*time.Minute {

		pm.results = make(map[string]*system.PingResult)
		return nil
	}

	// Create a copy to avoid race conditions
	results := make(map[string]*system.PingResult)
	for host, result := range pm.results {
		results[host] = &system.PingResult{
			Host:        result.Host,
			PacketLoss:  result.PacketLoss,
			MinRtt:      result.MinRtt,
			MaxRtt:      result.MaxRtt,
			AvgRtt:      result.AvgRtt,
			LastChecked: result.LastChecked,
		}
	}

	return results
}

// Close shuts down the ping manager
func (pm *PingManager) Close() {
	pm.cronScheduler.Stop()
	pm.cancel()
}

// schedulePingJob schedules the ping job with the current cron expression
func (pm *PingManager) schedulePingJob() {
	// Remove all existing jobs
	pm.cronScheduler.Stop()
	pm.cronScheduler = cron.New(cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow))) // 5-field format
	pm.cronScheduler.Start()

	// Only schedule if we have a valid cron expression
	if pm.cronExpression != "" {
		_, err := pm.cronScheduler.AddFunc(pm.cronExpression, func() {
			slog.Debug("Running ping tests")
			pm.checkPings()
		})
		if err != nil {
			slog.Error("Failed to schedule ping job", "cron_expression", pm.cronExpression, "error", err)
		} else {
			slog.Debug("Scheduled ping job")
		}
	} else {
		slog.Debug("No cron expression set, ping job not scheduled")
	}
}

// checkPings checks if any targets need to be pinged
func (pm *PingManager) checkPings() {
	pm.RLock()
	targets := make([]*pingTarget, 0, len(pm.targets))
	for _, target := range pm.targets {
		targets = append(targets, target)
	}
	pm.RUnlock()

	// Ping targets concurrently
	var wg sync.WaitGroup
	for _, target := range targets {
		wg.Add(1)
		go func(t *pingTarget) {
			defer wg.Done()
			pm.pingTarget(t)
		}(target)
	}
	wg.Wait()
}

// pingTarget performs a ping test to a specific target using pro-bing
func (pm *PingManager) pingTarget(target *pingTarget) {
	pm.Lock()
	target.lastPing = time.Now()
	pm.Unlock()

	result := &system.PingResult{
		Host:        target.Host,
		LastChecked: time.Now(),
	}

	pm.fping(target, result)
}

// fping performs a ping test using fping command
func (pm *PingManager) fping(target *pingTarget, result *system.PingResult) {

	// Build fping command with options
	// -c: count of pings
	// -t: timeout in milliseconds (default is 500ms per ping)
	// -q: quiet mode (only summary output)
	timeoutMs := int(target.Timeout.Milliseconds())
	if timeoutMs < 1000 {
		timeoutMs = 1000 // Minimum 1 second timeout
	}
	args := []string{"-c", strconv.Itoa(target.Count), "-t", strconv.Itoa(timeoutMs), "-q", target.Host}

	cmd := exec.Command("fping", args...)

	// Set timeout for the entire command - give fping enough time to complete
	ctx, cancel := context.WithTimeout(context.Background(), target.Timeout*time.Duration(target.Count)+10*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	// Execute fping
	output, _ := cmd.CombinedOutput()
	outputStr := string(output)

	// fping returns non-zero exit code even on successful pings, so we always parse output
	pm.parseFpingOutput(target.Host, outputStr, result)
}

// parseFpingOutput parses fping output and updates the result
func (pm *PingManager) parseFpingOutput(host, output string, result *system.PingResult) {
	// fping output format: host : xmt/rcv/%loss = 4/4/0%, min/avg/max = 8.91/9.01/9.12
	// or: host : xmt/rcv/%loss = 4/0/100%, min/avg/max = 0/0/0

	// If output is empty, skip this result
	if strings.TrimSpace(output) == "" {
		return
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, host) && strings.Contains(line, "xmt/rcv/%loss") {
			// Extract statistics
			statsRegex := regexp.MustCompile(`xmt/rcv/%loss = (\d+)/(\d+)/(\d+)%`)
			statsMatch := statsRegex.FindStringSubmatch(line)

			if len(statsMatch) >= 4 {
				packetsRecv, _ := strconv.Atoi(statsMatch[2])
				packetLoss, _ := strconv.Atoi(statsMatch[3])

				result.PacketLoss = float64(packetLoss)

				if packetsRecv > 0 {
					// Extract RTT statistics
					rttRegex := regexp.MustCompile(`min/avg/max = ([\d.]+)/([\d.]+)/([\d.]+)`)
					rttMatch := rttRegex.FindStringSubmatch(line)

					if len(rttMatch) >= 4 {
						minRtt, _ := strconv.ParseFloat(rttMatch[1], 64)
						avgRtt, _ := strconv.ParseFloat(rttMatch[2], 64)
						maxRtt, _ := strconv.ParseFloat(rttMatch[3], 64)

						result.MinRtt = minRtt
						result.AvgRtt = avgRtt
						result.MaxRtt = maxRtt
					}

					slog.Debug("fping completed", "host", host, "avg_rtt", result.AvgRtt)
					pm.updateResult(host, result)
				}
				return
			}
		}
	}
}

// updateResult updates the ping result for a host
func (pm *PingManager) updateResult(host string, result *system.PingResult) {
	pm.Lock()
	defer pm.Unlock()

	pm.results[host] = result
	pm.lastResultsTime = time.Now() // Update the timestamp when results are modified

}
