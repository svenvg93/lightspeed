package agent

import (
	"beszel/internal/entities/system"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDnsManager(t *testing.T) {
	dm, err := NewDnsManager()
	require.NoError(t, err)
	require.NotNil(t, dm)

	assert.NotNil(t, dm.targets)
	assert.NotNil(t, dm.results)
	assert.NotNil(t, dm.cronScheduler)
	assert.Empty(t, dm.cronExpression)
}

func TestDnsManager_UpdateConfig(t *testing.T) {
	dm, err := NewDnsManager()
	require.NoError(t, err)

	// Test empty config
	dm.UpdateConfig([]system.DnsTarget{}, "")
	assert.Empty(t, dm.targets)
	assert.Empty(t, dm.cronExpression)

	// Test with targets
	targets := []system.DnsTarget{
		{
			Domain:   "google.com",
			Server:   "8.8.8.8",
			Type:     "A",
			Timeout:  5 * time.Second,
			Protocol: "udp",
		},
		{
			Domain:   "cloudflare.com",
			Server:   "1.1.1.1",
			Type:     "AAAA",
			Timeout:  3 * time.Second,
			Protocol: "tcp",
		},
	}

	dm.UpdateConfig(targets, "*/2 * * * *")

	assert.Len(t, dm.targets, 2)
	assert.Equal(t, "*/2 * * * *", dm.cronExpression)

	// Verify targets were added correctly
	target1, exists := dm.targets["google.com@8.8.8.8#A"]
	assert.True(t, exists)
	assert.Equal(t, "google.com", target1.Domain)
	assert.Equal(t, "8.8.8.8", target1.Server)
	assert.Equal(t, "A", target1.Type)
	assert.Equal(t, 5*time.Second, target1.Timeout)
	assert.Equal(t, "udp", target1.Protocol)

	target2, exists := dm.targets["cloudflare.com@1.1.1.1#AAAA"]
	assert.True(t, exists)
	assert.Equal(t, "cloudflare.com", target2.Domain)
	assert.Equal(t, "1.1.1.1", target2.Server)
	assert.Equal(t, "AAAA", target2.Type)
	assert.Equal(t, 3*time.Second, target2.Timeout)
	assert.Equal(t, "tcp", target2.Protocol)
}

func TestDnsManager_GetResults(t *testing.T) {
	dm, err := NewDnsManager()
	require.NoError(t, err)

	// Test empty results
	results := dm.GetResults()
	assert.Nil(t, results)

	// Add some mock results
	dm.results["google.com@8.8.8.8#A"] = &system.DnsResult{
		Domain:      "google.com",
		Server:      "8.8.8.8",
		Type:        "A",
		Status:      "success",
		LookupTime:  15.5,
		LastChecked: time.Now(),
	}

	dm.results["cloudflare.com@1.1.1.1#AAAA"] = &system.DnsResult{
		Domain:      "cloudflare.com",
		Server:      "1.1.1.1",
		Type:        "AAAA",
		Status:      "success",
		LookupTime:  12.3,
		LastChecked: time.Now(),
	}

	results = dm.GetResults()
	assert.NotNil(t, results)
	assert.Len(t, results, 2)

	assert.Contains(t, results, "google.com@8.8.8.8#A")
	assert.Contains(t, results, "cloudflare.com@1.1.1.1#AAAA")
	assert.Equal(t, "success", results["google.com@8.8.8.8#A"].Status)
	assert.Equal(t, 15.5, results["google.com@8.8.8.8#A"].LookupTime)
}

func TestDnsManager_GetResults_Empty(t *testing.T) {
	dm, err := NewDnsManager()
	require.NoError(t, err)

	// Test with empty results
	results := dm.GetResults()
	assert.Nil(t, results)
	assert.Empty(t, dm.results)
}

func TestDnsManager_UpdateResult(t *testing.T) {
	dm, err := NewDnsManager()
	require.NoError(t, err)

	result := &system.DnsResult{
		Domain:      "google.com",
		Server:      "8.8.8.8",
		Type:        "A",
		Status:      "success",
		LookupTime:  15.5,
		LastChecked: time.Now(),
	}

	dm.updateResult("google.com@8.8.8.8#A", result)

	assert.Len(t, dm.results, 1)
	assert.Contains(t, dm.results, "google.com@8.8.8.8#A")
	assert.Equal(t, result, dm.results["google.com@8.8.8.8#A"])
}

func TestDnsManager_ManualResultClear(t *testing.T) {
	dm, err := NewDnsManager()
	require.NoError(t, err)

	// Add some results
	dm.results["google.com@8.8.8.8#A"] = &system.DnsResult{
		Domain:      "google.com",
		Server:      "8.8.8.8",
		Type:        "A",
		Status:      "success",
		LookupTime:  15.5,
		LastChecked: time.Now(),
	}

	// Manually clear results
	dm.results = make(map[string]*system.DnsResult)

	assert.Empty(t, dm.results)
}

func TestDnsManager_CronExpressionHandling(t *testing.T) {
	dm, err := NewDnsManager()
	require.NoError(t, err)

	// Test that cron expressions are stored correctly
	dm.UpdateConfig([]system.DnsTarget{}, "*/5 * * * *")
	assert.Equal(t, "*/5 * * * *", dm.cronExpression)

	// Test empty expression
	dm.UpdateConfig([]system.DnsTarget{}, "")
	assert.Equal(t, "", dm.cronExpression)
}

func TestDnsManager_ContextCancellation(t *testing.T) {
	dm, err := NewDnsManager()
	require.NoError(t, err)

	// Cancel the context
	dm.cancel()

	// Wait a bit for the context to be cancelled
	time.Sleep(100 * time.Millisecond)

	// The manager should still be functional for basic operations
	dm.UpdateConfig([]system.DnsTarget{}, "")
	assert.Empty(t, dm.targets)
}

func TestDnsManager_ConcurrentAccess(t *testing.T) {
	dm, err := NewDnsManager()
	require.NoError(t, err)

	// Test concurrent access to results
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			dm.updateResult("test", &system.DnsResult{
				Domain:      "test.com",
				Server:      "8.8.8.8",
				Type:        "A",
				Status:      "success",
				LookupTime:  float64(i),
				LastChecked: time.Now(),
			})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			dm.GetResults()
		}
		done <- true
	}()

	<-done
	<-done

	// Should not panic and should have results
	results := dm.GetResults()
	assert.NotNil(t, results)
	assert.Contains(t, results, "test")
}
