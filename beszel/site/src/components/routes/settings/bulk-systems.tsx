"use client"

import { useState, useEffect } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Checkbox } from "@/components/ui/checkbox"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Label } from "@/components/ui/label"
import { toast } from "sonner"
import { Loader2, Save, RefreshCw, AlertTriangle, CheckCircle, XCircle } from "lucide-react"
import { pb } from "@/lib/stores"
import { 
  PingConfigTab, 
  DnsConfigTab, 
  HttpConfigTab, 
  SpeedtestConfigTab,
  PingTarget,
  DnsTarget,
  HttpTarget,
  SpeedtestTarget
} from "@/components/system-config/monitoring-config-tabs"

interface System {
  id: string
  name: string
  host: string
  status: string
  monitoring_config?: string
}

interface MonitoringConfig {
  id: string
  system: string
  enabled: {
    ping: boolean
    dns: boolean
    http: boolean
    speedtest: boolean
  }
  global_interval: string
  ping: {
    targets: PingTarget[]
    interval: string
  }
  dns: {
    targets: DnsTarget[]
    interval: string
  }
  http: {
    targets: HttpTarget[]
    interval: string
  }
  speedtest: {
    targets: SpeedtestTarget[]
    interval: string
  }
}

export default function BulkSystemsSettings() {
  const [systems, setSystems] = useState<System[]>([])
  const [selectedSystems, setSelectedSystems] = useState<Set<string>>(new Set())
  const [configs, setConfigs] = useState<Record<string, MonitoringConfig>>({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [selectAll, setSelectAll] = useState(false)

  // Config state (matching dialog structure)
  const [pingConfig, setPingConfig] = useState<{ targets: PingTarget[], interval: string }>({
    targets: [],
    interval: ''
  })
  const [dnsConfig, setDnsConfig] = useState<{ targets: DnsTarget[], interval: string }>({
    targets: [],
    interval: ''
  })
  const [httpConfig, setHttpConfig] = useState<{ targets: HttpTarget[], interval: string }>({
    targets: [],
    interval: ''
  })
  const [speedtestConfig, setSpeedtestConfig] = useState<{ targets: SpeedtestTarget[], interval: string }>({
    targets: [],
    interval: ''
  })

  // Load systems and their configurations
  useEffect(() => {
    loadSystems()
  }, [])

  const loadSystems = async () => {
    try {
      setLoading(true)
      
      // Load all systems
      const systemsResponse = await pb.collection('systems').getList(1, 1000)
      const systemsData = systemsResponse.items as unknown as System[]
      setSystems(systemsData)

      // Load configurations for each system
      const configsData: Record<string, MonitoringConfig> = {}
      for (const system of systemsData) {
        try {
          const config = await pb.collection('monitoring_config').getFirstListItem(`system = "${system.id}"`)
          console.log(`Found config for ${system.name}:`, config)
          configsData[system.id] = config as unknown as MonitoringConfig
        } catch (error) {
          console.log(`No config found for system ${system.name} - will create one when needed`)
        }
      }
      setConfigs(configsData)
    } catch (error) {
      console.error('Error loading systems:', error)
      toast.error('Failed to load systems')
    } finally {
      setLoading(false)
    }
  }

  const handleSelectAll = (checked: boolean) => {
    setSelectAll(checked)
    if (checked) {
      setSelectedSystems(new Set(systems.map(s => s.id)))
    } else {
      setSelectedSystems(new Set())
    }
  }

  const handleSelectSystem = (systemId: string, checked: boolean) => {
    const newSelected = new Set(selectedSystems)
    if (checked) {
      newSelected.add(systemId)
    } else {
      newSelected.delete(systemId)
    }
    setSelectedSystems(newSelected)
    setSelectAll(newSelected.size === systems.length)
  }

  const loadConfigFromSelected = () => {
    if (selectedSystems.size === 0) {
      toast.error('Please select at least one system')
      return
    }

    const selectedConfigs = Array.from(selectedSystems)
      .map(id => configs[id])
      .filter(config => config)

    if (selectedConfigs.length === 0) {
      toast.error('No configurations found for selected systems')
      return
    }

    // Use the first config as a template
    const templateConfig = selectedConfigs[0]
    setPingConfig({
      targets: templateConfig.ping?.targets || [],
      interval: templateConfig.ping?.interval || ''
    })
    setDnsConfig({
      targets: templateConfig.dns?.targets || [],
      interval: templateConfig.dns?.interval || ''
    })
    setHttpConfig({
      targets: templateConfig.http?.targets || [],
      interval: templateConfig.http?.interval || ''
    })
    setSpeedtestConfig({
      targets: templateConfig.speedtest?.targets || [],
      interval: templateConfig.speedtest?.interval || ''
    })

    toast.success(`Loaded configuration from ${selectedConfigs.length} system(s)`)
  }

  const saveBulkConfig = async () => {
    if (selectedSystems.size === 0) {
      toast.error('Please select at least one system')
      return
    }

    try {
      setSaving(true)
      
      for (const systemId of selectedSystems) {
        const system = systems.find(s => s.id === systemId)
        if (!system) continue

        // Prepare the monitoring config data (matching dialog structure)
        const monitoringConfigData = {
          system: systemId,
          ping: {
            enabled: pingConfig.targets.length > 0,
            targets: pingConfig.targets,
            interval: pingConfig.interval
          },
          dns: {
            enabled: dnsConfig.targets.length > 0,
            targets: dnsConfig.targets,
            interval: dnsConfig.interval
          },
          http: {
            enabled: httpConfig.targets.length > 0,
            targets: httpConfig.targets,
            interval: httpConfig.interval
          },
          speedtest: {
            enabled: speedtestConfig.targets.length > 0,
            targets: speedtestConfig.targets,
            interval: speedtestConfig.interval
          }
        }

        // Check if config exists
        const existingConfig = configs[systemId]
        if (existingConfig) {
          // Update existing config
          await pb.collection('monitoring_config').update(existingConfig.id, monitoringConfigData)
        } else {
          // Create new config
          await pb.collection('monitoring_config').create(monitoringConfigData)
        }
      }

      toast.success(`Configuration updated for ${selectedSystems.size} system(s)`)
      await loadSystems() // Reload to get updated data
    } catch (error) {
      console.error('Error saving bulk config:', error)
      toast.error('Failed to save configuration')
    } finally {
      setSaving(false)
    }
  }

  const getSystemStatusIcon = (status: string) => {
    switch (status) {
      case 'up':
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case 'down':
        return <XCircle className="h-4 w-4 text-red-500" />
      default:
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Bulk System Settings</h2>
        <p className="text-muted-foreground">
          Configure monitoring settings for multiple systems at once.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Systems Selection */}
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle>Systems</CardTitle>
            <CardDescription>
              Select systems to configure
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center space-x-2">
              <Checkbox
                id="select-all"
                checked={selectAll}
                onCheckedChange={handleSelectAll}
              />
              <Label htmlFor="select-all">Select All ({systems.length})</Label>
            </div>
            
            <Separator />
            
            <div className="space-y-2 max-h-96 overflow-y-auto">
              {systems.map((system) => (
                <div key={system.id} className="flex items-center space-x-2">
                  <Checkbox
                    id={system.id}
                    checked={selectedSystems.has(system.id)}
                    onCheckedChange={(checked) => handleSelectSystem(system.id, checked as boolean)}
                  />
                  <div className="flex-1 min-w-0">
                    <Label htmlFor={system.id} className="text-sm font-medium truncate">
                      {system.name}
                    </Label>
                    <div className="flex items-center space-x-2 text-xs text-muted-foreground">
                      {getSystemStatusIcon(system.status)}
                      <span>{system.host}</span>
                      <Badge variant={system.status === 'up' ? 'default' : 'secondary'}>
                        {system.status}
                      </Badge>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {selectedSystems.size > 0 && (
              <Alert>
                <AlertDescription>
                  {selectedSystems.size} system(s) selected
                </AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>

        {/* Configuration */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Configuration</CardTitle>
            <CardDescription>
              Configure monitoring settings for selected systems
            </CardDescription>
          </CardHeader>
          <CardContent>
            {selectedSystems.size === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                Select systems to configure their settings
              </div>
            ) : (
              <div className="space-y-6">
                {/* Action Buttons */}
                <div className="flex justify-between items-center">
                  <div className="flex space-x-2">
                    <Button
                      variant="outline"
                      onClick={loadConfigFromSelected}
                      disabled={selectedSystems.size === 0}
                    >
                      Load from Selected
                    </Button>
                    <Button
                      variant="outline"
                      onClick={loadSystems}
                      disabled={saving}
                    >
                      <RefreshCw className="h-4 w-4 mr-2" />
                      Refresh
                    </Button>
                  </div>
                  <Button
                    onClick={saveBulkConfig}
                    disabled={saving || selectedSystems.size === 0}
                  >
                    {saving ? (
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    ) : (
                      <Save className="h-4 w-4 mr-2" />
                    )}
                    Save to {selectedSystems.size} System(s)
                  </Button>
                </div>

                <Separator />

                {/* Monitoring Types - using shared components */}
                <Tabs defaultValue="speedtest" className="w-full">
                  <TabsList className="grid w-full grid-cols-4">
                    <TabsTrigger value="speedtest">Speedtest</TabsTrigger>
                    <TabsTrigger value="ping">Ping</TabsTrigger>
                    <TabsTrigger value="dns">DNS</TabsTrigger>
                    <TabsTrigger value="http">HTTP</TabsTrigger>
                  </TabsList>
                  
                  <TabsContent value="speedtest" className="mt-6">
                    <SpeedtestConfigTab speedtestConfig={speedtestConfig} setSpeedtestConfig={setSpeedtestConfig} />
                  </TabsContent>
                  <TabsContent value="ping" className="mt-6">
                    <PingConfigTab pingConfig={pingConfig} setPingConfig={setPingConfig} />
                  </TabsContent>
                  <TabsContent value="dns" className="mt-6">
                    <DnsConfigTab dnsConfig={dnsConfig} setDnsConfig={setDnsConfig} />
                  </TabsContent>
                  <TabsContent value="http" className="mt-6">
                    <HttpConfigTab httpConfig={httpConfig} setHttpConfig={setHttpConfig} />
                  </TabsContent>
                </Tabs>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
