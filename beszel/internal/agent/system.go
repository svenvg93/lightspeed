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

			// Calculate average ping across all ping results
			var totalPing float64
			var pingCount int
			for _, result := range systemStats.PingResults {
				if result.AvgRtt > 0 {
					totalPing += result.AvgRtt
					pingCount++
				}
			}

			if pingCount > 0 {
				a.systemInfo.AvgPing = totalPing / float64(pingCount)
			} else {
				a.systemInfo.AvgPing = 0 // Reset to 0 if no ping results
			}
		} else {

			a.systemInfo.AvgPing = 0 // Reset to 0 if no ping results
		}
	} else {

		a.systemInfo.AvgPing = 0 // Reset to 0 if no ping manager
	}

	slog.Debug("sysinfo", "data", a.systemInfo)

	return systemStats
}
