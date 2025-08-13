package hub

import (
	"fmt"
	"math"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// SystemAverages represents the calculated averages for a system
type SystemAverages struct {
	AP  float64 `json:"ap"`  // Average ping latency
	APL float64 `json:"apl"` // Average ping packet loss
	AD  float64 `json:"ad"`  // Average DNS lookup time
	ADF float64 `json:"adf"` // Average DNS failure rate
	AH  float64 `json:"ah"`  // Average HTTP response time
	AHF float64 `json:"ahf"` // Average HTTP failure rate
	ADL float64 `json:"adl"` // Average download
	AUL float64 `json:"aul"` // Average upload
}

// calculateSystemAverages calculates averages from historical data for all systems
// and stores them in the system_averages collection
func (h *Hub) calculateSystemAverages() error {
	h.Logger().Debug("Starting system averages calculation")

	// Get all active systems
	systems, err := h.FindAllRecords("systems", dbx.NewExp("status != 'paused'"))
	if err != nil {
		h.Logger().Error("Failed to get systems", "err", err)
		return err
	}

	for _, systemRecord := range systems {
		systemID := systemRecord.Id

		averages, err := h.calculateAveragesForSystem(systemID)
		if err != nil {
			h.Logger().Error("Failed to calculate averages for system", "system", systemID, "err", err)
			continue
		}

		// Store historical averages (no longer updating system record)
		if err := h.storeHistoricalAverages(systemID, averages); err != nil {
			h.Logger().Error("Failed to store historical averages", "system", systemID, "err", err)
		} else {
			h.Logger().Debug("Stored historical averages", "system", systemID,
				"ping_latency", averages.AP, "ping_packet_loss", averages.APL,
				"dns_latency", averages.AD, "dns_failure_rate", averages.ADF,
				"http_latency", averages.AH, "http_failure_rate", averages.AHF,
				"download", averages.ADL, "upload", averages.AUL)
		}

		// Store historical averages in a separate collection
		if err := h.storeHistoricalAverages(systemID, averages); err != nil {
			h.Logger().Error("Failed to store historical averages", "system", systemID, "err", err)
		}
	}

	h.Logger().Debug("Completed system averages calculation")
	return nil
}

// calculateAveragesForSystem calculates averages for a specific system
func (h *Hub) calculateAveragesForSystem(systemID string) (*SystemAverages, error) {
	averages := &SystemAverages{}

	// Calculate ping average from ping_stats
	pingAvg, pingLossAvg, err := h.calculatePingAverage(systemID)
	if err != nil {
		h.Logger().Error("Failed to calculate ping average", "system", systemID, "err", err)
	} else {
		averages.AP = pingAvg
		averages.APL = pingLossAvg
	}

	// Calculate DNS average from dns_stats
	dnsAvg, dnsFailureAvg, err := h.calculateDNSAverage(systemID)
	if err != nil {
		h.Logger().Error("Failed to calculate DNS average", "system", systemID, "err", err)
	} else {
		averages.AD = dnsAvg
		averages.ADF = dnsFailureAvg
	}

	// Calculate HTTP average from http_stats
	httpAvg, httpFailureAvg, err := h.calculateHTTPAverage(systemID)
	if err != nil {
		h.Logger().Error("Failed to calculate HTTP average", "system", systemID, "err", err)
	} else {
		averages.AH = httpAvg
		averages.AHF = httpFailureAvg
	}

	// Calculate speedtest averages from speedtest_stats
	downloadAvg, uploadAvg, err := h.calculateSpeedtestAverages(systemID)
	if err != nil {
		h.Logger().Error("Failed to calculate speedtest averages", "system", systemID, "err", err)
	} else {
		averages.ADL = downloadAvg
		averages.AUL = uploadAvg
	}

	return averages, nil
}

// calculatePingAverage calculates the average ping time and packet loss from the last 10 ping_stats records
func (h *Hub) calculatePingAverage(systemID string) (float64, float64, error) {
	var pingStats []struct {
		AvgRtt     float64 `db:"avg_rtt"`
		PacketLoss float64 `db:"packet_loss"`
	}

	err := h.DB().NewQuery(`
		SELECT avg_rtt, packet_loss
		FROM ping_stats
		WHERE system = {:system}
		ORDER BY created DESC
		LIMIT 10
	`).Bind(dbx.Params{"system": systemID}).All(&pingStats)

	if err != nil || len(pingStats) == 0 {
		return 0, 0, err
	}

	totalLatency := 0.0
	totalPacketLoss := 0.0
	latencyCount := 0
	packetLossCount := 0

	for _, stat := range pingStats {
		// Calculate average latency (only for successful pings)
		if stat.AvgRtt > 0 {
			totalLatency += stat.AvgRtt
			latencyCount++
		}

		// Calculate average packet loss (include all records)
		totalPacketLoss += stat.PacketLoss
		packetLossCount++
	}

	avgLatency := 0.0
	if latencyCount > 0 {
		avgLatency = math.Round((totalLatency/float64(latencyCount))*100) / 100
	}

	avgPacketLoss := 0.0
	if packetLossCount > 0 {
		avgPacketLoss = math.Round((totalPacketLoss/float64(packetLossCount))*100) / 100
	}

	return avgLatency, avgPacketLoss, nil
}

// calculateDNSAverage calculates the average DNS lookup time and failure rate from the last 10 dns_stats records
func (h *Hub) calculateDNSAverage(systemID string) (float64, float64, error) {
	var dnsStats []struct {
		LookupTime float64 `db:"lookup_time"`
		Status     string  `db:"status"`
	}

	err := h.DB().NewQuery(`
		SELECT lookup_time, status
		FROM dns_stats 
		WHERE system = {:system}
		ORDER BY created DESC 
		LIMIT 10
	`).Bind(dbx.Params{"system": systemID}).All(&dnsStats)

	if err != nil || len(dnsStats) == 0 {
		return 0, 0, err
	}

	totalLookupTime := 0.0
	successfulLookups := 0
	failedLookups := 0

	for _, stat := range dnsStats {
		// Calculate average lookup time (only for successful lookups)
		if stat.Status == "success" && stat.LookupTime > 0 {
			totalLookupTime += stat.LookupTime
			successfulLookups++
		}

		// Count failures
		if stat.Status != "success" {
			failedLookups++
		}
	}

	// Calculate average lookup time
	avgLookupTime := 0.0
	if successfulLookups > 0 {
		avgLookupTime = math.Round((totalLookupTime/float64(successfulLookups))*100) / 100
	}

	// Calculate failure rate
	totalLookups := len(dnsStats)
	avgFailureRate := 0.0
	if totalLookups > 0 {
		avgFailureRate = math.Round((float64(failedLookups)/float64(totalLookups)*100)*100) / 100
	}

	return avgLookupTime, avgFailureRate, nil
}

// calculateHTTPAverage calculates the average HTTP response time and failure rate from the last 10 http_stats records
func (h *Hub) calculateHTTPAverage(systemID string) (float64, float64, error) {
	var httpStats []struct {
		ResponseTime float64 `db:"response_time"`
		Status       string  `db:"status"`
	}

	err := h.DB().NewQuery(`
		SELECT response_time, status
		FROM http_stats 
		WHERE system = {:system}
		ORDER BY created DESC 
		LIMIT 10
	`).Bind(dbx.Params{"system": systemID}).All(&httpStats)

	if err != nil || len(httpStats) == 0 {
		return 0, 0, err
	}

	totalResponseTime := 0.0
	successfulRequests := 0
	failedRequests := 0

	for _, stat := range httpStats {
		// Calculate average response time (only for successful requests)
		if stat.Status == "success" && stat.ResponseTime > 0 {
			totalResponseTime += stat.ResponseTime
			successfulRequests++
		}

		// Count failures
		if stat.Status != "success" {
			failedRequests++
		}
	}

	// Calculate average response time
	avgResponseTime := 0.0
	if successfulRequests > 0 {
		avgResponseTime = math.Round((totalResponseTime/float64(successfulRequests))*100) / 100
	}

	// Calculate failure rate
	totalRequests := len(httpStats)
	avgFailureRate := 0.0
	if totalRequests > 0 {
		avgFailureRate = math.Round((float64(failedRequests)/float64(totalRequests)*100)*100) / 100
	}

	return avgResponseTime, avgFailureRate, nil
}

// calculateSpeedtestAverages calculates the average download and upload speeds from the last 10 speedtest_stats records
func (h *Hub) calculateSpeedtestAverages(systemID string) (float64, float64, error) {
	var speedtestStats []struct {
		DownloadSpeed float64 `db:"download_speed"`
		UploadSpeed   float64 `db:"upload_speed"`
	}

	err := h.DB().NewQuery(`
		SELECT download_speed, upload_speed 
		FROM speedtest_stats 
		WHERE system = {:system} AND download_speed > 0 AND upload_speed > 0 AND status = 'success' 
		ORDER BY created DESC 
		LIMIT 10
	`).Bind(dbx.Params{"system": systemID}).All(&speedtestStats)

	if err != nil || len(speedtestStats) == 0 {
		return 0, 0, err
	}

	totalDownload := 0.0
	totalUpload := 0.0
	for _, stat := range speedtestStats {
		totalDownload += stat.DownloadSpeed
		totalUpload += stat.UploadSpeed
	}

	avgDownload := math.Round((totalDownload/float64(len(speedtestStats)))*100) / 100
	avgUpload := math.Round((totalUpload/float64(len(speedtestStats)))*100) / 100

	return avgDownload, avgUpload, nil
}

// storeHistoricalAverages stores the calculated averages in a historical collection
func (h *Hub) storeHistoricalAverages(systemID string, averages *SystemAverages) error {
	// Find the system_averages collection
	collection, err := h.FindCollectionByNameOrId("system_averages")
	if err != nil {
		// Collection doesn't exist yet, just log for now
		h.Logger().Debug("Historical averages calculated (collection not found)",
			"system", systemID,
			"ping_latency", averages.AP,
			"ping_packet_loss", averages.APL,
			"dns_latency", averages.AD,
			"dns_failure_rate", averages.ADF,
			"http_latency", averages.AH,
			"http_failure_rate", averages.AHF,
			"download_speed", averages.ADL,
			"upload_speed", averages.AUL,
			"timestamp", time.Now().UTC(),
		)
		return nil
	}

	// Create a new record with the averages
	record := core.NewRecord(collection)
	record.Set("system", systemID)
	record.Set("ping_latency", averages.AP)
	record.Set("ping_packet_loss", averages.APL)
	record.Set("dns_latency", averages.AD)
	record.Set("dns_failure_rate", averages.ADF)
	record.Set("http_latency", averages.AH)
	record.Set("http_failure_rate", averages.AHF)
	record.Set("download_speed", averages.ADL)
	record.Set("upload_speed", averages.AUL)

	if err := h.Save(record); err != nil {
		return fmt.Errorf("failed to save historical averages: %w", err)
	}

	h.Logger().Debug("Stored historical averages", "system", systemID, "record_id", record.Id)
	return nil
}
