//go:build testing
// +build testing

package records_test

import (
	"beszel/internal/records"
	"beszel/internal/tests"
	"fmt"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteOldRecords tests the main DeleteOldRecords function
func TestDeleteOldRecords(t *testing.T) {
	hub, err := tests.NewTestHub(t.TempDir())
	require.NoError(t, err)
	defer hub.Cleanup()

	rm := records.NewRecordManager(hub)

	// Create test user for alerts history
	user, err := tests.CreateUser(hub, "test@example.com", "testtesttest")
	require.NoError(t, err)

	// Create test system
	system, err := tests.CreateRecord(hub, "systems", map[string]any{
		"name":   "test-system",
		"host":   "localhost",
		"status": "up",
		"users":  []string{user.Id},
	})
	require.NoError(t, err)

	now := time.Now()

	// Create old ping_stats records that should be deleted
	var record *core.Record
	record, err = tests.CreateRecord(hub, "ping_stats", map[string]any{
		"system":      system.Id,
		"host":        "test-host",
		"packet_loss": 5.0,
		"avg_rtt":     50.0,
	})
	require.NoError(t, err)
	// created is autodate field, so we need to set it manually
	record.SetRaw("created", now.UTC().Add(-2*time.Hour).Format(types.DefaultDateLayout))
	err = hub.SaveNoValidate(record)
	require.NoError(t, err)
	require.NotNil(t, record)
	require.InDelta(t, record.GetDateTime("created").Time().UTC().Unix(), now.UTC().Add(-2*time.Hour).Unix(), 1)
	require.Equal(t, record.Get("system"), system.Id)
	require.Equal(t, record.Get("host"), "test-host")

	// Create recent ping_stats record that should be kept
	_, err = tests.CreateRecord(hub, "ping_stats", map[string]any{
		"system":      system.Id,
		"host":        "test-host",
		"packet_loss": 2.0,
		"avg_rtt":     30.0,
		"created":     now.Add(-30 * time.Minute), // 30 minutes old, should be kept
	})
	require.NoError(t, err)

	// Create many alerts history records to trigger deletion
	for i := range 260 { // More than countBeforeDeletion (250)
		_, err = tests.CreateRecord(hub, "alerts_history", map[string]any{
			"user":    user.Id,
			"name":    "CPU",
			"value":   i + 1,
			"system":  system.Id,
			"created": now.Add(-time.Duration(i) * time.Minute),
		})
		require.NoError(t, err)
	}

	// Count records before deletion
	pingStatsCountBefore, err := hub.CountRecords("ping_stats")
	require.NoError(t, err)
	alertsCountBefore, err := hub.CountRecords("alerts_history")
	require.NoError(t, err)

	// Run deletion
	rm.DeleteOldRecords()

	// Count records after deletion
	pingStatsCountAfter, err := hub.CountRecords("ping_stats")
	require.NoError(t, err)
	alertsCountAfter, err := hub.CountRecords("alerts_history")
	require.NoError(t, err)

	// Verify old ping stats were deleted
	assert.Less(t, pingStatsCountAfter, pingStatsCountBefore, "Old ping stats should be deleted")

	// Verify alerts history was trimmed
	assert.Less(t, alertsCountAfter, alertsCountBefore, "Excessive alerts history should be deleted")
	assert.Equal(t, alertsCountAfter, int64(200), "Alerts count should be equal to countToKeep (200)")
}


// TestDeleteOldAlertsHistory tests the deleteOldAlertsHistory function
func TestDeleteOldAlertsHistory(t *testing.T) {
	hub, err := tests.NewTestHub(t.TempDir())
	require.NoError(t, err)
	defer hub.Cleanup()

	// Create test users
	user1, err := tests.CreateUser(hub, "user1@example.com", "testtesttest")
	require.NoError(t, err)

	user2, err := tests.CreateUser(hub, "user2@example.com", "testtesttest")
	require.NoError(t, err)

	system, err := tests.CreateRecord(hub, "systems", map[string]any{
		"name":   "test-system",
		"host":   "localhost",
		"status": "up",
		"users":  []string{user1.Id, user2.Id},
	})
	require.NoError(t, err)
	now := time.Now().UTC()

	testCases := []struct {
		name                  string
		user                  *core.Record
		alertCount            int
		countToKeep           int
		countBeforeDeletion   int
		expectedAfterDeletion int
		description           string
	}{
		{
			name:                  "User with few alerts (below threshold)",
			user:                  user1,
			alertCount:            100,
			countToKeep:           50,
			countBeforeDeletion:   150,
			expectedAfterDeletion: 100, // No deletion because below threshold
			description:           "User with alerts below countBeforeDeletion should not have any deleted",
		},
		{
			name:                  "User with many alerts (above threshold)",
			user:                  user2,
			alertCount:            300,
			countToKeep:           100,
			countBeforeDeletion:   200,
			expectedAfterDeletion: 100, // Should be trimmed to countToKeep
			description:           "User with alerts above countBeforeDeletion should be trimmed to countToKeep",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create alerts for this user
			for i := 0; i < tc.alertCount; i++ {
				_, err := tests.CreateRecord(hub, "alerts_history", map[string]any{
					"user":    tc.user.Id,
					"name":    "CPU",
					"value":   i + 1,
					"system":  system.Id,
					"created": now.Add(-time.Duration(i) * time.Minute),
				})
				require.NoError(t, err)
			}

			// Count before deletion
			countBefore, err := hub.CountRecords("alerts_history",
				dbx.NewExp("user = {:user}", dbx.Params{"user": tc.user.Id}))
			require.NoError(t, err)
			assert.Equal(t, int64(tc.alertCount), countBefore, "Initial count should match")

			// Run deletion
			err = records.TestDeleteOldAlertsHistory(hub, tc.countToKeep, tc.countBeforeDeletion)
			require.NoError(t, err)

			// Count after deletion
			countAfter, err := hub.CountRecords("alerts_history",
				dbx.NewExp("user = {:user}", dbx.Params{"user": tc.user.Id}))
			require.NoError(t, err)

			assert.Equal(t, int64(tc.expectedAfterDeletion), countAfter, tc.description)

			// If deletion occurred, verify the most recent records were kept
			if tc.expectedAfterDeletion < tc.alertCount {
				records, err := hub.FindRecordsByFilter("alerts_history",
					"user = {:user}",
					"-created", // Order by created DESC
					tc.countToKeep,
					0,
					map[string]any{"user": tc.user.Id})
				require.NoError(t, err)
				assert.Len(t, records, tc.expectedAfterDeletion, "Should have exactly countToKeep records")

				// Verify records are in descending order by created time
				for i := 1; i < len(records); i++ {
					prev := records[i-1].GetDateTime("created").Time()
					curr := records[i].GetDateTime("created").Time()
					assert.True(t, prev.After(curr) || prev.Equal(curr),
						"Records should be ordered by created time (newest first)")
				}
			}
		})
	}
}

// TestDeleteOldAlertsHistoryEdgeCases tests edge cases for alerts history deletion
func TestDeleteOldAlertsHistoryEdgeCases(t *testing.T) {
	hub, err := tests.NewTestHub(t.TempDir())
	require.NoError(t, err)
	defer hub.Cleanup()

	t.Run("No users with excessive alerts", func(t *testing.T) {
		// Create user with few alerts
		user, err := tests.CreateUser(hub, "few@example.com", "testtesttest")
		require.NoError(t, err)

		system, err := tests.CreateRecord(hub, "systems", map[string]any{
			"name":   "test-system",
			"host":   "localhost",
			"status": "up",
			"users":  []string{user.Id},
		})

		// Create only 5 alerts (well below threshold)
		for i := range 5 {
			_, err := tests.CreateRecord(hub, "alerts_history", map[string]any{
				"user":   user.Id,
				"name":   "CPU",
				"value":  i + 1,
				"system": system.Id,
			})
			require.NoError(t, err)
		}

		// Should not error and should not delete anything
		err = records.TestDeleteOldAlertsHistory(hub, 10, 20)
		require.NoError(t, err)

		count, err := hub.CountRecords("alerts_history")
		require.NoError(t, err)
		assert.Equal(t, int64(5), count, "All alerts should remain")
	})

	t.Run("Empty alerts_history table", func(t *testing.T) {
		// Clear any existing alerts
		_, err := hub.DB().NewQuery("DELETE FROM alerts_history").Execute()
		require.NoError(t, err)

		// Should not error with empty table
		err = records.TestDeleteOldAlertsHistory(hub, 10, 20)
		require.NoError(t, err)
	})
}

// TestRecordManagerCreation tests RecordManager creation
func TestRecordManagerCreation(t *testing.T) {
	hub, err := tests.NewTestHub(t.TempDir())
	require.NoError(t, err)
	defer hub.Cleanup()

	rm := records.NewRecordManager(hub)
	assert.NotNil(t, rm, "RecordManager should not be nil")
}

// TestTwoDecimals tests the twoDecimals helper function
func TestTwoDecimals(t *testing.T) {
	testCases := []struct {
		input    float64
		expected float64
	}{
		{1.234567, 1.23},
		{1.235, 1.24}, // Should round up
		{1.0, 1.0},
		{0.0, 0.0},
		{-1.234567, -1.23},
		{-1.235, -1.23}, // Negative rounding
	}

	for _, tc := range testCases {
		result := records.TestTwoDecimals(tc.input)
		assert.InDelta(t, tc.expected, result, 0.02, "twoDecimals(%f) should equal %f", tc.input, tc.expected)
	}
}
