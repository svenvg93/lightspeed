package agent

import (
	"beszel/internal/entities/system"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPingManager(t *testing.T) {
	pm, err := NewPingManager()
	require.NoError(t, err)
	require.NotNil(t, pm)

	assert.NotNil(t, pm.targets)
	assert.NotNil(t, pm.results)
	assert.NotNil(t, pm.cronScheduler)
	assert.Empty(t, pm.cronExpression)
}

func TestPingManager_UpdateConfig(t *testing.T) {
	pm, err := NewPingManager()
	require.NoError(t, err)

	// Test empty config
	pm.UpdateConfig([]system.PingTarget{}, "")
	assert.Empty(t, pm.targets)
	assert.Empty(t, pm.cronExpression)

	// Test with targets
	targets := []system.PingTarget{
		{
			Host:    "8.8.8.8",
			Count:   4,
			Timeout: 5 * time.Second,
		},
		{
			Host:    "1.1.1.1",
			Count:   3,
			Timeout: 3 * time.Second,
		},
	}

	pm.UpdateConfig(targets, "*/2 * * * *")

	assert.Len(t, pm.targets, 2)
	assert.Equal(t, "*/2 * * * *", pm.cronExpression)

	// Verify targets were added correctly
	target1, exists := pm.targets["8.8.8.8"]
	assert.True(t, exists)
	assert.Equal(t, "8.8.8.8", target1.Host)
	assert.Equal(t, 4, target1.Count)
	assert.Equal(t, 5*time.Second, target1.Timeout)

	target2, exists := pm.targets["1.1.1.1"]
	assert.True(t, exists)
	assert.Equal(t, "1.1.1.1", target2.Host)
	assert.Equal(t, 3, target2.Count)
	assert.Equal(t, 3*time.Second, target2.Timeout)
}

func TestPingManager_GetResults(t *testing.T) {
	pm, err := NewPingManager()
	require.NoError(t, err)

	// Test empty results
	results := pm.GetResults()
	assert.Nil(t, results)

	// Add some mock results
	pm.results["8.8.8.8"] = &system.PingResult{
		Host:        "8.8.8.8",
		AvgRtt:      15.5,
		MinRtt:      12.0,
		MaxRtt:      18.0,
		PacketLoss:  0.0,
		LastChecked: time.Now(),
	}

	pm.results["1.1.1.1"] = &system.PingResult{
		Host:        "1.1.1.1",
		AvgRtt:      12.3,
		MinRtt:      10.0,
		MaxRtt:      15.0,
		PacketLoss:  0.0,
		LastChecked: time.Now(),
	}

	// Set lastResultsTime to avoid expiration
	pm.lastResultsTime = time.Now()

	results = pm.GetResults()
	assert.NotNil(t, results)
	assert.Len(t, results, 2)

	assert.Contains(t, results, "8.8.8.8")
	assert.Contains(t, results, "1.1.1.1")
	assert.Equal(t, 15.5, results["8.8.8.8"].AvgRtt)
	assert.Equal(t, 12.3, results["1.1.1.1"].AvgRtt)
}

func TestPingManager_GetResults_Expired(t *testing.T) {
	pm, err := NewPingManager()
	require.NoError(t, err)

	// Add old results (more than 5 minutes old)
	pm.results["8.8.8.8"] = &system.PingResult{
		Host:        "8.8.8.8",
		AvgRtt:      15.5,
		MinRtt:      12.0,
		MaxRtt:      18.0,
		PacketLoss:  0.0,
		LastChecked: time.Now().Add(-6 * time.Minute),
	}
	pm.lastResultsTime = time.Now().Add(-6 * time.Minute)

	// Results should be expired and return nil
	results := pm.GetResults()
	assert.Nil(t, results)
	assert.Empty(t, pm.results)
}

func TestPingManager_UpdateResult(t *testing.T) {
	pm, err := NewPingManager()
	require.NoError(t, err)

	result := &system.PingResult{
		Host:        "8.8.8.8",
		AvgRtt:      15.5,
		MinRtt:      12.0,
		MaxRtt:      18.0,
		PacketLoss:  0.0,
		LastChecked: time.Now(),
	}

	pm.updateResult("8.8.8.8", result)

	assert.Len(t, pm.results, 1)
	assert.Contains(t, pm.results, "8.8.8.8")
	assert.Equal(t, result, pm.results["8.8.8.8"])
	assert.False(t, pm.lastResultsTime.IsZero())
}

func TestPingManager_ManualResultClear(t *testing.T) {
	pm, err := NewPingManager()
	require.NoError(t, err)

	// Add some results
	pm.results["8.8.8.8"] = &system.PingResult{
		Host:        "8.8.8.8",
		AvgRtt:      15.5,
		MinRtt:      12.0,
		MaxRtt:      18.0,
		PacketLoss:  0.0,
		LastChecked: time.Now(),
	}
	pm.lastResultsTime = time.Now()

	// Manually clear results
	pm.results = make(map[string]*system.PingResult)
	pm.lastResultsTime = time.Time{}

	assert.Empty(t, pm.results)
	assert.True(t, pm.lastResultsTime.IsZero())
}

func TestPingManager_CronExpressionHandling(t *testing.T) {
	pm, err := NewPingManager()
	require.NoError(t, err)

	// Test that cron expressions are stored correctly
	pm.UpdateConfig([]system.PingTarget{}, "*/5 * * * *")
	assert.Equal(t, "*/5 * * * *", pm.cronExpression)

	// Test empty expression
	pm.UpdateConfig([]system.PingTarget{}, "")
	assert.Equal(t, "", pm.cronExpression)
}

func TestPingManager_ContextCancellation(t *testing.T) {
	pm, err := NewPingManager()
	require.NoError(t, err)

	// Cancel the context
	pm.cancel()

	// Wait a bit for the context to be cancelled
	time.Sleep(100 * time.Millisecond)

	// The manager should still be functional for basic operations
	pm.UpdateConfig([]system.PingTarget{}, "")
	assert.Empty(t, pm.targets)
}

func TestPingManager_ConcurrentAccess(t *testing.T) {
	pm, err := NewPingManager()
	require.NoError(t, err)

	// Test concurrent access to results
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			pm.updateResult("test", &system.PingResult{
				Host:        "test",
				AvgRtt:      float64(i),
				MinRtt:      float64(i) - 1,
				MaxRtt:      float64(i) + 1,
				PacketLoss:  0.0,
				LastChecked: time.Now(),
			})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			pm.GetResults()
		}
		done <- true
	}()

	<-done
	<-done

	// Should not panic and should have results
	results := pm.GetResults()
	assert.NotNil(t, results)
	assert.Contains(t, results, "test")
}
