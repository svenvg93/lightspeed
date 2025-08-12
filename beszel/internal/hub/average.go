package hub

import (
	"github.com/pocketbase/dbx"
)

// SystemAverages represents the calculated averages for a system
type SystemAverages struct {
	AP  float64 `json:"ap"`  // Average ping
	AD  float64 `json:"ad"`  // Average DNS
	AH  float64 `json:"ah"`  // Average HTTP
	ADL float64 `json:"adl"` // Average download
	AUL float64 `json:"aul"` // Average upload
}

// calculateSystemAverages calculates averages from historical data for all systems
// and updates the system.averages field with the calculated values
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

		// Update the system record with calculated averages
		systemRecord.Set("averages", averages)

		if err := h.Save(systemRecord); err != nil {
			h.Logger().Error("Failed to update system averages", "system", systemID, "err", err)
		} else {
			h.Logger().Debug("Updated system averages", "system", systemID,
				"ping", averages.AP, "dns", averages.AD, "http", averages.AH,
				"download", averages.ADL, "upload", averages.AUL)
		}
	}

	h.Logger().Debug("Completed system averages calculation")
	return nil
}

// calculateAveragesForSystem calculates averages for a specific system
func (h *Hub) calculateAveragesForSystem(systemID string) (*SystemAverages, error) {
	averages := &SystemAverages{}

	// Calculate ping average from ping_stats
	pingAvg, err := h.calculatePingAverage(systemID)
	if err != nil {
		h.Logger().Error("Failed to calculate ping average", "system", systemID, "err", err)
	} else {
		averages.AP = pingAvg
	}

	// Calculate DNS average from dns_stats
	dnsAvg, err := h.calculateDNSAverage(systemID)
	if err != nil {
		h.Logger().Error("Failed to calculate DNS average", "system", systemID, "err", err)
	} else {
		averages.AD = dnsAvg
	}

	// Calculate HTTP average from http_stats
	httpAvg, err := h.calculateHTTPAverage(systemID)
	if err != nil {
		h.Logger().Error("Failed to calculate HTTP average", "system", systemID, "err", err)
	} else {
		averages.AH = httpAvg
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

// calculatePingAverage calculates the average ping time from the last 10 ping_stats records
func (h *Hub) calculatePingAverage(systemID string) (float64, error) {
	var pingStats []struct {
		AvgRtt float64 `db:"avg_rtt"`
	}

	err := h.DB().NewQuery(`
		SELECT avg_rtt
		FROM ping_stats
		WHERE system = {:system} AND avg_rtt > 0
		ORDER BY created DESC
		LIMIT 10
	`).Bind(dbx.Params{"system": systemID}).All(&pingStats)

	if err != nil || len(pingStats) == 0 {
		return 0, err
	}

	total := 0.0
	for _, stat := range pingStats {
		total += stat.AvgRtt
	}
	return total / float64(len(pingStats)), nil
}

// calculateDNSAverage calculates the average DNS lookup time from the last 10 dns_stats records
func (h *Hub) calculateDNSAverage(systemID string) (float64, error) {
	var dnsStats []struct {
		LookupTime float64 `db:"lookup_time"`
	}

	err := h.DB().NewQuery(`
		SELECT lookup_time 
		FROM dns_stats 
		WHERE system = {:system} AND lookup_time > 0 AND status = 'success' 
		ORDER BY created DESC 
		LIMIT 10
	`).Bind(dbx.Params{"system": systemID}).All(&dnsStats)

	if err != nil || len(dnsStats) == 0 {
		return 0, err
	}

	total := 0.0
	for _, stat := range dnsStats {
		total += stat.LookupTime
	}
	return total / float64(len(dnsStats)), nil
}

// calculateHTTPAverage calculates the average HTTP response time from the last 10 http_stats records
func (h *Hub) calculateHTTPAverage(systemID string) (float64, error) {
	var httpStats []struct {
		ResponseTime float64 `db:"response_time"`
	}

	err := h.DB().NewQuery(`
		SELECT response_time 
		FROM http_stats 
		WHERE system = {:system} AND response_time > 0 AND status = 'success' 
		ORDER BY created DESC 
		LIMIT 10
	`).Bind(dbx.Params{"system": systemID}).All(&httpStats)

	if err != nil || len(httpStats) == 0 {
		return 0, err
	}

	total := 0.0
	for _, stat := range httpStats {
		total += stat.ResponseTime
	}
	return total / float64(len(httpStats)), nil
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

	avgDownload := totalDownload / float64(len(speedtestStats))
	avgUpload := totalUpload / float64(len(speedtestStats))

	return avgDownload, avgUpload, nil
}
