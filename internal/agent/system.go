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

// getAllNetworkInterfaces returns information about all network interfaces
// This can be used for debugging or future enhancements
func (a *Agent) getAllNetworkInterfaces() []struct {
	Name      string `json:"name"`
	Speed     uint64 `json:"speed_mbps"`
	IsVirtual bool   `json:"is_virtual"`
} {
	netInfo, err := ghwnet.New()
	if err != nil {
		slog.Debug("Failed to get network info", "error", err)
		return nil
	}

	var interfaces []struct {
		Name      string `json:"name"`
		Speed     uint64 `json:"speed_mbps"`
		IsVirtual bool   `json:"is_virtual"`
	}

	for _, nic := range netInfo.NICs {
		speedMbps := uint64(0)
		if nic.Speed != "" {
			speedMbps = a.parseSpeedString(nic.Speed)
		}

		interfaces = append(interfaces, struct {
			Name      string `json:"name"`
			Speed     uint64 `json:"speed_mbps"`
			IsVirtual bool   `json:"is_virtual"`
		}{
			Name:      nic.Name,
			Speed:     speedMbps,
			IsVirtual: nic.IsVirtual,
		})
	}

	return interfaces
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
		} else {
			slog.Debug("No ping results available - no tests have run recently")
		}
	} else {
		slog.Debug("No ping manager available")
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
		} else {
			slog.Debug("No DNS results available - no lookups have run recently")
		}
	} else {
		slog.Debug("No DNS manager available")
	}

	// get HTTP results if HTTP manager is available
	if a.httpManager != nil {
		httpResults := a.httpManager.GetResults()
		if httpResults != nil {
			systemStats.HttpResults = httpResults
			slog.Debug("HTTP results collected", "count", len(systemStats.HttpResults))
		} else {
			slog.Debug("No HTTP results available - no checks have run recently")
		}
	} else {
		slog.Debug("No HTTP manager available")
	}

	// get speedtest results if speedtest manager is available
	if a.speedtestManager != nil {
		speedtestResults := a.speedtestManager.GetResults()
		if speedtestResults != nil {
			systemStats.SpeedtestResults = speedtestResults
			slog.Debug("Speedtest results collected", "count", len(systemStats.SpeedtestResults))
			// Debug log each speedtest result
			for serverID, result := range systemStats.SpeedtestResults {
				slog.Debug("Speedtest result from manager", "server_id", serverID, "download", result.DownloadSpeed, "upload", result.UploadSpeed, "latency", result.Latency, "last_checked", result.LastChecked)
			}
		} else {
			slog.Debug("No speedtest results available - no tests have run recently")
		}
	} else {
		slog.Debug("No speedtest manager available")
	}

	slog.Debug("sysinfo", "data", a.systemInfo)

	return systemStats
}
