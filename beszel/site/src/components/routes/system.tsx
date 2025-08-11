import { t } from "@lingui/core/macro"
import { Plural } from "@lingui/react/macro"
import {
	$systems,
	pb,
	$chartTime,
} from "@/lib/stores"
import { SystemRecord, PingStatsRecord, DnsStatsRecord, ChartData, ChartTimes } from "@/types"
import React, { lazy, useEffect, useMemo, useState } from "react"
import { Card, CardHeader, CardTitle, CardDescription } from "../ui/card"
import { useStore } from "@nanostores/react"
import { GlobeIcon, MonitorIcon, EthernetPortIcon, LayoutGridIcon, Building2Icon, RouteIcon } from "lucide-react"
import { Rows } from "../ui/icons"
import {
	cn,
	getHostDisplayValue,
	listen,
	getPbTimestamp,
	chartTimeData,
	parseCronInterval,
	useLocalStorage,
} from "@/lib/utils"
import { Separator } from "../ui/separator"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "../ui/tooltip"
import { useLingui } from "@lingui/react/macro"
import { $router, navigate } from "../router"
import { getPagePath } from "@nanostores/router"
import Spinner from "../spinner"
import { useIntersectionObserver } from "@/lib/use-intersection-observer"
import ChartTimeSelect from "../charts/chart-time-select"
import { Button } from "../ui/button"
import { SystemConfigDialog } from "../system-config/system-config-dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../ui/tabs"

const PingChart = lazy(() => import("../charts/ping-chart"))
const DnsChart = lazy(() => import("../charts/dns-chart"))

// Helper function to get default tick count for chart time periods
function getDefaultTickCount(chartTime: ChartTimes): number {
	switch (chartTime) {
		case "1h":
			return 6 // Every 10 minutes
		case "24h":
			return 8 // Every 3 hours
		default:
			return 6 // Default fallback
	}
}

export default function SystemDetail({ name }: { name: string }) {
	const { t } = useLingui()
	const systems = useStore($systems)
	const chartTime = useStore($chartTime)
	const [grid, setGrid] = useLocalStorage("grid", true)
	const [system, setSystem] = useState({} as SystemRecord)
	const [pingStats, setPingStats] = useState([] as PingStatsRecord[])
	const [dnsStats, setDnsStats] = useState([] as DnsStatsRecord[])
	const [chartLoading, setChartLoading] = useState(true)

	useEffect(() => {
		document.title = `${name} / Beszel`
	}, [name])

	// find matching system
	useEffect(() => {
		if (system.id && system.name === name) {
			return
		}
		const matchingSystem = systems.find((s) => s.name === name) as SystemRecord
		if (matchingSystem) {
			setSystem(matchingSystem)
		}
	}, [name, system, systems])

	// update system when new data is available
	useEffect(() => {
		if (!system.id) {
			return
		}
		pb.collection<SystemRecord>("systems").subscribe(system.id, (e) => {
			setSystem(e.record)
		})
		return () => {
			pb.collection("systems").unsubscribe(system.id)
		}
	}, [system.id])

	// Fetch ping stats
	useEffect(() => {
		if (!system.id || !chartTime) {
			return
		}
		setChartLoading(true)
		
		pb.collection<PingStatsRecord>("ping_stats").getFullList({
			filter: pb.filter("system={:id} && created > {:created}", {
				id: system.id,
				created: getPbTimestamp(chartTime, undefined),
			}),
			fields: "host,packet_loss,min_rtt,max_rtt,avg_rtt,created",
			sort: "created",
		}).then((records) => {
			setPingStats(records)
			setChartLoading(false)
		}).catch((error) => {
			console.error("Error fetching ping stats:", error)
			setChartLoading(false)
		})
	}, [system.id, chartTime])

	// Fetch DNS stats
	useEffect(() => {
		if (!system.id || !chartTime) {
			return
		}
		
		pb.collection<DnsStatsRecord>("dns_stats").getFullList({
			filter: pb.filter("system={:id} && created > {:created}", {
				id: system.id,
				created: getPbTimestamp(chartTime, undefined),
			}),
			fields: "domain,server,type,status,lookup_time,error_code,created",
			sort: "created",
		}).then((records) => {
			setDnsStats(records)
		}).catch((error) => {
			console.error("Failed to fetch DNS stats:", error)
		})
	}, [system.id, chartTime])

	// Create chart data
	const chartData: ChartData = useMemo(() => {
		// Get all unique hosts
		const allHosts = new Set<string>()
		pingStats.forEach(record => allHosts.add(record.host))
		
		// Sort records by timestamp
		const sortedRecords = [...pingStats].sort((a, b) => 
			new Date(a.created).getTime() - new Date(b.created).getTime()
		)
		
		// Group records by timestamp and add gaps for missing intervals
		const pingData: any[] = []
		let prevTimestamp = 0
		
		// Get the user-defined ping interval from the unified monitoring configuration
		let expectedInterval = 3 * 60 * 1000 // Default fallback: 3 minutes
		if (system.monitoring_config?.ping?.interval) {
			const userInterval = parseCronInterval(String(system.monitoring_config.ping.interval))
			if (userInterval) {
				expectedInterval = userInterval
			}
		} else if (system.monitoring_config?.global_interval) {
			const userInterval = parseCronInterval(String(system.monitoring_config.global_interval))
			if (userInterval) {
				expectedInterval = userInterval
			}
		}
		
		sortedRecords.forEach(record => {
			const timestamp = new Date(record.created).getTime()
			
			// Add gap if interval is too large (more than 2x the expected interval)
			if (prevTimestamp && (timestamp - prevTimestamp) > expectedInterval * 2) {
				// Add null record to create gap
				const gapDataPoint: any = { created: null }
				allHosts.forEach(host => {
					gapDataPoint[host] = null
				})
				pingData.push(gapDataPoint)
			}
			
			// Find or create data point for this timestamp
			let dataPoint = pingData.find(dp => dp.created === timestamp)
			if (!dataPoint) {
				dataPoint = { created: timestamp }
				allHosts.forEach(host => {
					dataPoint[host] = null
				})
				pingData.push(dataPoint)
			}
			
			// Add the actual data for this host
			dataPoint[record.host] = {
				host: record.host,
				packet_loss: record.packet_loss,
				min_rtt: record.min_rtt,
				max_rtt: record.max_rtt,
				avg_rtt: record.avg_rtt,
			}
			
			prevTimestamp = timestamp
		})

		// Process DNS data
		const dnsData: any[] = []
		if (dnsStats.length > 0) {
			// Get all unique DNS targets
			const allDnsTargets = new Set<string>()
			dnsStats.forEach(record => {
				// Handle empty type field - if type is empty, don't include it in the key
				const typePart = record.type && record.type.trim() ? record.type : ''
				const key = `${record.domain}@${record.server}#${typePart}`
				allDnsTargets.add(key)
			})
			
			// Sort DNS records by timestamp
			const sortedDnsRecords = [...dnsStats].sort((a, b) => 
				new Date(a.created).getTime() - new Date(b.created).getTime()
			)
			
			// Group DNS records by timestamp
			let prevDnsTimestamp = 0
			
			// Get the user-defined DNS interval from the unified monitoring configuration
			let dnsExpectedInterval = 5 * 60 * 1000 // Default fallback: 5 minutes
			if (system.monitoring_config?.dns?.interval) {
				const userInterval = parseCronInterval(String(system.monitoring_config.dns.interval))
				if (userInterval) {
					dnsExpectedInterval = userInterval
				}
			} else if (system.monitoring_config?.global_interval) {
				const userInterval = parseCronInterval(String(system.monitoring_config.global_interval))
				if (userInterval) {
					dnsExpectedInterval = userInterval
				}
			}
			

			
			sortedDnsRecords.forEach(record => {
				const timestamp = new Date(record.created).getTime()
				// Handle empty type field - if type is empty, don't include it in the key
				const typePart = record.type && record.type.trim() ? record.type : ''
				const key = `${record.domain}@${record.server}#${typePart}`
				
				// Add gap if interval is too large
				const timeDiff = timestamp - prevDnsTimestamp
				if (prevDnsTimestamp && timeDiff > dnsExpectedInterval * 2) {
					const gapDataPoint: any = { created: null }
					allDnsTargets.forEach(targetKey => {
						gapDataPoint[targetKey] = null
					})
					dnsData.push(gapDataPoint)
				}
				
				// Find or create data point for this timestamp
				let dataPoint = dnsData.find(dp => dp.created === timestamp)
				if (!dataPoint) {
					dataPoint = { created: timestamp }
					allDnsTargets.forEach(targetKey => {
						dataPoint[targetKey] = null
					})
					dnsData.push(dataPoint)
				}
				
				// Add the actual DNS data for this target
				dataPoint[key] = {
					domain: record.domain,
					server: record.server,
					type: record.type,
					status: record.status,
					lookup_time: record.lookup_time,
					error_code: record.error_code,
				}
				
				prevDnsTimestamp = timestamp
			})
		}

		// Calculate time domain and ticks
		const now = new Date()
		const startTime = chartTimeData[chartTime].getOffset(now)
		const domain = [startTime.getTime(), now.getTime()]
		
		// Create ticks based on chart time configuration
		let ticks: number[]
		const tickCount = chartTimeData[chartTime].ticks || getDefaultTickCount(chartTime)
		const interval = (now.getTime() - startTime.getTime()) / (tickCount - 1)
		ticks = Array.from({ length: tickCount }, (_, i) => startTime.getTime() + i * interval)

		const result: ChartData = {
			pingData,
			dnsData,
			systemStats: [],
			containerData: [],
			orientation: "left" as const,
			ticks,
			domain,
			chartTime,
			agentVersion: { major: 0, minor: 0, patch: 0 },
		}
		
		// Debug: Check if gap data points are in the final DNS data
		const gapPoints = dnsData.filter(dp => dp.created === null)
		if (gapPoints.length > 0) {
			console.log('🔍 Debug DNS Gaps - Final data contains', gapPoints.length, 'gap points')
		}
		
		return result
	}, [pingStats, dnsStats, chartTime])

	// Get unique hosts with friendly names from ping stats and config
	const pingHosts = useMemo(() => {
		const hosts = new Set<string>()
		pingStats.forEach(record => hosts.add(record.host))
		
		// Create a map of host to friendly name from ping config
		const hostToFriendlyName = new Map<string, string>()
		if (system.monitoring_config?.ping?.targets) {
			system.monitoring_config.ping.targets.forEach((target: any) => {
				if (target.friendly_name && target.friendly_name.trim()) {
					hostToFriendlyName.set(target.host, target.friendly_name.trim())
				}
			})
		}
		
		return Array.from(hosts).map(host => ({
			host,
			friendlyName: hostToFriendlyName.get(host) || host
		}))
	}, [pingStats, system.monitoring_config])

	// Get unique DNS targets from DNS stats with friendly names
	const dnsTargets = useMemo(() => {
		// Start with DNS config targets to ensure we have friendly names
		const configTargets = new Map<string, string>()
		if (system.monitoring_config?.dns?.targets) {
			system.monitoring_config.dns.targets.forEach(target => {
				// Handle empty type field - if type is empty, don't include it in the key
				const typePart = target.type && target.type.trim() ? target.type : ''
				const protocol = target.protocol || 'udp'
				const key = `${target.domain}@${target.server}#${typePart}`
				const friendlyName = target.friendly_name && target.friendly_name.trim() ? 
					target.friendly_name.trim() : 
					`${target.domain} @ ${target.server} (${typePart})`
				configTargets.set(key, friendlyName)
			})
		}
		
		// Add any additional targets from DNS stats that aren't in config
		const targets = new Set<string>()
		dnsStats.forEach(record => {
			// Handle empty type field - if type is empty, don't include it in the key
			const typePart = record.type && record.type.trim() ? record.type : ''
			const key = `${record.domain}@${record.server}#${typePart}`
			targets.add(key)
		})
		
		return Array.from(targets).map(key => {
			// Try exact match first
			let friendlyName = configTargets.get(key)
			
			// If no exact match, try matching without type (for cases where stats have empty type)
			if (!friendlyName) {
				const [domainPart, rest] = key.split('@')
				const [server, type] = rest.split('#')
				
				// Try matching with different type variations
				const variations = [
					`${domainPart}@${server}#A`,  // Try with A type
					`${domainPart}@${server}#`,   // Try with empty type
					`${domainPart}@${server}#AAAA`, // Try with AAAA type
					`${domainPart}@${server}#CNAME`, // Try with CNAME type
				]
				
				for (const variation of variations) {
					friendlyName = configTargets.get(variation)
					if (friendlyName) {
						break
					}
				}
			}
			
			if (friendlyName) {
				return {
					key,
					friendlyName: friendlyName
				}
			} else {
				// Parse the key to extract domain, server, and type for fallback
				const [domainPart, rest] = key.split('@')
				const [server, type] = rest.split('#')
				const fallbackName = type && type.trim() ? 
					`${domainPart} @ ${server} (${type})` : 
					`${domainPart} @ ${server}`
				return {
					key,
					friendlyName: fallbackName
				}
			}
		})
	}, [dnsStats, system.monitoring_config])

	// values for system info bar
	const systemInfo = useMemo(() => {
		if (!system.info) {
			return []
		}

		return [
			{ value: getHostDisplayValue(system), Icon: GlobeIcon },
			{
				value: system.info.h,
				Icon: MonitorIcon,
				label: "Hostname",
				// hide if hostname is same as host or name
				hide: system.info.h === system.host || system.info.h === system.name,
			},
			{
				value: system.info.ns ? `${system.info.ns} Mbps` : undefined,
				Icon: EthernetPortIcon,
				label: t`Network Interface Speed`,
				hide: !system.info.ns,
			},
			{
				value: system.info.ip,
				Icon: GlobeIcon,
				label: t`Public IP`,
				hide: !system.info.ip,
			},
			{
				value: system.info.isp,
				Icon: Building2Icon,
				label: t`ISP`,
				hide: !system.info.isp,
			},
			{
				value: system.info.asn,
				Icon: RouteIcon,
				label: t`ASN`,
				hide: !system.info.asn,
			},
		] as {
			value: string | number | undefined
			label?: string
			Icon: any
			hide?: boolean
		}[]
	}, [system.info])

	// keyboard navigation between systems
	useEffect(() => {
		if (!systems.length) {
			return
		}
		const handleKeyUp = (e: KeyboardEvent) => {
			if (
				e.target instanceof HTMLInputElement ||
				e.target instanceof HTMLTextAreaElement ||
				e.shiftKey ||
				e.ctrlKey ||
				e.metaKey
			) {
				return
			}
			const currentIndex = systems.findIndex((s) => s.name === name)
			if (currentIndex === -1 || systems.length <= 1) {
				return
			}
			switch (e.key) {
				case "ArrowLeft":
				case "h":
					const prevIndex = (currentIndex - 1 + systems.length) % systems.length
					return navigate(getPagePath($router, "system", { name: systems[prevIndex].name }))
				case "ArrowRight":
				case "l":
					const nextIndex = (currentIndex + 1) % systems.length
					return navigate(getPagePath($router, "system", { name: systems[nextIndex].name }))
			}
		}
		return listen(document, "keyup", handleKeyUp)
	}, [name, systems])

	if (!system.id) {
		return null
	}

	let translatedStatus: string = system.status
	if (system.status === "up") {
		translatedStatus = t({ message: "Up", comment: "Context: System is up" })
	} else if (system.status === "down") {
		translatedStatus = t({ message: "Down", comment: "Context: System is down" })
	}

	return (
		<div className="grid gap-4 mb-10 overflow-x-clip">
			{/* system info */}
			<Card>
				<div className="grid xl:flex gap-4 px-4 sm:px-6 pt-3 sm:pt-4 pb-5">
					<div>
						<h1 className="text-[1.6rem] font-semibold mb-1.5">{system.name}</h1>
						<div className="flex flex-wrap items-center gap-3 gap-y-2 text-sm opacity-90">
							<div className="capitalize flex gap-2 items-center">
								<span className={cn("relative flex h-3 w-3")}>
									{system.status === "up" && (
										<span
											className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"
											style={{ animationDuration: "1.5s" }}
										></span>
									)}
									<span
										className={cn("relative inline-flex rounded-full h-3 w-3", {
											"bg-green-500": system.status === "up",
											"bg-red-500": system.status === "down",
											"bg-primary/40": system.status === "paused",
											"bg-yellow-500": system.status === "pending",
										})}
									></span>
								</span>
								{translatedStatus}
							</div>
							{systemInfo.map(({ value, label, Icon, hide }, i) => {
								if (hide || !value) {
									return null
								}
								const content = (
									<div className="flex gap-1.5 items-center">
										<Icon className="h-4 w-4" /> {value}
									</div>
								)
								return (
									<div key={i} className="contents">
										<Separator orientation="vertical" className="h-4 bg-primary/30" />
										{label ? (
											<TooltipProvider>
												<Tooltip delayDuration={150}>
													<TooltipTrigger asChild>{content}</TooltipTrigger>
													<TooltipContent>{label}</TooltipContent>
												</Tooltip>
											</TooltipProvider>
										) : (
											content
										)}
									</div>
								)
							})}
						</div>
					</div>
					<div className="xl:ms-auto flex items-center gap-2 max-sm:-mb-1">
						<SystemConfigDialog system={system} />
						<ChartTimeSelect className="w-full xl:w-40" />
						<TooltipProvider delayDuration={100}>
							<Tooltip>
								<TooltipTrigger asChild>
									<Button
										aria-label={t`Toggle grid`}
										variant="outline"
										size="icon"
										className="hidden xl:flex p-0 text-primary"
										onClick={() => setGrid(!grid)}
									>
										{grid ? (
											<LayoutGridIcon className="h-[1.2rem] w-[1.2rem] opacity-85" />
										) : (
											<Rows className="h-[1.3rem] w-[1.3rem] opacity-85" />
										)}
									</Button>
								</TooltipTrigger>
								<TooltipContent>{t`Toggle grid`}</TooltipContent>
							</Tooltip>
						</TooltipProvider>
					</div>
				</div>
			</Card>

			{/* Charts with Tabs */}
			{(pingHosts.length > 0 || dnsTargets.length > 0) ? (
				<Tabs defaultValue="ping" className="w-full">
					<TabsList className="grid w-full grid-cols-2">
						<TabsTrigger value="ping" disabled={pingHosts.length === 0}>
							{t`Ping`}
						</TabsTrigger>
						<TabsTrigger value="dns" disabled={dnsTargets.length === 0}>
							{t`DNS`}
						</TabsTrigger>
					</TabsList>
					
					<TabsContent value="ping" className="mt-6">
						{pingHosts.length > 0 ? (
							<div className="grid xl:grid-cols-2 gap-4">
								{pingHosts.map(({ host, friendlyName }) => (
									<ChartCard
										key={host}
										grid={grid}
										empty={chartLoading || pingStats.length === 0}
										title={`${friendlyName}`}
										description={t`Response time to ${host}`}
									>
										<PingChart chartData={chartData} host={host} />
									</ChartCard>
								))}
							</div>
						) : (
							<Card>
								<CardHeader>
									<CardTitle>{t`Ping Monitoring`}</CardTitle>
									<CardDescription>{t`No ping targets configured for this system`}</CardDescription>
								</CardHeader>
							</Card>
						)}
					</TabsContent>
					
					<TabsContent value="dns" className="mt-6">
						{dnsTargets.length > 0 ? (
							<div className="grid xl:grid-cols-2 gap-4">
								{dnsTargets.map(({ key, friendlyName }) => {
									// Parse the key to extract domain and server for description
									const [domainPart, rest] = key.split('@')
									const [server] = rest.split('#')
									
									// Get protocol from DNS config for this target
									let protocol = 'UDP'
									if (system.monitoring_config?.dns?.targets) {
										// Try to find the target by matching domain, server, and type
										const target = system.monitoring_config.dns.targets.find((t: any) => {
											const configTypePart = t.type && t.type.trim() ? t.type : ''
											const configKey = `${t.domain}@${t.server}#${configTypePart}`
											return configKey === key
										})
										
										if (target?.protocol) {
											protocol = target.protocol.toUpperCase()
										} else {
											// Fallback: try to find by domain and server only
											const fallbackTarget = system.monitoring_config.dns.targets.find((t: any) => 
												t.domain === domainPart && t.server === server
											)
											if (fallbackTarget?.protocol) {
												protocol = fallbackTarget.protocol.toUpperCase()
											}
										}
									}
									
									return (
										<ChartCard
											key={key}
											grid={grid}
											empty={chartLoading || dnsStats.length === 0}
											title={friendlyName}
											description={t`DNS lookup performance for ${domainPart} @ ${server} (${protocol})`}
										>
											<DnsChart chartData={chartData} targetKey={key} />
										</ChartCard>
									)
								})}
							</div>
						) : (
							<Card>
								<CardHeader>
									<CardTitle>{t`DNS Monitoring`}</CardTitle>
									<CardDescription>{t`No DNS targets configured for this system`}</CardDescription>
								</CardHeader>
							</Card>
						)}
					</TabsContent>
				</Tabs>
			) : !chartLoading && (
				<Card>
					<CardHeader>
						<CardTitle>{t`Monitoring`}</CardTitle>
						<CardDescription>{t`No monitoring targets configured for this system`}</CardDescription>
					</CardHeader>
				</Card>
			)}
		</div>
	)
}

function ChartCard({
	title,
	description,
	children,
	grid,
	empty,
}: {
	title: string
	description: string
	children: React.ReactNode
	grid?: boolean
	empty?: boolean
}) {
	const { isIntersecting, ref } = useIntersectionObserver()

	return (
		<Card className={cn("pb-2 sm:pb-4", { "col-span-full": !grid })} ref={ref}>
			<CardHeader className="pb-5 pt-4 relative space-y-1 max-sm:py-3 max-sm:px-4">
				<CardTitle className="text-xl sm:text-2xl">{title}</CardTitle>
				<CardDescription>{description}</CardDescription>
			</CardHeader>
			<div className="w-[calc(100%-0.5em)] h-48 md:h-52 relative group">
				{empty ? (
					<Spinner msg={t`Waiting for ping data to display`} />
				) : (
					isIntersecting && children
				)}
			</div>
		</Card>
	)
}
