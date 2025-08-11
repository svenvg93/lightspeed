package system

import "time"

type Stats struct {
	PingResults map[string]*PingResult `json:"ping,omitempty" cbor:"0,keyasint,omitempty"`
	DnsResults  map[string]*DnsResult  `json:"dns,omitempty" cbor:"1,keyasint,omitempty"`
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

type Info struct {
	Hostname     string  `json:"h" cbor:"0,keyasint"`
	AgentVersion string  `json:"v" cbor:"10,keyasint"`
	NetworkSpeed uint64  `json:"ns" cbor:"11,keyasint"`           // Network interface speed in Mbps
	PublicIP     string  `json:"ip" cbor:"12,keyasint"`           // Public IP address
	ISP          string  `json:"isp" cbor:"13,keyasint"`          // Internet Service Provider
	ASN          string  `json:"asn" cbor:"14,keyasint"`          // Autonomous System Number
	AvgPing      float64 `json:"ap" cbor:"15,keyasint,omitempty"` // Average ping across all targets (ms)
	AvgDns       float64 `json:"ad" cbor:"16,keyasint,omitempty"` // Average DNS lookup time across all targets (ms)
}

// Final data structure to return to the hub
type CombinedData struct {
	Stats Stats `json:"stats" cbor:"0,keyasint"`
	Info  Info  `json:"info" cbor:"1,keyasint"`
}
