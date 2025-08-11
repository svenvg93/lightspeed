package system

import "time"

type Stats struct {
	PingResults      map[string]*PingResult      `json:"ping,omitempty" cbor:"0,keyasint,omitempty"`
	DnsResults       map[string]*DnsResult       `json:"dns,omitempty" cbor:"1,keyasint,omitempty"`
	HttpResults      map[string]*HttpResult      `json:"http,omitempty" cbor:"2,keyasint,omitempty"`
	SpeedtestResults map[string]*SpeedtestResult `json:"speedtest,omitempty" cbor:"3,keyasint,omitempty"`
}

type PingResult struct {
	Host        string    `json:"host" cbor:"0,keyasint"`
	PacketLoss  float64   `json:"loss" cbor:"1,keyasint"`    // Percentage
	MinRtt      float64   `json:"min_rtt" cbor:"2,keyasint"` // Milliseconds
	MaxRtt      float64   `json:"max_rtt" cbor:"3,keyasint"` // Milliseconds
	AvgRtt      float64   `json:"avg_rtt" cbor:"4,keyasint"` // Milliseconds
	LastChecked time.Time `json:"last_checked" cbor:"5,keyasint"`
}

type PingTarget struct {
	Host    string        `json:"host"`
	Count   int           `json:"count"`
	Timeout time.Duration `json:"timeout"`
}

type DnsResult struct {
	Domain      string    `json:"domain" cbor:"0,keyasint"`
	Server      string    `json:"server" cbor:"1,keyasint"`
	Type        string    `json:"type" cbor:"2,keyasint"`        // "A", "AAAA", "MX", "TXT", etc.
	Status      string    `json:"status" cbor:"3,keyasint"`      // "success", "timeout", "error"
	LookupTime  float64   `json:"lookup_time" cbor:"4,keyasint"` // Milliseconds
	ErrorCode   string    `json:"error_code,omitempty" cbor:"5,keyasint,omitempty"`
	LastChecked time.Time `json:"last_checked" cbor:"6,keyasint"`
}

type DnsTarget struct {
	Domain   string        `json:"domain"`
	Server   string        `json:"server"`
	Type     string        `json:"type"` // "A", "AAAA", "MX", "TXT", etc.
	Timeout  time.Duration `json:"timeout"`
	Protocol string        `json:"protocol,omitempty"` // "udp", "tcp", "doh", "dot"
}

type HttpResult struct {
	URL          string    `json:"url" cbor:"0,keyasint"`
	Status       string    `json:"status" cbor:"1,keyasint"`        // "success", "timeout", "error"
	ResponseTime float64   `json:"response_time" cbor:"2,keyasint"` // Milliseconds
	StatusCode   int       `json:"status_code" cbor:"3,keyasint"`
	ErrorCode    string    `json:"error_code,omitempty" cbor:"4,keyasint,omitempty"`
	LastChecked  time.Time `json:"last_checked" cbor:"5,keyasint"`
}

type HttpTarget struct {
	URL     string `json:"url"`
	Timeout int    `json:"timeout"` // Timeout in seconds
}

type SpeedtestResult struct {
	ServerURL     string    `json:"server_url" cbor:"0,keyasint"`
	Status        string    `json:"status" cbor:"1,keyasint"`         // "success", "timeout", "error"
	DownloadSpeed float64   `json:"download_speed" cbor:"2,keyasint"` // Mbps
	UploadSpeed   float64   `json:"upload_speed" cbor:"3,keyasint"`   // Mbps
	Latency       float64   `json:"latency" cbor:"4,keyasint"`        // Milliseconds
	ErrorCode     string    `json:"error_code,omitempty" cbor:"5,keyasint,omitempty"`
	LastChecked   time.Time `json:"last_checked" cbor:"6,keyasint"`
	// Additional detailed information
	PingJitter            float64 `json:"ping_jitter,omitempty" cbor:"7,keyasint,omitempty"`
	PingLow               float64 `json:"ping_low,omitempty" cbor:"8,keyasint,omitempty"`
	PingHigh              float64 `json:"ping_high,omitempty" cbor:"9,keyasint,omitempty"`
	DownloadBytes         int64   `json:"download_bytes,omitempty" cbor:"10,keyasint,omitempty"`
	DownloadElapsed       int64   `json:"download_elapsed,omitempty" cbor:"11,keyasint,omitempty"`
	DownloadLatencyIQM    float64 `json:"download_latency_iqm,omitempty" cbor:"12,keyasint,omitempty"`
	DownloadLatencyLow    float64 `json:"download_latency_low,omitempty" cbor:"13,keyasint,omitempty"`
	DownloadLatencyHigh   float64 `json:"download_latency_high,omitempty" cbor:"14,keyasint,omitempty"`
	DownloadLatencyJitter float64 `json:"download_latency_jitter,omitempty" cbor:"15,keyasint,omitempty"`
	UploadBytes           int64   `json:"upload_bytes,omitempty" cbor:"16,keyasint,omitempty"`
	UploadElapsed         int64   `json:"upload_elapsed,omitempty" cbor:"17,keyasint,omitempty"`
	UploadLatencyIQM      float64 `json:"upload_latency_iqm,omitempty" cbor:"18,keyasint,omitempty"`
	UploadLatencyLow      float64 `json:"upload_latency_low,omitempty" cbor:"19,keyasint,omitempty"`
	UploadLatencyHigh     float64 `json:"upload_latency_high,omitempty" cbor:"20,keyasint,omitempty"`
	UploadLatencyJitter   float64 `json:"upload_latency_jitter,omitempty" cbor:"21,keyasint,omitempty"`
	PacketLoss            int     `json:"packet_loss,omitempty" cbor:"22,keyasint,omitempty"`
	ISP                   string  `json:"isp,omitempty" cbor:"23,keyasint,omitempty"`
	InterfaceExternalIP   string  `json:"interface_external_ip,omitempty" cbor:"24,keyasint,omitempty"`
	ServerName            string  `json:"server_name,omitempty" cbor:"25,keyasint,omitempty"`
	ServerLocation        string  `json:"server_location,omitempty" cbor:"26,keyasint,omitempty"`
	ServerCountry         string  `json:"server_country,omitempty" cbor:"27,keyasint,omitempty"`
	ServerHost            string  `json:"server_host,omitempty" cbor:"28,keyasint,omitempty"`
	ServerIP              string  `json:"server_ip,omitempty" cbor:"29,keyasint,omitempty"`
}

type SpeedtestTarget struct {
	ServerID string        `json:"server_id"`
	Timeout  time.Duration `json:"timeout"`
}

// Unified monitoring configuration
type MonitoringConfig struct {
	Enabled struct {
		Ping      bool `json:"ping"`
		Dns       bool `json:"dns"`
		Http      bool `json:"http,omitempty"`
		Speedtest bool `json:"speedtest,omitempty"`
	} `json:"enabled"`
	GlobalInterval string `json:"global_interval,omitempty"` // Cron expression
	Ping           struct {
		Targets  []PingTarget `json:"targets"`
		Interval string       `json:"interval,omitempty"` // Override global interval
	} `json:"ping,omitempty"`
	Dns struct {
		Targets  []DnsTarget `json:"targets"`
		Interval string      `json:"interval,omitempty"` // Override global interval
	} `json:"dns,omitempty"`
	Http struct {
		Targets  []HttpTarget `json:"targets"`
		Interval string       `json:"interval,omitempty"` // Override global interval
	} `json:"http,omitempty"`
	Speedtest struct {
		Targets  []SpeedtestTarget `json:"targets"`
		Interval string            `json:"interval,omitempty"` // Override global interval
	} `json:"speedtest,omitempty"`
}

type Info struct {
	Hostname     string  `json:"h" cbor:"0,keyasint"`
	AgentVersion string  `json:"v" cbor:"10,keyasint"`
	NetworkSpeed uint64  `json:"ns" cbor:"11,keyasint"`            // Network interface speed in Mbps
	PublicIP     string  `json:"ip" cbor:"12,keyasint"`            // Public IP address
	ISP          string  `json:"isp" cbor:"13,keyasint"`           // Internet Service Provider
	ASN          string  `json:"asn" cbor:"14,keyasint"`           // Autonomous System Number
	AvgPing      float64 `json:"ap" cbor:"15,keyasint,omitempty"`  // Average ping across all targets (ms)
	AvgDns       float64 `json:"ad" cbor:"16,keyasint,omitempty"`  // Average DNS lookup time across all targets (ms)
	AvgHttp      float64 `json:"ah" cbor:"17,keyasint,omitempty"`  // Average HTTP response time across all targets (ms)
	AvgDownload  float64 `json:"adl" cbor:"18,keyasint,omitempty"` // Average download speed across all speedtest targets (Mbps)
	AvgUpload    float64 `json:"aul" cbor:"19,keyasint,omitempty"` // Average upload speed across all speedtest targets (Mbps)
}

// Final data structure to return to the hub
type CombinedData struct {
	Stats Stats `json:"stats" cbor:"0,keyasint"`
	Info  Info  `json:"info" cbor:"1,keyasint"`
}
