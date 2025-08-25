import { t } from "@lingui/core/macro"
import { memo, useMemo, useState, useEffect } from "react"
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
	DialogTrigger,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { ActivityIcon } from "lucide-react"
import { SystemRecord } from "@/types"
import { Separator } from "@/components/ui/separator"
import { toast } from "sonner"
import { pb } from "@/lib/stores"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { 
  PingConfigTab, 
  DnsConfigTab, 
  HttpConfigTab, 
  SpeedtestConfigTab,
  ExpectedPerformanceTab,
  PingTarget,
  DnsTarget,
  HttpTarget,
  SpeedtestTarget
} from "@/components/system-config/monitoring-config-tabs"

// Using shared interfaces from monitoring-config-tabs

interface MonitoringConfig {
	enabled: {
		ping: boolean
		dns: boolean
		http?: boolean
		speedtest?: boolean
	}
	global_interval?: string
	ping?: {
		targets: PingTarget[]
		interval?: string
	}
	dns?: {
		targets: DnsTarget[]
		interval?: string
	}
	http?: {
		targets: HttpTarget[]
		interval?: string
	}
	speedtest?: {
		targets: SpeedtestTarget[]
		interval?: string
	}
}

// Using shared components from monitoring-config-tabs

export const SystemConfigDialog = memo(function SystemConfigDialog({ system }: { system: SystemRecord }) {
	// Unified state management
	const [pingConfig, setPingConfig] = useState<{ targets: PingTarget[], interval: string, expected_latency?: number }>({ targets: [], interval: "*/3 * * * *" })
	const [dnsConfig, setDnsConfig] = useState<{ targets: DnsTarget[], interval: string, expected_lookup_time?: number }>({ targets: [], interval: "*/5 * * * *" })
	const [httpConfig, setHttpConfig] = useState<{ targets: HttpTarget[], interval: string, expected_response_time?: number }>({ targets: [], interval: "*/2 * * * *" })
	const [speedtestConfig, setSpeedtestConfig] = useState<{ targets: SpeedtestTarget[], interval: string }>({ targets: [], interval: "0 */6 * * *" })
	const [isLoading, setIsLoading] = useState(false)
	const [isConfigLoading, setIsConfigLoading] = useState(true)
	const [monitoringConfigId, setMonitoringConfigId] = useState<string | null>(null)

	// Load existing configs from the monitoring_config collection
	useEffect(() => {
		const loadMonitoringConfig = async () => {
			setIsConfigLoading(true)
			try {
				// Add a small delay to ensure the system is properly loaded
				await new Promise(resolve => setTimeout(resolve, 100))
				
				const existingConfig = await pb.collection("monitoring_config").getFirstListItem(`system = "${system.id}"`)
				console.log("Monitoring config from collection:", existingConfig)
				
				if (existingConfig) {
					setMonitoringConfigId(existingConfig.id)
					
					// Parse ping configuration
					if (existingConfig.ping) {
						const pingData = typeof existingConfig.ping === 'string' ? JSON.parse(existingConfig.ping) : existingConfig.ping
						setPingConfig({
							targets: pingData.targets || [],
							interval: pingData.interval || "*/3 * * * *",
							expected_latency: system.expected_performance?.ping_latency
						})
					}
					
					// Parse DNS configuration
					if (existingConfig.dns) {
						const dnsData = typeof existingConfig.dns === 'string' ? JSON.parse(existingConfig.dns) : existingConfig.dns
						setDnsConfig({
							targets: dnsData.targets || [],
							interval: dnsData.interval || "*/5 * * * *",
							expected_lookup_time: system.expected_performance?.dns_lookup_time
						})
					}
					
					// Parse HTTP configuration
					if (existingConfig.http) {
						const httpData = typeof existingConfig.http === 'string' ? JSON.parse(existingConfig.http) : existingConfig.http
						setHttpConfig({
							targets: httpData.targets || [],
							interval: httpData.interval || "*/2 * * * *",
							expected_response_time: system.expected_performance?.http_response_time
						})
					}
					
					// Parse speedtest configuration
					if (existingConfig.speedtest) {
						const speedtestData = typeof existingConfig.speedtest === 'string' ? JSON.parse(existingConfig.speedtest) : existingConfig.speedtest
						setSpeedtestConfig({
							targets: speedtestData.targets || [],
							interval: speedtestData.interval || "0 */6 * * *"
						})
					}
				}
			} catch (error) {
				// No existing config found, use defaults but still load expected performance from system
				setPingConfig({
					targets: [],
					interval: "*/3 * * * *",
					expected_latency: system.expected_performance?.ping_latency
				})
				setDnsConfig({
					targets: [],
					interval: "*/5 * * * *",
					expected_lookup_time: system.expected_performance?.dns_lookup_time
				})
				setHttpConfig({
					targets: [],
					interval: "*/2 * * * *",
					expected_response_time: system.expected_performance?.http_response_time
				})
				setSpeedtestConfig({
					targets: [],
					interval: "0 */6 * * *"
				})
			} finally {
				setIsConfigLoading(false)
			}
		}
		
		loadMonitoringConfig()
	}, [system.id])

	const validateCronExpression = (expression: string): string | null => {
		const cronParts = expression.split(' ')
		if (cronParts.length !== 5) {
			return t`Invalid cron expression. Must have 5 parts: minute hour day month weekday`
		}
		return null
	}

	const saveAllConfigs = async () => {
		if (isConfigLoading) {
			console.log("🔍 Debug Config Dialog - Cannot save while config is still loading")
			return
		}
		
		setIsLoading(true)
		try {
			// Validate all cron expressions
			const pingError = validateCronExpression(pingConfig.interval)
			const dnsError = validateCronExpression(dnsConfig.interval)
			const httpError = validateCronExpression(httpConfig.interval)
			const speedtestError = validateCronExpression(speedtestConfig.interval)

			if (pingError || dnsError || httpError || speedtestError) {
				toast.error(pingError || dnsError || httpError || speedtestError)
				return
			}

			// Validate targets
			const validPingTargets = pingConfig.targets.filter(target => 
				target.host.trim() !== '' && 
				target.count > 0 && 
				target.timeout > 0
			)

			const validDnsTargets = dnsConfig.targets.filter(target => 
				target.domain.trim() !== '' && 
				target.server.trim() !== '' && 
				target.type.trim() !== '' && 
				target.timeout > 0
			)

			const validHttpTargets = httpConfig.targets.filter(target => 
				target.url.trim() !== ''
			)

			const validSpeedtestTargets = speedtestConfig.targets.filter(target => 
				// Allow empty server_id for auto-selection, but require at least a friendly_name or server_id
				target.server_id.trim() !== '' || target.friendly_name?.trim() !== ''
			)

			// Prepare the monitoring config data
			const monitoringConfigData = {
				system: system.id,
				ping: {
					enabled: validPingTargets.length > 0,
					targets: validPingTargets,
					interval: pingConfig.interval
				},
				dns: {
					enabled: validDnsTargets.length > 0,
					targets: validDnsTargets,
					interval: dnsConfig.interval
				},
				http: {
					enabled: validHttpTargets.length > 0,
					targets: validHttpTargets,
					interval: httpConfig.interval
				},
				speedtest: {
					enabled: validSpeedtestTargets.length > 0,
					targets: validSpeedtestTargets,
					interval: speedtestConfig.interval
				}
			}

			// Save monitoring config to separate collection
			if (monitoringConfigId) {
				// Update existing record
				await pb.collection("monitoring_config").update(monitoringConfigId, monitoringConfigData)
			} else {
				// Create new record
				const newRecord = await pb.collection("monitoring_config").create(monitoringConfigData)
				setMonitoringConfigId(newRecord.id)
			}
			
			// Save expected performance values to system record
			const expectedPerformanceData = {
				expected_performance: {
					ping_latency: pingConfig.expected_latency,
					dns_lookup_time: dnsConfig.expected_lookup_time,
					http_response_time: httpConfig.expected_response_time,
					download_speed: speedtestConfig.expected_download_speed,
					upload_speed: speedtestConfig.expected_upload_speed
				}
			}
			
			console.log("Saving expected performance data:", expectedPerformanceData)
			
			const updatedSystem = await pb.collection("systems").update(system.id, expectedPerformanceData)
			console.log("Updated system record:", updatedSystem)

			toast.success(t`Configuration saved and pushed to agent successfully. Changes will take effect immediately.`)
		} catch (error) {
			console.error("Failed to save configs:", error)
			toast.error(t`Failed to save configuration. Please try again.`)
		} finally {
			setIsLoading(false)
		}
	}

	const hasAnyConfig = pingConfig.targets.length > 0 || dnsConfig.targets.length > 0 || httpConfig.targets.length > 0 || speedtestConfig.targets.length > 0

	return (
		<Dialog>
			<DialogTrigger asChild>
				<Button variant="ghost" size="icon" aria-label={t`Configure Monitoring`} data-nolink>
					<ActivityIcon className="h-4 w-4" />
				</Button>
			</DialogTrigger>
			<DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
				<DialogHeader>
					<DialogTitle>{t`System Monitoring Configuration`}</DialogTitle>
					<DialogDescription>
						{t`Configure monitoring targets and intervals for ${system.name}. Changes will be pushed to the agent automatically and take effect immediately.`}
					</DialogDescription>
				</DialogHeader>
				<Tabs defaultValue="speedtest" className="w-full">
					<TabsList className="grid w-full grid-cols-5">
						<TabsTrigger value="speedtest">{t`Speedtest`}</TabsTrigger>
						<TabsTrigger value="ping">{t`Ping`}</TabsTrigger>
						<TabsTrigger value="dns">{t`DNS`}</TabsTrigger>
						<TabsTrigger value="http">{t`HTTP`}</TabsTrigger>
						<TabsTrigger value="performance">{t`Thresholds`}</TabsTrigger>
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
					<TabsContent value="performance" className="mt-6">
						<ExpectedPerformanceTab 
							pingConfig={pingConfig}
							setPingConfig={setPingConfig}
							dnsConfig={dnsConfig}
							setDnsConfig={setDnsConfig}
							httpConfig={httpConfig}
							setHttpConfig={setHttpConfig}
							speedtestConfig={speedtestConfig}
							setSpeedtestConfig={setSpeedtestConfig}
						/>
					</TabsContent>
				</Tabs>

				<div className="flex justify-end">
					<Button
						type="button"
						onClick={saveAllConfigs}
						disabled={isLoading || isConfigLoading}
					>
						{isConfigLoading ? t`Loading...` : isLoading ? t`Saving...` : t`Save Configuration`}
					</Button>
				</div>
			</DialogContent>
		</Dialog>
	)
})
