//go:build testing
// +build testing

package records

import (
	"github.com/pocketbase/pocketbase/core"
)


// TestDeleteOldAlertsHistory exposes deleteOldAlertsHistory for testing
func TestDeleteOldAlertsHistory(app core.App, countToKeep, countBeforeDeletion int) error {
	return deleteOldAlertsHistory(app, countToKeep, countBeforeDeletion)
}

// TestTwoDecimals exposes twoDecimals for testing
func TestTwoDecimals(value float64) float64 {
	return twoDecimals(value)
}
