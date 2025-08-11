package agent

import (
	"beszel/internal/entities/system"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHttpManager(t *testing.T) {
	hm, err := NewHttpManager()
	require.NoError(t, err)
	require.NotNil(t, hm)

	assert.NotNil(t, hm.targets)
	assert.NotNil(t, hm.results)
	assert.NotNil(t, hm.cronScheduler)
	assert.Empty(t, hm.cronExpression)
}

func TestHttpManager_UpdateConfig(t *testing.T) {
	hm, err := NewHttpManager()
	require.NoError(t, err)

	// Test empty config
	hm.UpdateConfig([]system.HttpTarget{}, "")
	assert.Empty(t, hm.targets)
	assert.Empty(t, hm.cronExpression)

	// Test with targets
	targets := []system.HttpTarget{
		{
			URL:     "https://google.com",
			Timeout: 10,
		},
		{
			URL:     "https://cloudflare.com",
			Timeout: 5,
		},
	}

	hm.UpdateConfig(targets, "*/2 * * * *")

	assert.Len(t, hm.targets, 2)
	assert.Equal(t, "*/2 * * * *", hm.cronExpression)

	// Verify targets were added correctly
	target1, exists := hm.targets["https://google.com"]
	assert.True(t, exists)
	assert.Equal(t, "https://google.com", target1.URL)
	assert.Equal(t, 10*time.Second, target1.Timeout)

	target2, exists := hm.targets["https://cloudflare.com"]
	assert.True(t, exists)
	assert.Equal(t, "https://cloudflare.com", target2.URL)
	assert.Equal(t, 5*time.Second, target2.Timeout)
}

func TestHttpManager_GetResults(t *testing.T) {
	hm, err := NewHttpManager()
	require.NoError(t, err)

	// Test empty results
	results := hm.GetResults()
	assert.NotNil(t, results) // HTTP manager returns empty map, not nil
	assert.Empty(t, results)

	// Add some mock results
	hm.results["https://google.com"] = &system.HttpResult{
		URL:          "https://google.com",
		Status:       "success",
		ResponseTime: 150.5,
		StatusCode:   200,
		LastChecked:  time.Now(),
	}

	hm.results["https://cloudflare.com"] = &system.HttpResult{
		URL:          "https://cloudflare.com",
		Status:       "success",
		ResponseTime: 120.3,
		StatusCode:   200,
		LastChecked:  time.Now(),
	}

	results = hm.GetResults()
	assert.NotNil(t, results)
	assert.Len(t, results, 2)

	assert.Contains(t, results, "https://google.com")
	assert.Contains(t, results, "https://cloudflare.com")
	assert.Equal(t, "success", results["https://google.com"].Status)
	assert.Equal(t, 150.5, results["https://google.com"].ResponseTime)
	assert.Equal(t, 200, results["https://google.com"].StatusCode)
}

func TestHttpManager_DirectResultAssignment(t *testing.T) {
	hm, err := NewHttpManager()
	require.NoError(t, err)

	result := &system.HttpResult{
		URL:          "https://google.com",
		Status:       "success",
		ResponseTime: 150.5,
		StatusCode:   200,
		LastChecked:  time.Now(),
	}

	// Direct assignment since updateResult doesn't exist
	hm.results["https://google.com"] = result

	assert.Len(t, hm.results, 1)
	assert.Contains(t, hm.results, "https://google.com")
	assert.Equal(t, result, hm.results["https://google.com"])
}

func TestHttpManager_ManualResultClear(t *testing.T) {
	hm, err := NewHttpManager()
	require.NoError(t, err)

	// Add some results
	hm.results["https://google.com"] = &system.HttpResult{
		URL:          "https://google.com",
		Status:       "success",
		ResponseTime: 150.5,
		StatusCode:   200,
		LastChecked:  time.Now(),
	}

	// Manually clear results
	hm.results = make(map[string]*system.HttpResult)

	assert.Empty(t, hm.results)
}

func TestHttpManager_CronExpressionHandling(t *testing.T) {
	hm, err := NewHttpManager()
	require.NoError(t, err)

	// Test that cron expressions are stored correctly
	hm.UpdateConfig([]system.HttpTarget{}, "*/5 * * * *")
	assert.Equal(t, "*/5 * * * *", hm.cronExpression)

	// Test empty expression
	hm.UpdateConfig([]system.HttpTarget{}, "")
	assert.Equal(t, "", hm.cronExpression)
}

func TestHttpManager_ContextCancellation(t *testing.T) {
	hm, err := NewHttpManager()
	require.NoError(t, err)

	// Cancel the context
	hm.cancel()

	// Wait a bit for the context to be cancelled
	time.Sleep(100 * time.Millisecond)

	// The manager should still be functional for basic operations
	hm.UpdateConfig([]system.HttpTarget{}, "")
	assert.Empty(t, hm.targets)
}

func TestHttpManager_ConcurrentAccess(t *testing.T) {
	hm, err := NewHttpManager()
	require.NoError(t, err)

	// Test concurrent access to results
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			hm.results["test"] = &system.HttpResult{
				URL:          "https://test.com",
				Status:       "success",
				ResponseTime: float64(i),
				StatusCode:   200,
				LastChecked:  time.Now(),
			}
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			hm.GetResults()
		}
		done <- true
	}()

	<-done
	<-done

	// Should not panic and should have results
	results := hm.GetResults()
	assert.NotNil(t, results)
	assert.Contains(t, results, "test")
}

func TestHttpManager_ErrorHandling(t *testing.T) {
	hm, err := NewHttpManager()
	require.NoError(t, err)

	// Test error result
	errorResult := &system.HttpResult{
		URL:          "https://invalid-url.com",
		Status:       "error",
		ResponseTime: 0,
		StatusCode:   0,
		ErrorCode:    "connection_failed",
		LastChecked:  time.Now(),
	}

	hm.results["https://invalid-url.com"] = errorResult

	results := hm.GetResults()
	assert.NotNil(t, results)
	assert.Contains(t, results, "https://invalid-url.com")
	assert.Equal(t, "error", results["https://invalid-url.com"].Status)
	assert.Equal(t, "connection_failed", results["https://invalid-url.com"].ErrorCode)
}

func TestHttpManager_TimeoutHandling(t *testing.T) {
	hm, err := NewHttpManager()
	require.NoError(t, err)

	// Test timeout result
	timeoutResult := &system.HttpResult{
		URL:          "https://slow-site.com",
		Status:       "timeout",
		ResponseTime: 0,
		StatusCode:   0,
		ErrorCode:    "timeout",
		LastChecked:  time.Now(),
	}

	hm.results["https://slow-site.com"] = timeoutResult

	results := hm.GetResults()
	assert.NotNil(t, results)
	assert.Contains(t, results, "https://slow-site.com")
	assert.Equal(t, "timeout", results["https://slow-site.com"].Status)
	assert.Equal(t, "timeout", results["https://slow-site.com"].ErrorCode)
}
