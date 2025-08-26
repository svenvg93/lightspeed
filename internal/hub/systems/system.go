package systems

import (
	"beszel/internal/entities/system"
	"beszel/internal/hub/ws"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/blang/semver"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type System struct {
	Id                string               `db:"id"`
	Host              string               `db:"host"`
	Status            string               `db:"status"`
	manager           *SystemManager       // Manager that this system belongs to
	data              *system.CombinedData // system data from agent
	ctx               context.Context      // Context for stopping the updater
	cancel            context.CancelFunc   // Stops and removes system from updater
	WsConn            *ws.WsConn           // Handler for agent WebSocket connection
	agentVersion      semver.Version       // Agent version
	updateTicker      *time.Ticker         // Ticker for updating the system
	lastPingTime      time.Time            // Track when ping records were last created
	lastDnsTime       time.Time            // Track when DNS records were last created
	lastHttpTime      time.Time            // Track when HTTP records were last created
	lastSpeedtestTime time.Time            // Track when speedtest records were last created
}

func (sm *SystemManager) NewSystem(systemId string) *System {
	system := &System{
		Id:   systemId,
		data: &system.CombinedData{},
	}
	system.ctx, system.cancel = system.getContext()
	return system
}

// StartUpdater starts the system updater.
// It first fetches the data from the agent then updates the records.
// If the data is not found or the system is down, it sets the system down.
func (sys *System) StartUpdater() {
	// Channel that can be used to set the system down. Currently only used to
	// allow a short delay for reconnection after websocket connection is closed.
	var downChan chan struct{}

	// Add random jitter to first WebSocket connection to prevent
	// clustering if all agents are started at the same time.
	var jitter <-chan time.Time
	if sys.WsConn != nil {
		jitter = getJitter()
		// use the websocket connection's down channel to set the system down
		downChan = sys.WsConn.DownChan
	} else {
		// if the system does not have a websocket connection, wait before updating
		// to allow the agent to connect via websocket (makes sure fingerprint is set).
		time.Sleep(11 * time.Second)
	}

	// update immediately if system is not paused
	if sys.Status != paused && sys.ctx.Err() == nil {
		// Add a small delay to allow the WebSocket connection to fully establish
		time.Sleep(1 * time.Second)
		if err := sys.update(); err != nil {
			_ = sys.setDown(err)
		}
	}

	sys.updateTicker = time.NewTicker(time.Duration(interval) * time.Millisecond)
	// Go 1.23+ will automatically stop the ticker when the system is garbage collected, however we seem to need this or testing/synctest will block even if calling runtime.GC()
	defer sys.updateTicker.Stop()

	for {
		select {
		case <-sys.ctx.Done():
			return
		case <-sys.updateTicker.C:
			if err := sys.update(); err != nil {
				_ = sys.setDown(err)
			}
		case <-downChan:
			sys.WsConn = nil
			downChan = nil
			_ = sys.setDown(nil)
		case <-jitter:
			sys.updateTicker.Reset(time.Duration(interval) * time.Millisecond)
			if err := sys.update(); err != nil {
				_ = sys.setDown(err)
			}
		}
	}
}

// update updates the system data and records.
func (sys *System) update() error {
	if sys.Status == paused {
		sys.handlePaused()
		return nil
	}
	data, err := sys.fetchDataFromAgent()
	if err == nil {
		_, err = sys.createRecords(data)
	}
	return err
}

func (sys *System) handlePaused() {
	if sys.WsConn == nil {
		// if the system is paused and there's no websocket connection, remove the system
		_ = sys.manager.RemoveSystem(sys.Id)
	} else {
		// Send a ping to the agent to keep the connection alive if the system is paused
		if err := sys.WsConn.Ping(); err != nil {
			sys.manager.hub.Logger().Warn("Failed to ping agent", "system", sys.Id, "err", err)
			_ = sys.manager.RemoveSystem(sys.Id)
		}
	}
}

// createRecords updates the system record and adds individual stats records
func (sys *System) createRecords(data *system.CombinedData) (*core.Record, error) {
	systemRecord, err := sys.getRecord()
	if err != nil {
		return nil, err
	}
	hub := sys.manager.hub

	// Create ping_stats records if we have ping data and it's new
	if data.Stats.PingResults != nil && len(data.Stats.PingResults) > 0 {
		// Check if we have new ping data by comparing LastChecked times
		var hasNewData bool
		for _, result := range data.Stats.PingResults {
			if result.LastChecked.After(sys.lastPingTime) {
				hasNewData = true
				break
			}
		}

		if hasNewData {
			sys.manager.hub.Logger().Debug("Creating ping records", "count", len(data.Stats.PingResults))
			pingStatsCollection, err := hub.FindCollectionByNameOrId("ping_stats")
			if err != nil {
				return nil, err
			}

			// Create a separate record for each ping result
			for host, result := range data.Stats.PingResults {
				pingStatsRecord := core.NewRecord(pingStatsCollection)
				pingStatsRecord.Set("system", systemRecord.Id)
				pingStatsRecord.Set("host", host)
				pingStatsRecord.Set("packet_loss", result.PacketLoss)
				pingStatsRecord.Set("min_rtt", result.MinRtt)
				pingStatsRecord.Set("max_rtt", result.MaxRtt)
				pingStatsRecord.Set("avg_rtt", result.AvgRtt)
				// No type field needed - we're storing all raw data

				if err := hub.Save(pingStatsRecord); err != nil {
					return nil, err
				}
			}

			// Update the last ping time to the most recent LastChecked time
			for _, result := range data.Stats.PingResults {
				if result.LastChecked.After(sys.lastPingTime) {
					sys.lastPingTime = result.LastChecked
				}
			}
		}
	}

	// Create dns_stats records if we have DNS data and it's new
	if data.Stats.DnsResults != nil && len(data.Stats.DnsResults) > 0 {
		// Check if we have new DNS data by comparing LastChecked times
		var hasNewData bool
		for _, result := range data.Stats.DnsResults {
			if result.LastChecked.After(sys.lastDnsTime) {
				hasNewData = true
				break
			}
		}

		if hasNewData {
			sys.manager.hub.Logger().Debug("Creating DNS records", "count", len(data.Stats.DnsResults))
			dnsStatsCollection, err := hub.FindCollectionByNameOrId("dns_stats")
			if err != nil {
				return nil, err
			}

			// Create a separate record for each DNS result
			for _, result := range data.Stats.DnsResults {
				dnsStatsRecord := core.NewRecord(dnsStatsCollection)
				dnsStatsRecord.Set("system", systemRecord.Id)
				dnsStatsRecord.Set("domain", result.Domain)
				dnsStatsRecord.Set("server", result.Server)
				dnsStatsRecord.Set("type", result.Type)
				// No period_type field needed - we're storing all raw data
				dnsStatsRecord.Set("status", result.Status)
				dnsStatsRecord.Set("lookup_time", result.LookupTime)
				dnsStatsRecord.Set("error_code", result.ErrorCode)

				if err := hub.Save(dnsStatsRecord); err != nil {
					return nil, err
				}
			}

			// Update the last DNS time to the most recent LastChecked time
			for _, result := range data.Stats.DnsResults {
				if result.LastChecked.After(sys.lastDnsTime) {
					sys.lastDnsTime = result.LastChecked
				}
			}
		}
	}

	// Create http_stats records if we have HTTP data and it's new
	if data.Stats.HttpResults != nil && len(data.Stats.HttpResults) > 0 {
		// Check if we have new HTTP data by comparing LastChecked times
		var hasNewData bool
		for _, result := range data.Stats.HttpResults {
			if result.LastChecked.After(sys.lastHttpTime) {
				hasNewData = true
				break
			}
		}

		if hasNewData {
			sys.manager.hub.Logger().Debug("Creating HTTP records", "count", len(data.Stats.HttpResults))
			httpStatsCollection, err := hub.FindCollectionByNameOrId("http_stats")
			if err != nil {
				return nil, err
			}

			// Create a separate record for each HTTP result
			for url, result := range data.Stats.HttpResults {
				httpStatsRecord := core.NewRecord(httpStatsCollection)
				httpStatsRecord.Set("system", systemRecord.Id)
				httpStatsRecord.Set("url", url)
				httpStatsRecord.Set("status", result.Status)
				httpStatsRecord.Set("response_time", result.ResponseTime)
				httpStatsRecord.Set("status_code", result.StatusCode)
				httpStatsRecord.Set("error_code", result.ErrorCode)
				// No type field needed - we're storing all raw data

				if err := hub.Save(httpStatsRecord); err != nil {
					return nil, err
				}
			}

			// Update the last HTTP time to the most recent LastChecked time
			for _, result := range data.Stats.HttpResults {
				if result.LastChecked.After(sys.lastHttpTime) {
					sys.lastHttpTime = result.LastChecked
				}
			}
		}
	}

	// Create speedtest_stats records if we have speedtest data and it's new
	if data.Stats.SpeedtestResults != nil && len(data.Stats.SpeedtestResults) > 0 {
		
		// Check if we have new speedtest data by comparing LastChecked times
		var hasNewData bool
		for _, result := range data.Stats.SpeedtestResults {
			if result.LastChecked.After(sys.lastSpeedtestTime) {
				hasNewData = true
				break
			}
		}

		if hasNewData {
			// Use all speedtest results without validation - agent restart handles config changes
			validResults := data.Stats.SpeedtestResults

			if len(validResults) > 0 {
				sys.manager.hub.Logger().Debug("Creating speedtest records", "count", len(validResults))
				speedtestStatsCollection, err := hub.FindCollectionByNameOrId("speedtest_stats")
				if err != nil {
					return nil, err
				}

				// Create a separate record for each speedtest result
				for serverID, result := range validResults {
					speedtestStatsRecord := core.NewRecord(speedtestStatsCollection)
					speedtestStatsRecord.Set("system", systemRecord.Id)
					speedtestStatsRecord.Set("server_id", serverID)
					speedtestStatsRecord.Set("status", result.Status)
					speedtestStatsRecord.Set("download_speed", result.DownloadSpeed)
					speedtestStatsRecord.Set("upload_speed", result.UploadSpeed)
					speedtestStatsRecord.Set("latency", result.Latency)
					speedtestStatsRecord.Set("error_code", result.ErrorCode)
					speedtestStatsRecord.Set("ping_jitter", result.PingJitter)
					speedtestStatsRecord.Set("type", "raw") // Raw data type for initial records
					speedtestStatsRecord.Set("ping_low", result.PingLow)
					speedtestStatsRecord.Set("ping_high", result.PingHigh)
					speedtestStatsRecord.Set("download_bytes", result.DownloadBytes)
					speedtestStatsRecord.Set("download_elapsed", result.DownloadElapsed)
					speedtestStatsRecord.Set("download_latency_iqm", result.DownloadLatencyIQM)
					speedtestStatsRecord.Set("download_latency_low", result.DownloadLatencyLow)
					speedtestStatsRecord.Set("download_latency_high", result.DownloadLatencyHigh)
					speedtestStatsRecord.Set("download_latency_jitter", result.DownloadLatencyJitter)
					speedtestStatsRecord.Set("upload_bytes", result.UploadBytes)
					speedtestStatsRecord.Set("upload_elapsed", result.UploadElapsed)
					speedtestStatsRecord.Set("upload_latency_iqm", result.UploadLatencyIQM)
					speedtestStatsRecord.Set("upload_latency_low", result.UploadLatencyLow)
					speedtestStatsRecord.Set("upload_latency_high", result.UploadLatencyHigh)
					speedtestStatsRecord.Set("upload_latency_jitter", result.UploadLatencyJitter)
					speedtestStatsRecord.Set("packet_loss", result.PacketLoss)
					speedtestStatsRecord.Set("isp", result.ISP)
					speedtestStatsRecord.Set("interface_external_ip", result.InterfaceExternalIP)
					speedtestStatsRecord.Set("server_name", result.ServerName)
					speedtestStatsRecord.Set("server_location", result.ServerLocation)
					speedtestStatsRecord.Set("server_country", result.ServerCountry)
					speedtestStatsRecord.Set("server_host", result.ServerHost)
					speedtestStatsRecord.Set("server_ip", result.ServerIP)

					if err := hub.Save(speedtestStatsRecord); err != nil {
						return nil, err
					}
				}

				// Update the last speedtest time to the most recent LastChecked time
				for _, result := range validResults {
					if result.LastChecked.After(sys.lastSpeedtestTime) {
						sys.lastSpeedtestTime = result.LastChecked
					}
				}
			}
		}
	}

	// update system record (do this last because it triggers alerts and we need above records to be inserted first)
	systemRecord.Set("status", up)
	systemRecord.Set("info", data.Info)
	if err := hub.SaveNoValidate(systemRecord); err != nil {
		return nil, err
	}

	// Update current averages after saving all new stats
	if err := sys.updateCurrentAverages(); err != nil {
		// Log error but don't fail the entire update
		sys.manager.hub.Logger().Error("Failed to update current averages", "system", sys.Id, "error", err)
	}

	return systemRecord, nil
}

// getRecord retrieves the system record from the database.
// If the record is not found, it removes the system from the manager.
func (sys *System) getRecord() (*core.Record, error) {
	record, err := sys.manager.hub.FindRecordById("systems", sys.Id)
	if err != nil || record == nil {
		_ = sys.manager.RemoveSystem(sys.Id)
		return nil, err
	}
	return record, nil
}

// setDown marks a system as down in the database.
// It takes the original error that caused the system to go down and returns any error
// encountered during the process of updating the system status.
func (sys *System) setDown(originalError error) error {
	if sys.Status == down || sys.Status == paused {
		return nil
	}
	record, err := sys.getRecord()
	if err != nil {
		return err
	}
	if originalError != nil {
		sys.manager.hub.Logger().Error("System down", "system", record.GetString("name"), "err", originalError)
	}
	record.Set("status", down)
	return sys.manager.hub.SaveNoValidate(record)
}

func (sys *System) getContext() (context.Context, context.CancelFunc) {
	if sys.ctx == nil {
		sys.ctx, sys.cancel = context.WithCancel(context.Background())
	}
	return sys.ctx, sys.cancel
}

// fetchDataFromAgent attempts to fetch data from the agent via WebSocket.
func (sys *System) fetchDataFromAgent() (*system.CombinedData, error) {
	if sys.data == nil {
		sys.data = &system.CombinedData{}
	}

	if sys.WsConn != nil && sys.WsConn.IsConnected() {
		return sys.fetchDataViaWebSocket()
	}

	return nil, errors.New("no websocket connection available")
}

func (sys *System) fetchDataViaWebSocket() (*system.CombinedData, error) {
	if sys.WsConn == nil || !sys.WsConn.IsConnected() {
		return nil, errors.New("no websocket connection")
	}
	err := sys.WsConn.RequestSystemData(sys.data)
	if err != nil {
		return nil, err
	}
	return sys.data, nil
}

// closeWebSocketConnection closes the WebSocket connection but keeps the system in the manager.
// The system will be set as down a few seconds later if the connection is not re-established.
func (sys *System) closeWebSocketConnection() {
	if sys.WsConn != nil {
		sys.WsConn.Close(nil)
	}
}

// updateCurrentAverages calculates and stores current averages directly in the system record
// This provides real-time averages for the frontend without needing separate queries
func (sys *System) updateCurrentAverages() error {
	if sys.manager == nil || sys.manager.hub == nil {
		return fmt.Errorf("system manager or hub is nil")
	}

	sys.manager.hub.Logger().Debug("Calculating current averages", "system", sys.Id)

	// Calculate averages from the last 10 records of each stats table
	averages := struct {
		AP  float64 `json:"ap"`  // Average ping latency
		APL float64 `json:"apl"` // Average ping packet loss
		AD  float64 `json:"ad"`  // Average DNS lookup time
		ADF float64 `json:"adf"` // Average DNS failure rate
		AH  float64 `json:"ah"`  // Average HTTP response time
		AHF float64 `json:"ahf"` // Average HTTP failure rate
		ADL float64 `json:"adl"` // Average download speed
		AUL float64 `json:"aul"` // Average upload speed
		LastUpdated string `json:"last_updated"`
	}{}

	// Get current time for last_updated
	averages.LastUpdated = time.Now().UTC().Format(time.RFC3339)

	// Calculate ping averages from last 10 records
	pingQuery := sys.manager.hub.DB().NewQuery(`
		SELECT AVG(avg_rtt) as avg_latency, AVG(packet_loss) as avg_packet_loss
		FROM (
			SELECT avg_rtt, packet_loss
			FROM ping_stats 
			WHERE system = {:system}
			ORDER BY created DESC
			LIMIT 10
		)
	`).Bind(dbx.Params{
		"system": sys.Id,
	})

	pingResult := struct {
		AvgLatency    *float64 `db:"avg_latency"`
		AvgPacketLoss *float64 `db:"avg_packet_loss"`
	}{}

	if err := pingQuery.One(&pingResult); err == nil {
		if pingResult.AvgLatency != nil {
			averages.AP = *pingResult.AvgLatency
		}
		if pingResult.AvgPacketLoss != nil {
			averages.APL = *pingResult.AvgPacketLoss
		}
	}

	// Calculate DNS averages from last 10 records
	dnsQuery := sys.manager.hub.DB().NewQuery(`
		SELECT AVG(lookup_time) as avg_lookup_time,
		       (COUNT(CASE WHEN status != 'success' THEN 1 END) * 100.0 / COUNT(*)) as failure_rate
		FROM (
			SELECT lookup_time, status
			FROM dns_stats 
			WHERE system = {:system}
			ORDER BY created DESC
			LIMIT 10
		)
	`).Bind(dbx.Params{
		"system": sys.Id,
	})

	dnsResult := struct {
		AvgLookupTime *float64 `db:"avg_lookup_time"`
		FailureRate   *float64 `db:"failure_rate"`
	}{}

	if err := dnsQuery.One(&dnsResult); err == nil {
		if dnsResult.AvgLookupTime != nil {
			averages.AD = *dnsResult.AvgLookupTime
		}
		if dnsResult.FailureRate != nil {
			averages.ADF = *dnsResult.FailureRate
		}
	}

	// Calculate HTTP averages from last 10 records
	httpQuery := sys.manager.hub.DB().NewQuery(`
		SELECT AVG(response_time) as avg_response_time,
		       (COUNT(CASE WHEN status != 'success' THEN 1 END) * 100.0 / COUNT(*)) as failure_rate
		FROM (
			SELECT response_time, status
			FROM http_stats 
			WHERE system = {:system}
			ORDER BY created DESC
			LIMIT 10
		)
	`).Bind(dbx.Params{
		"system": sys.Id,
	})

	httpResult := struct {
		AvgResponseTime *float64 `db:"avg_response_time"`
		FailureRate     *float64 `db:"failure_rate"`
	}{}

	if err := httpQuery.One(&httpResult); err == nil {
		if httpResult.AvgResponseTime != nil {
			averages.AH = *httpResult.AvgResponseTime
		}
		if httpResult.FailureRate != nil {
			averages.AHF = *httpResult.FailureRate
		}
	}

	// Calculate speedtest averages from last 10 records
	speedtestQuery := sys.manager.hub.DB().NewQuery(`
		SELECT AVG(download_speed) as avg_download, AVG(upload_speed) as avg_upload
		FROM (
			SELECT download_speed, upload_speed
			FROM speedtest_stats 
			WHERE system = {:system} AND status = 'success'
			ORDER BY created DESC
			LIMIT 10
		)
	`).Bind(dbx.Params{
		"system": sys.Id,
	})

	speedtestResult := struct {
		AvgDownload *float64 `db:"avg_download"`
		AvgUpload   *float64 `db:"avg_upload"`
	}{}

	if err := speedtestQuery.One(&speedtestResult); err == nil {
		if speedtestResult.AvgDownload != nil {
			averages.ADL = *speedtestResult.AvgDownload
		}
		if speedtestResult.AvgUpload != nil {
			averages.AUL = *speedtestResult.AvgUpload
		}
	}

	sys.manager.hub.Logger().Debug("Calculated averages", "system", sys.Id,
		"ping", averages.AP, "ping_loss", averages.APL,
		"dns", averages.AD, "dns_failure", averages.ADF,
		"http", averages.AH, "http_failure", averages.AHF,
		"download", averages.ADL, "upload", averages.AUL)

	// Update the system record with current averages
	systemCollection, err := sys.manager.hub.FindCollectionByNameOrId("systems")
	if err != nil {
		return err
	}

	systemRecord, err := sys.manager.hub.FindRecordById(systemCollection, sys.Id)
	if err != nil {
		return err
	}

	systemRecord.Set("current_averages", averages)

	if err := sys.manager.hub.Save(systemRecord); err != nil {
		return err
	}

	sys.manager.hub.Logger().Debug("Updated current averages for system", 
		"system", sys.Id,
		"ping_latency", averages.AP,
		"ping_packet_loss", averages.APL,
		"dns_latency", averages.AD,
		"dns_failure_rate", averages.ADF,
		"http_latency", averages.AH,
		"http_failure_rate", averages.AHF,
		"download_speed", averages.ADL,
		"upload_speed", averages.AUL)

	return nil
}

// getJitter returns a channel that will be triggered after a random delay
// between 40% and 90% of the interval.
// This is used to stagger the initial WebSocket connections to prevent clustering.
func getJitter() <-chan time.Time {
	minPercent := 40
	maxPercent := 90
	jitterRange := maxPercent - minPercent
	msDelay := (interval * minPercent / 100) + rand.Intn(interval*jitterRange/100)
	return time.After(time.Duration(msDelay) * time.Millisecond)
}
