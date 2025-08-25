package agent

import (
	"beszel/internal/entities/system"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type HttpManager struct {
	sync.RWMutex
	targets         map[string]*httpTarget
	results         map[string]*system.HttpResult
	lastResultsTime time.Time
	ctx             context.Context
	cancel          context.CancelFunc
	cronScheduler   *cron.Cron
	cronExpression  string
}

type httpTarget struct {
	URL       string
	Timeout   time.Duration
	lastCheck time.Time
}

// NewHttpManager creates a new HTTP manager
func NewHttpManager() (*HttpManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	hm := &HttpManager{
		targets:        make(map[string]*httpTarget),
		results:        make(map[string]*system.HttpResult),
		ctx:            ctx,
		cancel:         cancel,
		cronScheduler:  cron.New(cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow))),
		cronExpression: "",
	}

	slog.Debug("HTTP manager initialized")

	// Start the cron scheduler
	hm.cronScheduler.Start()

	// Schedule the HTTP job
	hm.scheduleHttpJob()

	return hm, nil
}

// UpdateConfig updates the HTTP configuration with targets and cron expression
func (hm *HttpManager) UpdateConfig(targets []system.HttpTarget, cronExpression string) {
	hm.Lock()
	defer hm.Unlock()

	oldTargetsCount := len(hm.targets)
	oldResultsCount := len(hm.results)
	
	slog.Debug("UpdateConfig called", "old_targets", oldTargetsCount, "new_targets", len(targets), "cron_expression", cronExpression)

	// Use cron expression directly
	hm.cronExpression = cronExpression

	// Clear existing targets and results to prevent stale data
	hm.targets = make(map[string]*httpTarget)
	hm.results = make(map[string]*system.HttpResult)
	
	if oldTargetsCount > 0 || oldResultsCount > 0 {
		slog.Info("Cleared old HTTP configuration", "old_targets", oldTargetsCount, "old_results", oldResultsCount)
	}

	// Add new targets
	for _, target := range targets {
		timeout := target.Timeout
		if timeout <= 0 {
			timeout = 10 // Default 10 seconds
		}

		hm.targets[target.URL] = &httpTarget{
			URL:       target.URL,
			Timeout:   time.Duration(timeout) * time.Second,
			lastCheck: time.Time{}, // Will trigger immediate check
		}
	}

	// Reschedule the HTTP job with new cron expression
	hm.scheduleHttpJob()

	slog.Debug("Updated HTTP config", "targets", len(targets))
}

// GetResults returns the current HTTP results
func (hm *HttpManager) GetResults() map[string]*system.HttpResult {
	hm.Lock()
	defer hm.Unlock()

	// If no results are available, return nil to indicate no HTTP checks have run
	if len(hm.results) == 0 {
		return nil
	}

	// Create a copy to avoid race conditions
	results := make(map[string]*system.HttpResult)
	for url, result := range hm.results {
		results[url] = &system.HttpResult{
			URL:          result.URL,
			Status:       result.Status,
			ResponseTime: result.ResponseTime,
			StatusCode:   result.StatusCode,
			ErrorCode:    result.ErrorCode,
			LastChecked:  result.LastChecked,
		}
	}

	// Clear the results after they've been retrieved
	// This ensures HTTP data is only sent once per test run
	hm.results = make(map[string]*system.HttpResult)

	return results
}

// scheduleHttpJob schedules the HTTP monitoring job
func (hm *HttpManager) scheduleHttpJob() {
	// Remove all existing jobs by creating a new scheduler
	hm.cronScheduler.Stop()
	hm.cronScheduler = cron.New(cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow))) // 5-field format
	hm.cronScheduler.Start()

	// Only schedule if we have a valid cron expression
	if hm.cronExpression != "" {
		_, err := hm.cronScheduler.AddFunc(hm.cronExpression, func() {
			slog.Debug("Running HTTP checks")
			hm.performHttpChecks()
		})
		if err != nil {
			slog.Error("Failed to schedule HTTP job", "cron_expression", hm.cronExpression, "error", err)
		} else {
			slog.Debug("HTTP job scheduled", "expression", hm.cronExpression)
		}
	} else {
		slog.Debug("No cron expression set, HTTP job not scheduled")
	}
}

// performHttpChecks performs HTTP checks for all targets
func (hm *HttpManager) performHttpChecks() {
	hm.RLock()
	targets := make([]*httpTarget, 0, len(hm.targets))
	for _, target := range hm.targets {
		targets = append(targets, target)
	}
	hm.RUnlock()

	slog.Debug("Performing HTTP checks", "targets", len(targets))

	// Check targets concurrently
	var wg sync.WaitGroup
	for _, target := range targets {
		wg.Add(1)
		go func(t *httpTarget) {
			defer wg.Done()
			result := hm.performHttpCheck(t)

			hm.Lock()
			hm.results[t.URL] = result
			hm.lastResultsTime = time.Now()
			hm.Unlock()

			slog.Debug("HTTP check completed",
				"url", t.URL,
				"status", result.Status,
				"response_time", result.ResponseTime,
				"status_code", result.StatusCode)
		}(target)
	}
	wg.Wait()
}

// performHttpCheck performs a single HTTP check
func (hm *HttpManager) performHttpCheck(target *httpTarget) *system.HttpResult {
	startTime := time.Now()

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: target.Timeout,
	}

	// Create request
	req, err := http.NewRequest("GET", target.URL, nil)
	if err != nil {
		return &system.HttpResult{
			URL:          target.URL,
			Status:       "error",
			ResponseTime: 0,
			StatusCode:   0,
			ErrorCode:    fmt.Sprintf("request_error: %v", err),
			LastChecked:  time.Now(),
		}
	}

	// Perform the request
	resp, err := client.Do(req)
	responseTime := time.Since(startTime).Milliseconds()

	if err != nil {
		return &system.HttpResult{
			URL:          target.URL,
			Status:       "error",
			ResponseTime: float64(responseTime),
			StatusCode:   0,
			ErrorCode:    fmt.Sprintf("request_failed: %v", err),
			LastChecked:  time.Now(),
		}
	}
	defer resp.Body.Close()

	// Read response body
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return &system.HttpResult{
			URL:          target.URL,
			Status:       "error",
			ResponseTime: float64(responseTime),
			StatusCode:   resp.StatusCode,
			ErrorCode:    fmt.Sprintf("body_read_error: %v", err),
			LastChecked:  time.Now(),
		}
	}

	// Always consider it successful if we get a response
	status := "success"
	errorCode := ""

	return &system.HttpResult{
		URL:          target.URL,
		Status:       status,
		ResponseTime: float64(responseTime),
		StatusCode:   resp.StatusCode,
		ErrorCode:    errorCode,
		LastChecked:  time.Now(),
	}
}

// Stop stops the HTTP manager
func (hm *HttpManager) Stop() {
	hm.cancel()
	if hm.cronScheduler != nil {
		hm.cronScheduler.Stop()
	}
	slog.Debug("HTTP manager stopped")
}
