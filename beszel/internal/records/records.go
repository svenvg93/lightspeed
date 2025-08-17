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
	cutoffDate := time.Now().UTC().Add(-retentionPeriod)

	// Delete old records from all stats collections using optimized queries
	collections := []string{"ping_stats", "dns_stats", "http_stats", "speedtest_stats", "system_averages"}

	for _, collectionName := range collections {
		if err := rm.deleteOldRecordsFromCollection(collectionName, cutoffDate); err != nil {
			fmt.Printf("Error deleting old records from %s: %v\n", collectionName, err)
		}
	}

	// Clean up alerts history with optimized query
	if err := rm.deleteOldAlertsHistoryOptimized(); err != nil {
		fmt.Printf("Error deleting old alerts history: %v\n", err)
	}
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

// deleteOldRecordsFromCollection deletes old records from a specific collection using direct date comparison
func (rm *RecordManager) deleteOldRecordsFromCollection(collectionName string, cutoffDate time.Time) error {
	db := rm.app.DB()

	// Use direct date comparison for better performance
	query := fmt.Sprintf("DELETE FROM %s WHERE created < {:cutoffDate}", collectionName)

	result, err := db.NewQuery(query).Bind(dbx.Params{"cutoffDate": cutoffDate}).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete old records from %s: %w", collectionName, err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Deleted %d old records from %s\n", rowsAffected, collectionName)

	return nil
}

// deleteOldRecordsPaginated deletes old records in batches to avoid long-running transactions
func (rm *RecordManager) deleteOldRecordsPaginated(collectionName string, cutoffDate time.Time, batchSize int) error {
	db := rm.app.DB()

	for {
		// Delete in batches to avoid long-running transactions
		query := fmt.Sprintf("DELETE FROM %s WHERE created < {:cutoffDate} LIMIT {:batchSize}", collectionName)

		result, err := db.NewQuery(query).Bind(dbx.Params{
			"cutoffDate": cutoffDate,
			"batchSize":  batchSize,
		}).Execute()

		if err != nil {
			return fmt.Errorf("failed to delete old records from %s: %w", collectionName, err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected < int64(batchSize) {
			break // No more records to delete
		}

		fmt.Printf("Deleted batch of %d records from %s\n", rowsAffected, collectionName)

		// Small delay to prevent overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// deleteOldAlertsHistoryOptimized deletes old alerts history records using an optimized query
func (rm *RecordManager) deleteOldAlertsHistoryOptimized() error {
	db := rm.app.DB()

	// Get count to keep from environment or use default
	countToKeep := 1000
	if countStr := os.Getenv("BESZEL_ALERTS_HISTORY_KEEP"); countStr != "" {
		if count, err := strconv.Atoi(countStr); err == nil && count > 0 {
			countToKeep = count
		}
	}

	// Count total records
	var totalCount int
	err := db.NewQuery("SELECT COUNT(*) FROM alerts_history").One(&totalCount)
	if err != nil {
		return fmt.Errorf("failed to count alerts history records: %w", err)
	}

	// If we have more records than the threshold, delete old ones
	if totalCount > countToKeep {
		query := `
			DELETE FROM alerts_history 
			WHERE id NOT IN (
				SELECT id FROM alerts_history 
				ORDER BY created DESC 
				LIMIT {:countToKeep}
			)
		`

		result, err := db.NewQuery(query).Bind(dbx.Params{"countToKeep": countToKeep}).Execute()
		if err != nil {
			return fmt.Errorf("failed to delete old alerts history: %w", err)
		}

		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("Deleted %d old alerts history records\n", rowsAffected)
	}

	return nil
}

// GetDatabaseStats returns statistics about the database collections
func (rm *RecordManager) GetDatabaseStats() (map[string]interface{}, error) {
	db := rm.app.DB()
	stats := make(map[string]interface{})

	// Get record counts for each collection
	collections := []string{"ping_stats", "dns_stats", "http_stats", "speedtest_stats", "alerts_history", "system_averages"}

	for _, collectionName := range collections {
		var count int
		err := db.NewQuery(fmt.Sprintf("SELECT COUNT(*) FROM %s", collectionName)).One(&count)
		if err != nil {
			continue // Skip if collection doesn't exist or error
		}
		stats[collectionName+"_count"] = count
	}

	// Get oldest and newest record dates for each collection
	for _, collectionName := range collections {
		var oldest, newest time.Time

		// Get oldest record
		oldestQuery := fmt.Sprintf("SELECT MIN(created) FROM %s", collectionName)
		if err := db.NewQuery(oldestQuery).One(&oldest); err == nil {
			stats[collectionName+"_oldest"] = oldest
		}

		// Get newest record
		newestQuery := fmt.Sprintf("SELECT MAX(created) FROM %s", collectionName)
		if err := db.NewQuery(newestQuery).One(&newest); err == nil {
			stats[collectionName+"_newest"] = newest
		}
	}

	return stats, nil
}

// CleanupDatabase performs a comprehensive database cleanup with statistics
func (rm *RecordManager) CleanupDatabase() error {
	fmt.Println("Starting comprehensive database cleanup...")

	// Delete old records
	rm.DeleteOldRecords()

	// Get and log database statistics
	stats, err := rm.GetDatabaseStats()
	if err != nil {
		fmt.Printf("Error getting database stats: %v\n", err)
	} else {
		fmt.Println("Database statistics after cleanup:")
		for key, value := range stats {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	fmt.Println("Database cleanup completed successfully")
	return nil
}

/* Round float to two decimals */
func twoDecimals(value float64) float64 {
	return math.Round(value*100) / 100
}
