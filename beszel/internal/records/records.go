// Package records handles deleting old records based on retention policy.
package records

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type RecordManager struct {
	app core.App
}

func NewRecordManager(app core.App) *RecordManager {
	return &RecordManager{app}
}

// getRetentionPeriod returns the retention period from environment variable
// Returns error if BESZEL_RETENTION_DAYS is not set or invalid
func (rm *RecordManager) getRetentionPeriod() (time.Duration, error) {
	retentionDays := os.Getenv("BESZEL_RETENTION_DAYS")
	if retentionDays == "" {
		return 0, fmt.Errorf("BESZEL_RETENTION_DAYS environment variable is required")
	}

	days, err := strconv.Atoi(retentionDays)
	if err != nil {
		return 0, fmt.Errorf("Invalid BESZEL_RETENTION_DAYS value: %s", retentionDays)
	}

	if days <= 0 {
		return 0, fmt.Errorf("BESZEL_RETENTION_DAYS must be greater than 0")
	}

	return time.Duration(days) * 24 * time.Hour, nil
}

// Delete old records based on retention policy
func (rm *RecordManager) DeleteOldRecords() {
	retentionPeriod, err := rm.getRetentionPeriod()
	if err != nil {
		// Log info message when retention is not configured
		if err.Error() == "BESZEL_RETENTION_DAYS environment variable is required" {
			fmt.Printf("Info: Data retention not configured, skipping cleanup operation\n")
		} else {
			fmt.Printf("Retention configuration error: %v\n", err)
		}
		return
	}
	cutoffTime := time.Now().UTC().Add(-retentionPeriod)

	rm.app.RunInTransaction(func(txApp core.App) error {
		// Collections to process
		collections := [5]string{"ping_stats", "dns_stats", "http_stats", "speedtest_stats", "system_averages"}

		for _, collection := range collections {
			// Delete records older than the retention period
			rawQuery := fmt.Sprintf("DELETE FROM %s WHERE created < {:cutoff}", collection)
			if _, err := txApp.DB().NewQuery(rawQuery).Bind(dbx.Params{"cutoff": cutoffTime}).Execute(); err != nil {
				return fmt.Errorf("failed to delete from %s: %v", collection, err)
			}
		}

		// Also clean up alerts history
		err := deleteOldAlertsHistory(txApp, 200, 250)
		if err != nil {
			return err
		}

		return nil
	})
}

// Delete old alerts history records
func deleteOldAlertsHistory(app core.App, countToKeep, countBeforeDeletion int) error {
	db := app.DB()

	// Count total records
	var totalCount int
	err := db.NewQuery("SELECT COUNT(*) FROM alerts_history").One(&totalCount)
	if err != nil {
		return err
	}

	// If we have more records than the threshold, delete old ones
	if totalCount > countBeforeDeletion {
		_, err = db.NewQuery("DELETE FROM alerts_history WHERE id NOT IN (SELECT id FROM alerts_history ORDER BY created DESC LIMIT {:countToKeep})").Bind(dbx.Params{"countToKeep": countToKeep}).Execute()
		if err != nil {
			return err
		}
	}
	return nil
}

/* Round float to two decimals */
func twoDecimals(value float64) float64 {
	return math.Round(value*100) / 100
}
