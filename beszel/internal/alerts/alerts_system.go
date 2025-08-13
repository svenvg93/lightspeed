package alerts

import (
	"beszel/internal/entities/system"
	"fmt"
	"math"
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
		case "PingPacketLoss":
			// Check average packet loss across all ping targets
			if data.Stats.PingResults != nil {
				var totalPacketLoss float64
				var hostCount int
				for _, result := range data.Stats.PingResults {
					totalPacketLoss += result.PacketLoss
					hostCount++
				}
				if hostCount > 0 {
					val = totalPacketLoss / float64(hostCount)
					unit = "%"
				} else {
					continue
				}
			} else {
				continue
			}
		case "PingLatency":
			// Check average latency across all ping targets
			if data.Stats.PingResults != nil {
				var totalLatency float64
				var hostCount int
				for _, result := range data.Stats.PingResults {
					if result.AvgRtt > 0 { // Only include hosts that responded
						totalLatency += result.AvgRtt
						hostCount++
					}
				}
				if hostCount > 0 {
					val = totalLatency / float64(hostCount)
					unit = " ms"
				} else {
					continue
				}
			} else {
				continue
			}
		case "SpeedtestDownload":
			// Check average download speed across all speedtest servers
			if data.Stats.SpeedtestResults != nil {
				var totalDownload float64
				var serverCount int
				for _, result := range data.Stats.SpeedtestResults {
					if result.Status == "success" {
						totalDownload += result.DownloadSpeed
						serverCount++
					}
				}
				if serverCount > 0 {
					val = totalDownload / float64(serverCount)
					unit = " Mbps"
				} else {
					continue
				}
			} else {
				continue
			}
		case "SpeedtestUpload":
			// Check average upload speed across all speedtest servers
			if data.Stats.SpeedtestResults != nil {
				var totalUpload float64
				var serverCount int
				for _, result := range data.Stats.SpeedtestResults {
					if result.Status == "success" {
						totalUpload += result.UploadSpeed
						serverCount++
					}
				}
				if serverCount > 0 {
					val = totalUpload / float64(serverCount)
					unit = " Mbps"
				} else {
					continue
				}
			} else {
				continue
			}
		case "HTTPResponseTime":
			// Check average HTTP response time across all HTTP targets
			if data.Stats.HttpResults != nil {
				var totalResponseTime float64
				var requestCount int
				for _, result := range data.Stats.HttpResults {
					if result.Status == "success" && result.ResponseTime > 0 {
						totalResponseTime += result.ResponseTime
						requestCount++
					}
				}
				if requestCount > 0 {
					val = totalResponseTime / float64(requestCount)
					unit = " ms"
				} else {
					continue
				}
			} else {
				continue
			}
		case "HTTPFailures":
			// Check HTTP response failures (same as HTTP but with different name)
			if data.Stats.HttpResults != nil {
				var failedRequests []string
				var totalRequests int
				for url, result := range data.Stats.HttpResults {
					totalRequests++
					if result.Status != "success" {
						failedRequests = append(failedRequests, url)
					}
				}
				if totalRequests > 0 {
					val = float64(len(failedRequests)) / float64(totalRequests) * 100
					unit = "% failed"
				} else {
					continue
				}
			} else {
				continue
			}
		case "DNSTime":
			// Check average DNS lookup time across all DNS targets
			if data.Stats.DnsResults != nil {
				var totalLookupTime float64
				var lookupCount int
				for _, result := range data.Stats.DnsResults {
					if result.Status == "success" && result.LookupTime > 0 {
						totalLookupTime += result.LookupTime
						lookupCount++
					}
				}
				if lookupCount > 0 {
					val = totalLookupTime / float64(lookupCount)
					unit = " ms"
				} else {
					continue
				}
			} else {
				continue
			}
		case "DNSFailures":
			// Check DNS lookup failures (same as DNS but with different name)
			if data.Stats.DnsResults != nil {
				var failedLookups []string
				var totalLookups int
				for key, result := range data.Stats.DnsResults {
					totalLookups++
					if result.Status != "success" {
						failedLookups = append(failedLookups, key)
					}
				}
				if totalLookups > 0 {
					val = float64(len(failedLookups)) / float64(totalLookups) * 100
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

		// Determine if we should trigger based on metric type
		var shouldTrigger bool
		switch name {
		case "SpeedtestDownload", "SpeedtestUpload":
			// For speed metrics, alert when value is BELOW threshold
			shouldTrigger = (!triggered && val < threshold) || (triggered && val >= threshold)
			// Debug logging

		case "DNSFailures", "HTTPFailures", "PingPacketLoss", "PingLatency":
			// For failure/performance metrics, alert when value is ABOVE threshold
			shouldTrigger = (!triggered && val > threshold) || (triggered && val <= threshold)
		case "DNSTime", "HTTPResponseTime":
			// For time-based metrics, alert when value is ABOVE threshold
			shouldTrigger = (!triggered && val > threshold) || (triggered && val <= threshold)
		default:
			// For other metrics, use existing logic
			shouldTrigger = (!triggered && val <= threshold) || (triggered && val > threshold)
		}

		// CONTINUE
		// IF alert should not trigger
		if !shouldTrigger {
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
			// Determine if alert should be triggered based on metric type
			switch alert.name {
			case "SpeedtestDownload", "SpeedtestUpload":
				// For speed metrics, alert when value is below threshold
				alert.triggered = val < threshold
			case "DNSFailures", "HTTPFailures", "PingPacketLoss", "PingLatency":
				// For failure/performance metrics, alert when value is above threshold
				alert.triggered = val > threshold
			case "DNSTime", "HTTPResponseTime":
				// For time-based metrics, alert when value is above threshold
				alert.triggered = val > threshold
			default:
				// For other metrics, use existing logic
				alert.triggered = val > threshold
			}
			go am.sendSystemAlert(alert)
			continue
		}

		alert.time = now.Add(-time.Duration(min) * time.Minute)
		if alert.time.Before(oldestTime) {
			oldestTime = alert.time
		}

		validAlerts = append(validAlerts, alert)
	}

	// Query system_averages collection for historical data
	systemAverages := []struct {
		PingLatency     *float64       `db:"ping_latency"`
		PingPacketLoss  *float64       `db:"ping_packet_loss"`
		DnsLatency      *float64       `db:"dns_latency"`
		DnsFailureRate  *float64       `db:"dns_failure_rate"`
		HttpLatency     *float64       `db:"http_latency"`
		HttpFailureRate *float64       `db:"http_failure_rate"`
		DownloadSpeed   *float64       `db:"download_speed"`
		UploadSpeed     *float64       `db:"upload_speed"`
		Created         types.DateTime `db:"created"`
	}{}

	err = am.hub.DB().NewQuery(`
		SELECT ping_latency, ping_packet_loss, dns_latency, dns_failure_rate, http_latency, http_failure_rate, download_speed, upload_speed, created
		FROM system_averages 
		WHERE system = {:system} AND created > {:created}
		ORDER BY created
	`).Bind(dbx.Params{
		"system":  systemRecord.Id,
		"created": oldestTime.Add(-time.Second * 90),
	}).All(&systemAverages)

	if err != nil || len(systemAverages) == 0 {
		return err
	}

	// get oldest record creation time from first record in the slice
	oldestRecordTime := systemAverages[0].Created.Time()

	// Filter validAlerts to keep only those with time newer than oldestRecord
	filteredAlerts := make([]SystemAlertData, 0, len(validAlerts))
	for _, alert := range validAlerts {
		if alert.time.After(oldestRecordTime) {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}
	validAlerts = filteredAlerts

	if len(validAlerts) == 0 {
		return nil
	}

	// Process historical data for time-based alerts
	for _, alert := range validAlerts {
		// Calculate average over the specified time period
		var sum float64
		var count int

		for _, avg := range systemAverages {
			avgTime := avg.Created.Time()
			if avgTime.After(alert.time) && avgTime.Before(now) {
				var metricValue float64
				var hasValue bool

				switch alert.name {
				case "SpeedtestDownload":
					if avg.DownloadSpeed != nil {
						metricValue = *avg.DownloadSpeed
						hasValue = true
						// Debug logging

					}
				case "SpeedtestUpload":
					if avg.UploadSpeed != nil {
						metricValue = *avg.UploadSpeed
						hasValue = true
					}
				case "PingPacketLoss":
					if avg.PingPacketLoss != nil {
						metricValue = *avg.PingPacketLoss
						hasValue = true
					}
				case "PingLatency":
					if avg.PingLatency != nil {
						metricValue = *avg.PingLatency
						hasValue = true
					}
				case "HTTPResponseTime":
					if avg.HttpLatency != nil {
						metricValue = *avg.HttpLatency
						hasValue = true
					}
				case "HTTPFailures":
					if avg.HttpFailureRate != nil {
						metricValue = *avg.HttpFailureRate
						hasValue = true
					}
				case "DNSTime":
					if avg.DnsLatency != nil {
						metricValue = *avg.DnsLatency
						hasValue = true
					}
				case "DNSFailures":
					if avg.DnsFailureRate != nil {
						metricValue = *avg.DnsFailureRate
						hasValue = true
					}
				}

				if hasValue && metricValue >= 0 {
					sum += metricValue
					count++
				}
			}
		}

		if count > 0 {
			averageValue := math.Round((sum/float64(count))*100) / 100
			alert.val = averageValue

			// Determine if alert should be triggered based on metric type
			switch alert.name {
			case "SpeedtestDownload", "SpeedtestUpload":
				// For speed metrics, alert when average is below threshold
				alert.triggered = averageValue < alert.threshold
				// Debug logging
				fmt.Printf("Final SpeedtestDownload: average=%.2f, threshold=%.2f, triggered=%v\n", averageValue, alert.threshold, alert.triggered)
			case "DNSFailures", "HTTPFailures", "PingPacketLoss", "PingLatency":
				// For failure/performance metrics, alert when average is above threshold
				alert.triggered = averageValue > alert.threshold
			case "DNSTime", "HTTPResponseTime":
				// For time-based metrics, alert when average is above threshold
				alert.triggered = averageValue > alert.threshold
			default:
				// For other metrics, use existing logic
				alert.triggered = averageValue > alert.threshold
			}

			go am.sendSystemAlert(alert)
		}
	}

	return nil
}

func (am *AlertManager) sendSystemAlert(alert SystemAlertData) {
	// Debug logging
	am.hub.Logger().Info("sendSystemAlert called", "alertName", alert.name, "value", alert.val, "threshold", alert.threshold, "triggered", alert.triggered)

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
		// Determine the appropriate message based on metric type
		switch alert.name {
		case "SpeedtestDownload", "SpeedtestUpload":
			subject = fmt.Sprintf("%s %s below threshold", systemName, titleAlertName)
		case "DNSFailures", "HTTPFailures", "PingPacketLoss", "PingLatency":
			subject = fmt.Sprintf("%s %s above threshold", systemName, titleAlertName)
		case "DNSTime", "HTTPResponseTime":
			subject = fmt.Sprintf("%s %s above threshold", systemName, titleAlertName)
		default:
			subject = fmt.Sprintf("%s %s above threshold", systemName, titleAlertName)
		}
	} else {
		// Determine the appropriate message based on metric type
		switch alert.name {
		case "SpeedtestDownload", "SpeedtestUpload":
			subject = fmt.Sprintf("%s %s above threshold", systemName, titleAlertName)
		case "DNS", "HTTP", "DNSFailures", "HTTPFailures", "PingPacketLoss", "PingLatency":
			subject = fmt.Sprintf("%s %s below threshold", systemName, titleAlertName)
		case "DNSTime", "HTTPResponseTime":
			subject = fmt.Sprintf("%s %s below threshold", systemName, titleAlertName)
		default:
			subject = fmt.Sprintf("%s %s below threshold", systemName, titleAlertName)
		}
	}
	minutesLabel := "minute"
	if alert.min > 1 {
		minutesLabel += "s"
	}
	if alert.descriptor == "" {
		alert.descriptor = alert.name
	}

	// Create appropriate message body based on metric type
	var body string
	switch alert.name {
	case "SpeedtestDownload", "SpeedtestUpload":
		body = fmt.Sprintf("Average %s across all speedtest servers was %.2f%s for the previous %v %s.",
			strings.ToLower(alert.name), alert.val, alert.unit, alert.min, minutesLabel)
	case "PingPacketLoss":
		body = fmt.Sprintf("Average packet loss across all ping targets was %.2f%s for the previous %v %s.",
			alert.val, alert.unit, alert.min, minutesLabel)
	case "PingLatency":
		body = fmt.Sprintf("Average latency across all ping targets was %.2f%s for the previous %v %s.",
			alert.val, alert.unit, alert.min, minutesLabel)
	case "DNSTime":
		body = fmt.Sprintf("Average DNS lookup time across all targets was %.2f%s for the previous %v %s.",
			alert.val, alert.unit, alert.min, minutesLabel)
	case "DNSFailures":
		body = fmt.Sprintf("DNS lookup failures averaged %.2f%s for the previous %v %s.",
			alert.val, alert.unit, alert.min, minutesLabel)
	case "HTTPResponseTime":
		body = fmt.Sprintf("Average HTTP response time across all targets was %.2f%s for the previous %v %s.",
			alert.val, alert.unit, alert.min, minutesLabel)
	case "HTTPFailures":
		body = fmt.Sprintf("HTTP request failures averaged %.2f%s for the previous %v %s.",
			alert.val, alert.unit, alert.min, minutesLabel)
	default:
		body = fmt.Sprintf("%s averaged %.2f%s for the previous %v %s.",
			alert.descriptor, alert.val, alert.unit, alert.min, minutesLabel)
	}

	alert.alertRecord.Set("triggered", alert.triggered)
	if err := am.hub.Save(alert.alertRecord); err != nil {
		// app.Logger().Error("failed to save alert record", "err", err)
		return
	}
	am.SendAlert(AlertMessageData{
		UserID:   "", // Not used anymore - sends to all users
		Title:    subject,
		Message:  body,
		Link:     am.hub.MakeLink("system", systemName),
		LinkText: "View " + systemName,
	})
}
