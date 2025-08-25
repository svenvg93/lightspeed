package agent

import (
	"beszel/internal/entities/system"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/robfig/cron/v3"
)

type DnsManager struct {
	sync.RWMutex
	targets        map[string]*dnsTarget
	results        map[string]*system.DnsResult
	ctx            context.Context
	cancel         context.CancelFunc
	cronScheduler  *cron.Cron
	cronExpression string // Cron expression for DNS scheduling
}

type dnsTarget struct {
	system.DnsTarget
	lastLookup time.Time
}

// NewDnsManager creates a new DNS manager
func NewDnsManager() (*DnsManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	dm := &DnsManager{
		targets:        make(map[string]*dnsTarget),
		results:        make(map[string]*system.DnsResult),
		ctx:            ctx,
		cancel:         cancel,
		cronScheduler:  cron.New(cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow))),
		cronExpression: "", // Will be set by hub configuration (5-field format: minute hour day month weekday)
	}

	slog.Debug("DNS manager initialized - using miekg/dns with cron scheduling")

	// Start the cron scheduler
	dm.cronScheduler.Start()

	// Schedule the DNS job
	dm.scheduleDnsJob()

	return dm, nil
}

// UpdateConfig updates the DNS configuration with targets and cron expression
func (dm *DnsManager) UpdateConfig(targets []system.DnsTarget, cronExpression string) {
	dm.Lock()
	defer dm.Unlock()

	oldTargetsCount := len(dm.targets)
	oldResultsCount := len(dm.results)
	
	slog.Debug("UpdateConfig called", "old_targets", oldTargetsCount, "new_targets", len(targets), "cron_expression", cronExpression)

	// Update cron expression
	dm.cronExpression = cronExpression

	// Clear existing targets and results to prevent stale data
	dm.targets = make(map[string]*dnsTarget)
	dm.results = make(map[string]*system.DnsResult)
	
	if oldTargetsCount > 0 || oldResultsCount > 0 {
		slog.Info("Cleared old DNS configuration", "old_targets", oldTargetsCount, "old_results", oldResultsCount)
	}

	// Add new targets
	for _, target := range targets {
		// Fix timeout: if it's less than 1 second, assume it's in seconds and convert
		if target.Timeout < time.Second {
			target.Timeout = target.Timeout * time.Second
			slog.Debug("Converted timeout from seconds to duration", "original", target.Timeout/time.Second, "converted", target.Timeout)
		}
		if target.Timeout <= 0 {
			target.Timeout = 5 * time.Second
		}
		if target.Type == "" {
			target.Type = "A" // Default to A record
		}
		if target.Protocol == "" {
			target.Protocol = "udp" // Default to UDP
		}

		// Create a unique key for this target
		key := target.Domain + "@" + target.Server + "#" + target.Type

		dm.targets[key] = &dnsTarget{
			DnsTarget:  target,
			lastLookup: time.Time{}, // Will trigger immediate lookup
		}

		slog.Debug("Added DNS target", "domain", target.Domain, "server", target.Server, "type", target.Type, "protocol", target.Protocol, "timeout", target.Timeout)
	}

	// Reschedule the DNS job with new cron expression
	dm.scheduleDnsJob()

	slog.Debug("Updated DNS config", "targets", len(targets), "cron_expression", cronExpression)
}

// GetResults returns the current DNS results and clears them after retrieval
// Returns nil if no results are available (no DNS lookups have run recently)
func (dm *DnsManager) GetResults() map[string]*system.DnsResult {
	dm.Lock()
	defer dm.Unlock()

	// If no results are available, return nil to indicate no DNS lookups have run
	if len(dm.results) == 0 {
		return nil
	}

	// Create a copy to avoid race conditions
	results := make(map[string]*system.DnsResult)
	for key, result := range dm.results {
		results[key] = &system.DnsResult{
			Domain:      result.Domain,
			Server:      result.Server,
			Status:      result.Status,
			LookupTime:  result.LookupTime,
			ErrorCode:   result.ErrorCode,
			LastChecked: result.LastChecked,
		}
	}

	// Clear the results after they've been retrieved
	// This ensures DNS data is only sent once per test run
	dm.results = make(map[string]*system.DnsResult)

	return results
}

// Close shuts down the DNS manager
func (dm *DnsManager) Close() {
	dm.cronScheduler.Stop()
	dm.cancel()
}

// scheduleDnsJob schedules the DNS job with the current cron expression
func (dm *DnsManager) scheduleDnsJob() {
	// Remove all existing jobs
	dm.cronScheduler.Stop()
	dm.cronScheduler = cron.New(cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)))
	dm.cronScheduler.Start()

	// Only schedule if we have a valid cron expression
	if dm.cronExpression != "" {
		entryID, err := dm.cronScheduler.AddFunc(dm.cronExpression, func() {
			slog.Debug("Cron job triggered - running DNS lookups", "cron_expression", dm.cronExpression)
			dm.checkDnsLookups()
		})
		if err != nil {
			slog.Error("Failed to schedule DNS job", "cron_expression", dm.cronExpression, "error", err)
		} else {
			slog.Debug("Scheduled DNS job", "cron_expression", dm.cronExpression, "entry_id", entryID)
		}
	} else {
		slog.Debug("No cron expression set, DNS job not scheduled")
	}
}

// checkDnsLookups checks if any targets need to be looked up
func (dm *DnsManager) checkDnsLookups() {
	dm.RLock()
	targets := make([]*dnsTarget, 0, len(dm.targets))
	for _, target := range dm.targets {
		targets = append(targets, target)
	}
	dm.RUnlock()

	// Lookup targets concurrently
	var wg sync.WaitGroup
	for _, target := range targets {
		wg.Add(1)
		go func(t *dnsTarget) {
			defer wg.Done()
			dm.lookupTarget(t)
		}(target)
	}
	wg.Wait()
}

// lookupTarget performs a DNS lookup to a specific target
func (dm *DnsManager) lookupTarget(target *dnsTarget) {
	dm.Lock()
	target.lastLookup = time.Now()
	dm.Unlock()

	result := &system.DnsResult{
		Domain:      target.Domain,
		Server:      target.Server,
		Type:        target.Type,
		Status:      "testing",
		LastChecked: time.Now(),
	}

	dm.performDnsLookup(target, result)
}

// performDnsLookup performs a DNS lookup using the appropriate protocol
func (dm *DnsManager) performDnsLookup(target *dnsTarget, result *system.DnsResult) {
	protocol := target.Protocol
	if protocol == "" {
		protocol = "udp" // Default to UDP
	}

	slog.Debug("Starting DNS lookup", "domain", target.Domain, "server", target.Server, "type", target.Type, "protocol", protocol)

	// Set up context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), target.Timeout)
	defer cancel()

	// Perform the lookup based on protocol
	startTime := time.Now()
	var err error
	var resp *dns.Msg

	switch protocol {
	case "doh":
		resp, err = dm.performDoHLookup(ctx, target)
	case "dot":
		resp, err = dm.performDoTLookup(ctx, target)
	case "tcp":
		resp, err = dm.performTCPLookup(ctx, target)
	default: // "udp" or any other value
		resp, err = dm.performUDPLookup(ctx, target)
	}

	lookupTime := time.Since(startTime).Milliseconds()

	if err != nil {
		result.Status = "error"
		result.ErrorCode = err.Error()
		result.LookupTime = float64(lookupTime)
		slog.Debug("DNS lookup failed", "domain", target.Domain, "server", target.Server, "protocol", protocol, "error", err)
	} else if resp == nil {
		result.Status = "timeout"
		result.ErrorCode = "No response received"
		result.LookupTime = float64(lookupTime)
		slog.Debug("DNS lookup timeout - no response", "domain", target.Domain, "server", target.Server, "protocol", protocol)
	} else if resp.Rcode != dns.RcodeSuccess {
		result.Status = "error"
		result.ErrorCode = dns.RcodeToString[resp.Rcode]
		result.LookupTime = float64(lookupTime)
		slog.Debug("DNS lookup returned error code", "domain", target.Domain, "server", target.Server, "protocol", protocol, "rcode", resp.Rcode)
	} else {
		result.Status = "success"
		result.LookupTime = float64(lookupTime)
		slog.Debug("DNS lookup completed successfully", "domain", target.Domain, "server", target.Server, "protocol", protocol, "lookup_time", lookupTime)
	}

	// Create a unique key for this result
	key := target.Domain + "@" + target.Server + "#" + target.Type
	dm.updateResult(key, result)
}

// getDnsType converts string DNS type to miekg/dns type
func (dm *DnsManager) getDnsType(typeStr string) uint16 {
	switch strings.ToUpper(typeStr) {
	case "A":
		return dns.TypeA
	case "AAAA":
		return dns.TypeAAAA
	case "CNAME":
		return dns.TypeCNAME
	case "MX":
		return dns.TypeMX
	case "TXT":
		return dns.TypeTXT
	case "NS":
		return dns.TypeNS
	case "PTR":
		return dns.TypePTR
	case "SOA":
		return dns.TypeSOA
	case "SRV":
		return dns.TypeSRV
	case "CAA":
		return dns.TypeCAA
	default:
		return dns.TypeA // Default to A record
	}
}

// performUDPLookup performs a DNS lookup using UDP
func (dm *DnsManager) performUDPLookup(ctx context.Context, target *dnsTarget) (*dns.Msg, error) {
	// Add default port (53) if no port is specified
	serverAddr := target.Server
	if !strings.Contains(serverAddr, ":") {
		serverAddr = serverAddr + ":53"
		slog.Debug("Added default port to DNS server", "original", target.Server, "with_port", serverAddr)
	}

	// Create a DNS client
	client := &dns.Client{
		Timeout: target.Timeout,
		Net:     "udp",
	}

	// Create a DNS message
	msg := &dns.Msg{}
	msg.SetQuestion(dns.Fqdn(target.Domain), dm.getDnsType(target.Type))
	msg.RecursionDesired = true

	// Perform the lookup
	slog.Debug("Attempting UDP DNS lookup", "domain", target.Domain, "server", serverAddr, "timeout", target.Timeout)
	resp, _, err := client.ExchangeContext(ctx, msg, serverAddr)
	return resp, err
}

// performTCPLookup performs a DNS lookup using TCP
func (dm *DnsManager) performTCPLookup(ctx context.Context, target *dnsTarget) (*dns.Msg, error) {
	// Add default port (53) if no port is specified
	serverAddr := target.Server
	if !strings.Contains(serverAddr, ":") {
		serverAddr = serverAddr + ":53"
		slog.Debug("Added default port to DNS server", "original", target.Server, "with_port", serverAddr)
	}

	// Create a DNS client
	client := &dns.Client{
		Timeout: target.Timeout,
		Net:     "tcp",
	}

	// Create a DNS message
	msg := &dns.Msg{}
	msg.SetQuestion(dns.Fqdn(target.Domain), dm.getDnsType(target.Type))
	msg.RecursionDesired = true

	// Perform the lookup
	slog.Debug("Attempting TCP DNS lookup", "domain", target.Domain, "server", serverAddr, "timeout", target.Timeout)
	resp, _, err := client.ExchangeContext(ctx, msg, serverAddr)
	return resp, err
}

// performDoTLookup performs a DNS lookup using DNS over TLS
func (dm *DnsManager) performDoTLookup(ctx context.Context, target *dnsTarget) (*dns.Msg, error) {
	// Add default port (853) if no port is specified
	serverAddr := target.Server
	if !strings.Contains(serverAddr, ":") {
		serverAddr = serverAddr + ":853"
		slog.Debug("Added default DoT port to DNS server", "original", target.Server, "with_port", serverAddr)
	}

	// Create a DNS client
	client := &dns.Client{
		Timeout: target.Timeout,
		Net:     "tcp-tls",
	}

	// Create a DNS message
	msg := &dns.Msg{}
	msg.SetQuestion(dns.Fqdn(target.Domain), dm.getDnsType(target.Type))
	msg.RecursionDesired = true

	// Perform the lookup
	slog.Debug("Attempting DoT DNS lookup", "domain", target.Domain, "server", serverAddr, "timeout", target.Timeout)
	resp, _, err := client.ExchangeContext(ctx, msg, serverAddr)
	return resp, err
}

// performDoHLookup performs a DNS lookup using DNS over HTTPS
func (dm *DnsManager) performDoHLookup(ctx context.Context, target *dnsTarget) (*dns.Msg, error) {
	// Create a DNS message
	msg := &dns.Msg{}
	msg.SetQuestion(dns.Fqdn(target.Domain), dm.getDnsType(target.Type))
	msg.RecursionDesired = true

	// Encode the DNS message to wire format
	dnsWire, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("failed to pack DNS message: %w", err)
	}

	// Encode to base64 for GET request or use raw bytes for POST
	dnsBase64 := base64.RawURLEncoding.EncodeToString(dnsWire)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: target.Timeout,
	}

	// Try GET method first (more widely supported)
	getURL := target.Server + "?dns=" + dnsBase64
	slog.Debug("Attempting DoH GET request", "domain", target.Domain, "server", target.Server, "url", getURL)

	req, err := http.NewRequestWithContext(ctx, "GET", getURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	// Set required headers for DoH
	req.Header.Set("Accept", "application/dns-message")
	req.Header.Set("User-Agent", "Beszel-DNS-Agent/1.0")

	resp, err := client.Do(req)
	if err != nil {
		// If GET fails, try POST method
		slog.Debug("DoH GET failed, trying POST", "domain", target.Domain, "server", target.Server, "error", err)

		req, err = http.NewRequestWithContext(ctx, "POST", target.Server, strings.NewReader(string(dnsWire)))
		if err != nil {
			return nil, fmt.Errorf("failed to create POST request: %w", err)
		}

		req.Header.Set("Content-Type", "application/dns-message")
		req.Header.Set("Accept", "application/dns-message")
		req.Header.Set("User-Agent", "Beszel-DNS-Agent/1.0")

		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("DoH POST request failed: %w", err)
		}
	}

	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DoH request failed with status: %s", resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read DoH response: %w", err)
	}

	// Parse DNS response
	dnsResp := &dns.Msg{}
	err = dnsResp.Unpack(body)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack DNS response: %w", err)
	}

	slog.Debug("DoH lookup completed", "domain", target.Domain, "server", target.Server, "response_rcode", dnsResp.Rcode)
	return dnsResp, nil
}

// updateResult updates the DNS result for a target
func (dm *DnsManager) updateResult(key string, result *system.DnsResult) {
	dm.Lock()
	defer dm.Unlock()
	slog.Debug("Adding DNS result", "key", key, "status", result.Status, "lookup_time", result.LookupTime, "results_count_before", len(dm.results))
	dm.results[key] = result
	slog.Debug("DNS result updated", "key", key, "status", result.Status, "lookup_time", result.LookupTime, "results_count_after", len(dm.results))
}
