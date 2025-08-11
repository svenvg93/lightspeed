package agent

import (
	"beszel"
	"beszel/internal/entities/system"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	ghwnet "github.com/jaypipes/ghw/pkg/net"
)

// Sets initial / non-changing values about the host system
func (a *Agent) initializeSystemInfo() {
	a.systemInfo.AgentVersion = beszel.Version
	a.systemInfo.Hostname, _ = os.Hostname()

	// Get network interface speed
	a.systemInfo.NetworkSpeed = a.getNetworkSpeed()

	// Get public IP, ISP, and ASN information
	a.getIPInfo()
}

// GeoJSResponse represents the response from the GeoJS API
type GeoJSResponse struct {
	Organization     string `json:"organization"`
	Country          string `json:"country"`
	OrganizationName string `json:"organization_name"`
	CountryCode      string `json:"country_code"`
	ASN              int    `json:"asn"`
	Region           string `json:"region"`
	IP               string `json:"ip"`
	City             string `json:"city"`
}

// getIPInfo collects public IP, ISP, and ASN information using GeoJS API
func (a *Agent) getIPInfo() {
	// Make HTTP request to GeoJS API
	resp, err := http.Get("https://get.geojs.io/v1/ip/geo.json")
	if err != nil {
		slog.Debug("Failed to get IP info from GeoJS", "error", err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Debug("Failed to read GeoJS response", "error", err)
		return
	}

	// Parse JSON response
	var geoInfo GeoJSResponse
	if err := json.Unmarshal(body, &geoInfo); err != nil {
		slog.Debug("Failed to parse GeoJS response", "error", err)
		return
	}

	// Set the collected information
	a.systemInfo.PublicIP = geoInfo.IP
	a.systemInfo.ISP = geoInfo.OrganizationName
	if geoInfo.ASN > 0 {
		a.systemInfo.ASN = fmt.Sprintf("AS%d", geoInfo.ASN)
	}

	slog.Debug("IP info collected from GeoJS",
		"ip", a.systemInfo.PublicIP,
		"isp", a.systemInfo.ISP,
		"asn", a.systemInfo.ASN,
		"city", geoInfo.City,
		"country", geoInfo.Country)
}

// getNetworkSpeed returns the speed of the primary network interface in Mbps
func (a *Agent) getNetworkSpeed() uint64 {
	netInfo, err := ghwnet.New()
	if err != nil {
		slog.Debug("Failed to get network info", "error", err)
		return 0
	}

	// Find the first active network interface with a valid speed
	for _, nic := range netInfo.NICs {
		if nic.IsVirtual {
			continue // Skip virtual interfaces
		}

		if nic.Speed != "" {
			// Parse speed string like "1000Mb/s" or "1Gb/s"
			speedMbps := a.parseSpeedString(nic.Speed)
			if speedMbps > 0 {
				slog.Debug("Found network interface", "name", nic.Name, "speed", nic.Speed, "speed_mbps", speedMbps)
				return speedMbps
			}
		}
	}

	slog.Debug("No network interface with valid speed found")
	return 0
}

// parseSpeedString parses speed strings like "1000Mb/s" or "1Gb/s" and returns Mbps
func (a *Agent) parseSpeedString(speed string) uint64 {
	// Common speed patterns: "1000Mb/s", "1Gb/s", "100Mb/s", etc.
	var value float64
	var unit string

	// Try to parse patterns like "1000Mb/s" or "1Gb/s"
	if _, err := fmt.Sscanf(speed, "%f%s", &value, &unit); err != nil {
		slog.Debug("Failed to parse speed string", "speed", speed, "error", err)
		return 0
	}

	// Convert to Mbps based on unit
	switch {
	case strings.HasPrefix(unit, "Gb"):
		return uint64(value * 1000) // 1 Gb = 1000 Mb
	case strings.HasPrefix(unit, "Mb"):
		return uint64(value)
	case strings.HasPrefix(unit, "Kb"):
		return uint64(value / 1000) // 1000 Kb = 1 Mb
	default:
		slog.Debug("Unknown speed unit", "unit", unit, "speed", speed)
		return 0
	}
}

// Returns current info, stats about the host system
func (a *Agent) getSystemStats() system.Stats {
	systemStats := system.Stats{}

	// get ping results if ping manager is available
	if a.pingManager != nil {
		pingResults := a.pingManager.GetResults()
		if pingResults != nil {
			systemStats.PingResults = pingResults
			slog.Debug("Ping results collected", "count", len(systemStats.PingResults))

			// Calculate average ping across all successful ping results
			var totalPing float64
			var successfulPings int
			for _, result := range systemStats.PingResults {
				if result.AvgRtt > 0 {
					totalPing += result.AvgRtt
					successfulPings++
				}
			}

			if successfulPings > 0 {
				a.systemInfo.AvgPing = totalPing / float64(successfulPings)
				slog.Debug("Average ping calculated", "ap", a.systemInfo.AvgPing, "successful_pings", successfulPings)
			} else {
				a.systemInfo.AvgPing = 0 // Reset to 0 if no successful pings
			}
		} else {
			slog.Debug("No ping results available - no tests have run recently")
			a.systemInfo.AvgPing = 0 // Reset to 0 if no ping results
		}
	} else {
		slog.Debug("No ping manager available")
		a.systemInfo.AvgPing = 0 // Reset to 0 if no ping manager
	}

	// get DNS results if DNS manager is available
	if a.dnsManager != nil {
		dnsResults := a.dnsManager.GetResults()
		if dnsResults != nil {
			systemStats.DnsResults = dnsResults
			slog.Debug("DNS results collected", "count", len(systemStats.DnsResults))
			for key, result := range systemStats.DnsResults {
				slog.Debug("DNS result", "key", key, "domain", result.Domain, "server", result.Server, "status", result.Status, "lookup_time", result.LookupTime)
			}

			// Calculate average DNS lookup time across all successful DNS results
			var totalDns float64
			var successfulDns int
			for _, result := range systemStats.DnsResults {
				if result.Status == "success" && result.LookupTime > 0 {
					totalDns += result.LookupTime
					successfulDns++
				}
			}

			if successfulDns > 0 {
				a.systemInfo.AvgDns = totalDns / float64(successfulDns)
				slog.Debug("Average DNS lookup time calculated", "ad", a.systemInfo.AvgDns, "successful_dns", successfulDns)
			} else {
				a.systemInfo.AvgDns = 0 // Reset to 0 if no successful DNS lookups
			}
		} else {
			slog.Debug("No DNS results available - no lookups have run recently")
			a.systemInfo.AvgDns = 0 // Reset to 0 if no DNS results
		}
	} else {
		slog.Debug("No DNS manager available")
		a.systemInfo.AvgDns = 0 // Reset to 0 if no DNS manager
	}

	// get HTTP results if HTTP manager is available
	if a.httpManager != nil {
		httpResults := a.httpManager.GetResults()
		if httpResults != nil {
			systemStats.HttpResults = httpResults
			slog.Debug("HTTP results collected", "count", len(systemStats.HttpResults))

			// Calculate average HTTP response time across all successful HTTP results
			var totalHttp float64
			var successfulHttp int
			for _, result := range systemStats.HttpResults {
				if result.Status == "success" && result.ResponseTime > 0 {
					totalHttp += result.ResponseTime
					successfulHttp++
				}
			}

			if successfulHttp > 0 {
				a.systemInfo.AvgHttp = totalHttp / float64(successfulHttp)
				slog.Debug("Average HTTP response time calculated", "ah", a.systemInfo.AvgHttp, "successful_http", successfulHttp)
			} else {
				a.systemInfo.AvgHttp = 0 // Reset to 0 if no successful HTTP requests
			}
		} else {
			slog.Debug("No HTTP results available - no checks have run recently")
			a.systemInfo.AvgHttp = 0 // Reset to 0 if no HTTP results
		}
	} else {
		slog.Debug("No HTTP manager available")
		a.systemInfo.AvgHttp = 0 // Reset to 0 if no HTTP manager
	}

	// get speedtest results if speedtest manager is available
	if a.speedtestManager != nil {
		speedtestResults := a.speedtestManager.GetResults()
		if speedtestResults != nil {
			systemStats.SpeedtestResults = speedtestResults
			slog.Debug("Speedtest results collected", "count", len(systemStats.SpeedtestResults))

			// Calculate average download and upload speeds across all successful speedtest results
			var totalDownload float64
			var totalUpload float64
			var successfulSpeedtest int
			for _, result := range systemStats.SpeedtestResults {
				if result.Status == "success" && result.DownloadSpeed > 0 && result.UploadSpeed > 0 {
					totalDownload += result.DownloadSpeed
					totalUpload += result.UploadSpeed
					successfulSpeedtest++
				}
			}

			if successfulSpeedtest > 0 {
				a.systemInfo.AvgDownload = totalDownload / float64(successfulSpeedtest)
				a.systemInfo.AvgUpload = totalUpload / float64(successfulSpeedtest)
				slog.Debug("Average speedtest speeds calculated",
					"avg_download", a.systemInfo.AvgDownload,
					"avg_upload", a.systemInfo.AvgUpload,
					"successful_speedtest", successfulSpeedtest)
			} else {
				a.systemInfo.AvgDownload = 0 // Reset to 0 if no successful speedtests
				a.systemInfo.AvgUpload = 0   // Reset to 0 if no successful speedtests
			}
		} else {
			slog.Debug("No speedtest results available - no tests have run recently")
			a.systemInfo.AvgDownload = 0 // Reset to 0 if no speedtest results
			a.systemInfo.AvgUpload = 0   // Reset to 0 if no speedtest results
		}
	} else {
		slog.Debug("No speedtest manager available")
		a.systemInfo.AvgDownload = 0 // Reset to 0 if no speedtest manager
		a.systemInfo.AvgUpload = 0   // Reset to 0 if no speedtest manager
	}

	slog.Debug("sysinfo", "data", a.systemInfo)

	return systemStats
}
