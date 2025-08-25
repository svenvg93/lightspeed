import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Card, CardContent } from "@/components/ui/card"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { PlusIcon, TrashIcon, ActivityIcon } from "lucide-react"

export interface PingTarget {
  host: string
  friendly_name?: string
  count: number
  timeout: number
}

export interface DnsTarget {
  domain: string
  server: string
  type: string
  timeout: number
  friendly_name?: string
  protocol?: "udp" | "tcp" | "doh" | "dot"
}

export interface HttpTarget {
  url: string
  friendly_name?: string
  timeout: number
}

export interface SpeedtestTarget {
  server_id: string
  friendly_name?: string
  timeout: number
}

export function ExpectedPerformanceTab({
  pingConfig,
  setPingConfig,
  dnsConfig,
  setDnsConfig,
  httpConfig,
  setHttpConfig,
  speedtestConfig,
  setSpeedtestConfig
}: {
  pingConfig: { targets: PingTarget[], interval: string, expected_latency?: number }
  setPingConfig: (config: { targets: PingTarget[], interval: string, expected_latency?: number }) => void
  dnsConfig: { targets: DnsTarget[], interval: string, expected_lookup_time?: number }
  setDnsConfig: (config: { targets: DnsTarget[], interval: string, expected_lookup_time?: number }) => void
  httpConfig: { targets: HttpTarget[], interval: string, expected_response_time?: number }
  setHttpConfig: (config: { targets: HttpTarget[], interval: string, expected_response_time?: number }) => void
  speedtestConfig: { targets: SpeedtestTarget[], interval: string }
  setSpeedtestConfig: (config: { targets: SpeedtestTarget[], interval: string }) => void
}): JSX.Element {
  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <h3 className="text-lg font-medium">Performance Thresholds</h3>
        <p className="text-sm text-muted-foreground">
          Set expected performance thresholds to display colored indicators in the systems table.
        </p>
      </div>

      <Separator />

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">


        {/* Ping Latency */}
        <div className="space-y-2">
          <Label htmlFor="expected-ping-latency">Expected Ping Latency (ms)</Label>
          <Input
            id="expected-ping-latency"
            type="number"
            min="0"
            step="0.1"
            placeholder="50"
            value={pingConfig.expected_latency || ''}
            onChange={(e) => setPingConfig({
              ...pingConfig,
              expected_latency: e.target.value ? parseFloat(e.target.value) : undefined
            })}
          />
          <p className="text-xs text-muted-foreground">
            Green: &lt;80% of expected, Yellow: 50-80%, Red: &lt;50%
          </p>
        </div>

        {/* DNS Lookup Time */}
        <div className="space-y-2">
          <Label htmlFor="expected-dns-lookup-time">Expected DNS Lookup Time (ms)</Label>
          <Input
            id="expected-dns-lookup-time"
            type="number"
            min="0"
            step="0.1"
            placeholder="100"
            value={dnsConfig.expected_lookup_time || ''}
            onChange={(e) => setDnsConfig({
              ...dnsConfig,
              expected_lookup_time: e.target.value ? parseFloat(e.target.value) : undefined
            })}
          />
          <p className="text-xs text-muted-foreground">
            Green: &lt;80% of expected, Yellow: 50-80%, Red: &lt;50%
          </p>
        </div>

        {/* HTTP Response Time */}
        <div className="space-y-2">
          <Label htmlFor="expected-http-response-time">Expected HTTP Response Time (ms)</Label>
          <Input
            id="expected-http-response-time"
            type="number"
            min="0"
            step="0.1"
            placeholder="200"
            value={httpConfig.expected_response_time || ''}
            onChange={(e) => setHttpConfig({
              ...httpConfig,
              expected_response_time: e.target.value ? parseFloat(e.target.value) : undefined
            })}
          />
          <p className="text-xs text-muted-foreground">
            Green: &lt;80% of expected, Yellow: 50-80%, Red: &lt;50%
          </p>
        </div>
      </div>
    </div>
  )
}

const DNS_TYPES = [
  { value: "A", label: "A (IPv4 Address)" },
  { value: "AAAA", label: "AAAA (IPv6 Address)" },
  { value: "MX", label: "MX (Mail Exchange)" },
  { value: "TXT", label: "TXT (Text Record)" },
  { value: "CNAME", label: "CNAME (Canonical Name)" },
  { value: "NS", label: "NS (Name Server)" },
  { value: "PTR", label: "PTR (Pointer)" },
  { value: "SOA", label: "SOA (Start of Authority)" },
]

const DNS_PROTOCOLS = [
  { value: "udp", label: "UDP (Standard DNS)" },
  { value: "tcp", label: "TCP (DNS over TCP)" },
  { value: "doh", label: "DoH (DNS over HTTPS)" },
  { value: "dot", label: "DoT (DNS over TLS)" },
]

export function PingConfigTab({ 
  pingConfig, 
  setPingConfig 
}: { 
  pingConfig: { targets: PingTarget[], interval: string, expected_latency?: number }
  setPingConfig: (config: { targets: PingTarget[], interval: string, expected_latency?: number }) => void
}): JSX.Element {

  const addTarget = () => {
    setPingConfig({
      ...pingConfig,
      targets: [
        ...pingConfig.targets,
        {
          host: '',
          friendly_name: '',
          count: 4,
          timeout: 5
        }
      ]
    })
  }

  const removeTarget = (index: number) => {
    setPingConfig({
      ...pingConfig,
      targets: pingConfig.targets.filter((_, i) => i !== index)
    })
  }

  const updateTargetString = (index: number, field: 'host' | 'friendly_name', value: string) => {
    setPingConfig({
      ...pingConfig,
      targets: pingConfig.targets.map((target, i) => 
        i === index 
          ? { ...target, [field]: value }
          : target
      )
    })
  }

  const updateTargetNumber = (index: number, field: 'count' | 'timeout', value: number) => {
    setPingConfig({
      ...pingConfig,
      targets: pingConfig.targets.map((target, i) => 
        i === index 
          ? { ...target, [field]: value }
          : target
      )
    })
  }

  return (
    <div className="space-y-6">
      {/* Check Interval at the top */}
      <div className="space-y-2">
        <Label htmlFor="ping-interval">Ping Interval</Label>
        <Input
          id="ping-interval"
          placeholder="*/3 * * * *"
          value={pingConfig.interval}
          onChange={(e) => setPingConfig({ ...pingConfig, interval: e.target.value })}
        />
        <p className="text-sm text-muted-foreground">
          Cron expression (e.g., "*/3 * * * *" for every 3 minutes)
        </p>
      </div>



      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-medium">Ping Targets</h3>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={addTarget}
            className="flex items-center gap-2"
          >
            <PlusIcon className="h-4 w-4" />
            Add Target
          </Button>
        </div>

        {pingConfig.targets.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-8 text-center">
              <ActivityIcon className="h-12 w-12 text-muted-foreground mb-4" />
              <p className="text-muted-foreground mb-2">No ping targets configured</p>
              <p className="text-sm text-muted-foreground">
                Add ping targets to monitor network connectivity.
              </p>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-4">
            {pingConfig.targets.map((target, index) => (
              <Card key={index}>
                <CardContent className="p-4">
                  <div className="flex items-center justify-between mb-4">
                    <h4 className="font-medium">{target.friendly_name || `Target ${index + 1}`}</h4>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => removeTarget(index)}
                      className="flex items-center gap-2"
                    >
                      <TrashIcon className="h-4 w-4" />
                      Remove
                    </Button>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label>Host</Label>
                      <Input
                        placeholder="8.8.8.8"
                        value={target.host}
                        onChange={(e) => updateTargetString(index, 'host', e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Friendly Name (Optional)</Label>
                      <Input
                        placeholder="Google DNS"
                        value={target.friendly_name || ''}
                        onChange={(e) => updateTargetString(index, 'friendly_name', e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Count</Label>
                      <Input
                        type="number"
                        min="1"
                        value={target.count}
                        onChange={(e) => updateTargetNumber(index, 'count', parseInt(e.target.value) || 1)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Timeout (seconds)</Label>
                      <Input
                        type="number"
                        min="1"
                        value={target.timeout}
                        onChange={(e) => updateTargetNumber(index, 'timeout', parseInt(e.target.value) || 1)}
                      />
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export function DnsConfigTab({ 
  dnsConfig, 
  setDnsConfig 
}: { 
  dnsConfig: { targets: DnsTarget[], interval: string, expected_lookup_time?: number }
  setDnsConfig: (config: { targets: DnsTarget[], interval: string, expected_lookup_time?: number }) => void
}): JSX.Element {

  const addTarget = () => {
    setDnsConfig({
      ...dnsConfig,
      targets: [
        ...dnsConfig.targets,
        {
          domain: '',
          server: '',
          type: 'A',
          timeout: 5,
          friendly_name: '',
          protocol: 'udp'
        }
      ]
    })
  }

  const removeTarget = (index: number) => {
    setDnsConfig({
      ...dnsConfig,
      targets: dnsConfig.targets.filter((_, i) => i !== index)
    })
  }

  const updateTargetString = (index: number, field: 'domain' | 'server' | 'friendly_name', value: string) => {
    setDnsConfig({
      ...dnsConfig,
      targets: dnsConfig.targets.map((target, i) => 
        i === index 
          ? { ...target, [field]: value }
          : target
      )
    })
  }

  const updateTargetType = (index: number, value: string) => {
    setDnsConfig({
      ...dnsConfig,
      targets: dnsConfig.targets.map((target, i) => 
        i === index 
          ? { ...target, type: value }
          : target
      )
    })
  }

  const updateTargetProtocol = (index: number, value: "udp" | "tcp" | "doh" | "dot") => {
    setDnsConfig({
      ...dnsConfig,
      targets: dnsConfig.targets.map((target, i) => 
        i === index 
          ? { ...target, protocol: value }
          : target
      )
    })
  }

  const updateTargetNumber = (index: number, field: 'timeout', value: number) => {
    setDnsConfig({
      ...dnsConfig,
      targets: dnsConfig.targets.map((target, i) => 
        i === index 
          ? { ...target, [field]: value }
          : target
      )
    })
  }

  return (
    <div className="space-y-6">
      {/* Check Interval at the top */}
      <div className="space-y-2">
        <Label htmlFor="dns-interval">DNS Interval</Label>
        <Input
          id="dns-interval"
          placeholder="*/5 * * * *"
          value={dnsConfig.interval}
          onChange={(e) => setDnsConfig({ ...dnsConfig, interval: e.target.value })}
        />
        <p className="text-sm text-muted-foreground">
          Cron expression (e.g., "*/5 * * * *" for every 5 minutes)
        </p>
      </div>



      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-medium">DNS Targets</h3>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={addTarget}
            className="flex items-center gap-2"
          >
            <PlusIcon className="h-4 w-4" />
            Add Target
          </Button>
        </div>

        {dnsConfig.targets.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-8 text-center">
              <ActivityIcon className="h-12 w-12 text-muted-foreground mb-4" />
              <p className="text-muted-foreground mb-2">No DNS targets configured</p>
              <p className="text-sm text-muted-foreground">
                Add DNS targets to monitor DNS resolution.
              </p>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-4">
            {dnsConfig.targets.map((target, index) => (
              <Card key={index}>
                <CardContent className="p-4">
                  <div className="flex items-center justify-between mb-4">
                  <h4 className="font-medium">{target.friendly_name || `Target ${index + 1}`}</h4>
                  <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => removeTarget(index)}
                      className="flex items-center gap-2"
                    >
                      <TrashIcon className="h-4 w-4" />
                      Remove
                    </Button>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label>Domain</Label>
                      <Input
                        placeholder="google.com"
                        value={target.domain}
                        onChange={(e) => updateTargetString(index, 'domain', e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>DNS Server</Label>
                      <Input
                        placeholder="8.8.8.8"
                        value={target.server}
                        onChange={(e) => updateTargetString(index, 'server', e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Record Type</Label>
                      <Select value={target.type} onValueChange={(value) => updateTargetType(index, value)}>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {DNS_TYPES.map((type) => (
                            <SelectItem key={type.value} value={type.value}>
                              {type.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label>Protocol</Label>
                      <Select value={target.protocol || 'udp'} onValueChange={(value: "udp" | "tcp" | "doh" | "dot") => updateTargetProtocol(index, value)}>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {DNS_PROTOCOLS.map((protocol) => (
                            <SelectItem key={protocol.value} value={protocol.value}>
                              {protocol.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label>Friendly Name (Optional)</Label>
                      <Input
                        placeholder="Google DNS"
                        value={target.friendly_name || ''}
                        onChange={(e) => updateTargetString(index, 'friendly_name', e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Timeout (seconds)</Label>
                      <Input
                        type="number"
                        min="1"
                        value={target.timeout}
                        onChange={(e) => updateTargetNumber(index, 'timeout', parseInt(e.target.value) || 1)}
                      />
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export function HttpConfigTab({ 
  httpConfig, 
  setHttpConfig 
}: { 
  httpConfig: { targets: HttpTarget[], interval: string, expected_response_time?: number }
  setHttpConfig: (config: { targets: HttpTarget[], interval: string, expected_response_time?: number }) => void
}): JSX.Element {

  const addTarget = () => {
    setHttpConfig({
      ...httpConfig,
      targets: [
        ...httpConfig.targets,
        {
          url: '',
          friendly_name: '',
          timeout: 10
        }
      ]
    })
  }

  const removeTarget = (index: number) => {
    setHttpConfig({
      ...httpConfig,
      targets: httpConfig.targets.filter((_, i) => i !== index)
    })
  }

  const updateTargetString = (index: number, field: 'url' | 'friendly_name', value: string) => {
    setHttpConfig({
      ...httpConfig,
      targets: httpConfig.targets.map((target, i) => 
        i === index 
          ? { ...target, [field]: value }
          : target
      )
    })
  }

  const updateTargetNumber = (index: number, field: 'timeout', value: number) => {
    setHttpConfig({
      ...httpConfig,
      targets: httpConfig.targets.map((target, i) => 
        i === index 
          ? { ...target, [field]: value }
          : target
      )
    })
  }

  return (
    <div className="space-y-6">
      {/* Check Interval at the top */}
      <div className="space-y-2">
        <Label htmlFor="http-interval">HTTP Interval</Label>
        <Input
          id="http-interval"
          placeholder="*/2 * * * *"
          value={httpConfig.interval}
          onChange={(e) => setHttpConfig({ ...httpConfig, interval: e.target.value })}
        />
        <p className="text-sm text-muted-foreground">
          Cron expression (e.g., "*/2 * * * *" for every 2 minutes)
        </p>
      </div>



      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-medium">HTTP Targets</h3>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={addTarget}
            className="flex items-center gap-2"
          >
            <PlusIcon className="h-4 w-4" />
            Add Target
          </Button>
        </div>

        {httpConfig.targets.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-8 text-center">
              <ActivityIcon className="h-12 w-12 text-muted-foreground mb-4" />
              <p className="text-muted-foreground mb-2">No HTTP targets configured</p>
              <p className="text-sm text-muted-foreground">
                Add HTTP targets to monitor web service availability.
              </p>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-4">
            {httpConfig.targets.map((target, index) => (
              <Card key={index}>
                <CardContent className="p-4">
                  <div className="flex items-center justify-between mb-4">
                  <h4 className="font-medium">{target.friendly_name || `Target ${index + 1}`}</h4>
                  <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => removeTarget(index)}
                      className="flex items-center gap-2"
                    >
                      <TrashIcon className="h-4 w-4" />
                      Remove
                    </Button>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label>URL</Label>
                      <Input
                        placeholder="https://google.com"
                        value={target.url}
                        onChange={(e) => updateTargetString(index, 'url', e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Friendly Name (Optional)</Label>
                      <Input
                        placeholder="Google"
                        value={target.friendly_name || ''}
                        onChange={(e) => updateTargetString(index, 'friendly_name', e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Timeout (seconds)</Label>
                      <Input
                        type="number"
                        min="1"
                        value={target.timeout}
                        onChange={(e) => updateTargetNumber(index, 'timeout', parseInt(e.target.value) || 1)}
                      />
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export function SpeedtestConfigTab({ 
  speedtestConfig, 
  setSpeedtestConfig 
}: { 
  speedtestConfig: { targets: SpeedtestTarget[], interval: string }
  setSpeedtestConfig: (config: { targets: SpeedtestTarget[], interval: string }) => void
}): JSX.Element {

  const addTarget = () => {
    setSpeedtestConfig({
      ...speedtestConfig,
      targets: [
        ...speedtestConfig.targets,
        {
          server_id: '',
          friendly_name: '',
          timeout: 60
        }
      ]
    })
  }

  const removeTarget = (index: number) => {
    setSpeedtestConfig({
      ...speedtestConfig,
      targets: speedtestConfig.targets.filter((_, i) => i !== index)
    })
  }

  const updateTargetString = (index: number, field: 'server_id' | 'friendly_name', value: string) => {
    setSpeedtestConfig({
      ...speedtestConfig,
      targets: speedtestConfig.targets.map((target, i) => 
        i === index 
          ? { ...target, [field]: value }
          : target
      )
    })
  }

  const updateTargetNumber = (index: number, field: 'timeout', value: number) => {
    setSpeedtestConfig({
      ...speedtestConfig,
      targets: speedtestConfig.targets.map((target, i) => 
        i === index 
          ? { ...target, [field]: value }
          : target
      )
    })
  }

  return (
    <div className="space-y-6">
      {/* Check Interval at the top */}
      <div className="space-y-2">
        <Label htmlFor="speedtest-interval">Speedtest Interval</Label>
        <Input
          id="speedtest-interval"
          placeholder="*/10 * * * *"
          value={speedtestConfig.interval}
          onChange={(e) => setSpeedtestConfig({ ...speedtestConfig, interval: e.target.value })}
        />
        <p className="text-sm text-muted-foreground">
          Cron expression (e.g., "*/10 * * * *" for every 10 minutes)
        </p>
      </div>



      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-medium">Speedtest Targets</h3>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={addTarget}
            className="flex items-center gap-2"
          >
            <PlusIcon className="h-4 w-4" />
            Add Target
          </Button>
        </div>

        {speedtestConfig.targets.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-8 text-center">
              <ActivityIcon className="h-12 w-12 text-muted-foreground mb-4" />
              <p className="text-muted-foreground mb-2">No speedtest targets configured</p>
              <p className="text-sm text-muted-foreground">
                Add speedtest targets to monitor network performance.
              </p>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-4">
            {speedtestConfig.targets.map((target, index) => (
              <Card key={index}>
                <CardContent className="p-4">
                  <div className="flex items-center justify-between mb-4">
                  <h4 className="font-medium">{target.friendly_name || `Target ${index + 1}`}</h4>
                  <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => removeTarget(index)}
                      className="flex items-center gap-2"
                    >
                      <TrashIcon className="h-4 w-4" />
                      Remove
                    </Button>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label>Server ID</Label>
                      <Input
                        placeholder="12345"
                        value={target.server_id}
                        onChange={(e) => updateTargetString(index, 'server_id', e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Friendly Name (Optional)</Label>
                      <Input
                        placeholder="Local Server"
                        value={target.friendly_name || ''}
                        onChange={(e) => updateTargetString(index, 'friendly_name', e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Timeout (seconds)</Label>
                      <Input
                        type="number"
                        min="1"
                        value={target.timeout}
                        onChange={(e) => updateTargetNumber(index, 'timeout', parseInt(e.target.value) || 1)}
                      />
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
