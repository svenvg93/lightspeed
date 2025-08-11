package alerts

import (
	"beszel/internal/entities/system"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/spf13/cast"
)

func (am *AlertManager) HandleSystemAlerts(systemRecord *core.Record, data *system.CombinedData) error {
	alertRecords, err := am.hub.FindAllRecords("alerts",
		dbx.NewExp("system={:system} AND name!='Status'", dbx.Params{"system": systemRecord.Id}),
	)
	if err != nil || len(alertRecords) == 0 {
		// log.Println("no alerts found for system")
		return nil
	}

	var validAlerts []SystemAlertData
	now := systemRecord.GetDateTime("updated").Time().UTC()
	oldestTime := now

	for _, alertRecord := range alertRecords {
		name := alertRecord.GetString("name")
		var val float64
		unit := "%"

		switch name {
		case "Ping":
			// Check ping results for failures (high packet loss or no response)
			if data.Stats.PingResults != nil {
				var failedPings []string
				var totalPings int
				for host, result := range data.Stats.PingResults {
					totalPings++
					if result.PacketLoss >= 100 || result.AvgRtt == 0 {
						failedPings = append(failedPings, host)
					}
				}
				if totalPings > 0 {
					val = float64(len(failedPings)) / float64(totalPings) * 100
					unit = "% failed"
				} else {
					continue
				}
			} else {
				continue
			}
		default:
			// No other metrics are collected anymore, skip all other alerts
			continue
		}

		triggered := alertRecord.GetBool("triggered")
		threshold := alertRecord.GetFloat("value")

		// CONTINUE
		// IF alert is not triggered and curValue is less than threshold
		// OR alert is triggered and curValue is greater than threshold
		if (!triggered && val <= threshold) || (triggered && val > threshold) {
			// log.Printf("Skipping alert %s: val %f | threshold %f | triggered %v\n", name, val, threshold, triggered)
			continue
		}

		min := max(1, cast.ToUint8(alertRecord.Get("min")))

		alert := SystemAlertData{
			systemRecord: systemRecord,
			alertRecord:  alertRecord,
			name:         name,
			unit:         unit,
			val:          val,
			threshold:    threshold,
			triggered:    triggered,
			min:          min,
		}

		// send alert immediately if min is 1 - no need to sum up values.
		if min == 1 {
			alert.triggered = val > threshold
			go am.sendSystemAlert(alert)
			continue
		}

		alert.time = now.Add(-time.Duration(min) * time.Minute)
		if alert.time.Before(oldestTime) {
			oldestTime = alert.time
		}

		validAlerts = append(validAlerts, alert)
	}

	systemStats := []struct {
		Stats   []byte         `db:"stats"`
		Created types.DateTime `db:"created"`
	}{}

	err = am.hub.DB().
		Select("stats", "created").
		From("system_stats").
		Where(dbx.NewExp(
			"system={:system} AND type='1m' AND created > {:created}",
			dbx.Params{
				"system": systemRecord.Id,
				// subtract some time to give us a bit of buffer
				"created": oldestTime.Add(-time.Second * 90),
			},
		)).
		OrderBy("created").
		All(&systemStats)
	if err != nil || len(systemStats) == 0 {
		return err
	}

	// get oldest record creation time from first record in the slice
	oldestRecordTime := systemStats[0].Created.Time()
	// log.Println("oldestRecordTime", oldestRecordTime.String())

	// Filter validAlerts to keep only those with time newer than oldestRecord
	filteredAlerts := make([]SystemAlertData, 0, len(validAlerts))
	for _, alert := range validAlerts {
		if alert.time.After(oldestRecordTime) {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}
	validAlerts = filteredAlerts

	if len(validAlerts) == 0 {
		// log.Println("no valid alerts found")
		return nil
	}

	// No historical system stats processing needed since we only have ping alerts now
	// Ping alerts are processed immediately above without historical data
	return nil
}

func (am *AlertManager) sendSystemAlert(alert SystemAlertData) {
	// log.Printf("Sending alert %s: val %f | count %d | threshold %f\n", alert.name, alert.val, alert.count, alert.threshold)
	systemName := alert.systemRecord.GetString("name")

	// change Disk to Disk usage
	if alert.name == "Disk" {
		alert.name += " usage"
	}
	// format LoadAvg5 and LoadAvg15
	if after, ok := strings.CutPrefix(alert.name, "LoadAvg"); ok {
		alert.name = after + "m Load"
	}

	// make title alert name lowercase if not CPU
	titleAlertName := alert.name
	if titleAlertName != "CPU" {
		titleAlertName = strings.ToLower(titleAlertName)
	}

	var subject string
	if alert.triggered {
		subject = fmt.Sprintf("%s %s above threshold", systemName, titleAlertName)
	} else {
		subject = fmt.Sprintf("%s %s below threshold", systemName, titleAlertName)
	}
	minutesLabel := "minute"
	if alert.min > 1 {
		minutesLabel += "s"
	}
	if alert.descriptor == "" {
		alert.descriptor = alert.name
	}
	body := fmt.Sprintf("%s averaged %.2f%s for the previous %v %s.", alert.descriptor, alert.val, alert.unit, alert.min, minutesLabel)

	alert.alertRecord.Set("triggered", alert.triggered)
	if err := am.hub.Save(alert.alertRecord); err != nil {
		// app.Logger().Error("failed to save alert record", "err", err)
		return
	}
	am.SendAlert(AlertMessageData{
		UserID:   alert.alertRecord.GetString("user"),
		Title:    subject,
		Message:  body,
		Link:     am.hub.MakeLink("system", systemName),
		LinkText: "View " + systemName,
	})
}
