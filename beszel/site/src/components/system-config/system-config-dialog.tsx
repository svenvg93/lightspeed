import { t } from "@lingui/core/macro"
import { Trans } from "@lingui/react/macro"
import { memo, useMemo, useState, useEffect } from "react"
import {
	Dialog,
	DialogTrigger,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog"
import { ActivityIcon, PlusIcon, TrashIcon, SettingsIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { SystemRecord } from "@/types"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { toast } from "@/components/ui/use-toast"
import { pb } from "@/lib/stores"
import { Card, CardContent } from "@/components/ui/card"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

interface PingTarget {
	host: string
	friendly_name?: string
	count: number
	timeout: number
}

interface DnsTarget {
	domain: string
	server: string
	type: string
	timeout: number
	friendly_name?: string
	protocol?: "udp" | "tcp" | "doh" | "dot"
}

interface HttpTarget {
	url: string
	friendly_name?: string
	timeout: number
}

interface SpeedtestTarget {
	server_id: string
	friendly_name?: string
	timeout: number
}

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



function PingConfigTab({ 
	pingConfig, 
	setPingConfig 
}: { 
	pingConfig: { targets: PingTarget[], interval: string }
	setPingConfig: (config: { targets: PingTarget[], interval: string }) => void
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
				<Label htmlFor="ping-interval">{t`Check Interval`}</Label>
				<Input
					id="ping-interval"
					placeholder="*/3 * * * *"
					value={pingConfig.interval}
					onChange={(e) => setPingConfig({ ...pingConfig, interval: e.target.value })}
				/>
				<p className="text-sm text-muted-foreground">
					{t`Cron expression (e.g., "*/3 * * * *" for every 3 minutes)`}
				</p>
			</div>

			<Separator />

			<div className="space-y-4">
				<div className="flex items-center justify-between">
					<h3 className="text-lg font-medium">{t`Ping Targets`}</h3>
					<Button
						type="button"
						variant="outline"
						size="sm"
						onClick={addTarget}
						className="flex items-center gap-2"
					>
						<PlusIcon className="h-4 w-4" />
						{t`Add Target`}
					</Button>
				</div>

				{pingConfig.targets.length === 0 ? (
					<Card>
						<CardContent className="flex flex-col items-center justify-center py-8 text-center">
							<ActivityIcon className="h-12 w-12 text-muted-foreground mb-4" />
							<p className="text-muted-foreground mb-2">{t`No ping targets configured`}</p>
							<p className="text-sm text-muted-foreground">
								{t`Add ping targets to monitor network connectivity.`}
							</p>
						</CardContent>
					</Card>
				) : (
					<div className="space-y-4">
						{pingConfig.targets.map((target, index) => (
							<Card key={index}>
								<CardContent className="p-4">
									<div className="flex items-center justify-between mb-4">
										<h4 className="font-medium">{t`Target ${index + 1}`}</h4>
										<Button
											type="button"
											variant="ghost"
											size="sm"
											onClick={() => removeTarget(index)}
											className="text-destructive hover:text-destructive"
										>
											<TrashIcon className="h-4 w-4" />
										</Button>
									</div>
									
									<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
										<div className="space-y-2">
											<Label htmlFor={`ping-host-${index}`}>{t`Host`}</Label>
											<Input
												id={`ping-host-${index}`}
												placeholder="example.com"
												value={target.host}
												onChange={(e) => updateTargetString(index, 'host', e.target.value)}
											/>
										</div>
										
										<div className="space-y-2">
											<Label htmlFor={`ping-friendly-${index}`}>{t`Friendly Name (Optional)`}</Label>
											<Input
												id={`ping-friendly-${index}`}
												placeholder="My Server"
												value={target.friendly_name || ''}
												onChange={(e) => updateTargetString(index, 'friendly_name', e.target.value)}
											/>
										</div>
										
										<div className="space-y-2">
											<Label htmlFor={`ping-count-${index}`}>{t`Count`}</Label>
											<Input
												id={`ping-count-${index}`}
												type="number"
												min="1"
												max="10"
												value={target.count}
												onChange={(e) => updateTargetNumber(index, 'count', parseInt(e.target.value) || 4)}
											/>
										</div>
										
										<div className="space-y-2">
											<Label htmlFor={`ping-timeout-${index}`}>{t`Timeout (seconds)`}</Label>
											<Input
												id={`ping-timeout-${index}`}
												type="number"
												min="1"
												max="30"
												value={target.timeout}
												onChange={(e) => updateTargetNumber(index, 'timeout', parseInt(e.target.value) || 5)}
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

function DnsConfigTab({ 
	dnsConfig, 
	setDnsConfig 
}: { 
	dnsConfig: { targets: DnsTarget[], interval: string }
	setDnsConfig: (config: { targets: DnsTarget[], interval: string }) => void
}): JSX.Element {

	const addTarget = () => {
		setDnsConfig({
			...dnsConfig,
			targets: [
				...dnsConfig.targets,
				{
					domain: '',
					server: '8.8.8.8',
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
				<Label htmlFor="dns-interval">{t`Check Interval`}</Label>
				<Input
					id="dns-interval"
					placeholder="*/5 * * * *"
					value={dnsConfig.interval}
					onChange={(e) => setDnsConfig({ ...dnsConfig, interval: e.target.value })}
				/>
				<p className="text-sm text-muted-foreground">
					{t`Cron expression (e.g., "*/5 * * * *" for every 5 minutes)`}
				</p>
			</div>

			<Separator />

			<div className="space-y-4">
				<div className="flex items-center justify-between">
					<h3 className="text-lg font-medium">{t`DNS Lookup Targets`}</h3>
					<Button
						type="button"
						variant="outline"
						size="sm"
						onClick={addTarget}
						className="flex items-center gap-2"
					>
						<PlusIcon className="h-4 w-4" />
						{t`Add Target`}
					</Button>
				</div>

				{dnsConfig.targets.length === 0 ? (
					<Card>
						<CardContent className="flex flex-col items-center justify-center py-8 text-center">
							<ActivityIcon className="h-12 w-12 text-muted-foreground mb-4" />
							<p className="text-muted-foreground mb-2">{t`No DNS targets configured`}</p>
							<p className="text-sm text-muted-foreground">
								{t`Add DNS lookup targets to monitor domain resolution performance.`}
							</p>
						</CardContent>
					</Card>
				) : (
					<div className="space-y-4">
						{dnsConfig.targets.map((target, index) => (
							<Card key={index}>
								<CardContent className="p-4">
									<div className="flex items-center justify-between mb-4">
										<h4 className="font-medium">{t`Target ${index + 1}`}</h4>
										<Button
											type="button"
											variant="ghost"
											size="sm"
											onClick={() => removeTarget(index)}
											className="text-destructive hover:text-destructive"
										>
											<TrashIcon className="h-4 w-4" />
										</Button>
									</div>
									
									<div className="grid grid-cols-1 md:grid-cols-3 gap-4">
										<div className="space-y-2">
											<Label htmlFor={`dns-domain-${index}`}>{t`Domain`}</Label>
											<Input
												id={`dns-domain-${index}`}
												placeholder="example.com"
												value={target.domain}
												onChange={(e) => updateTargetString(index, 'domain', e.target.value)}
											/>
										</div>
										
										<div className="space-y-2">
											<Label htmlFor={`dns-friendly-name-${index}`}>{t`Friendly Name`}</Label>
											<Input
												id={`dns-friendly-name-${index}`}
												placeholder="Google DNS"
												value={target.friendly_name || ''}
												onChange={(e) => updateTargetString(index, 'friendly_name', e.target.value)}
											/>
										</div>
										
										<div className="space-y-2">
											<Label htmlFor={`dns-server-${index}`}>{t`DNS Server`}</Label>
											<Input
												id={`dns-server-${index}`}
												placeholder="8.8.8.8"
												value={target.server}
												onChange={(e) => updateTargetString(index, 'server', e.target.value)}
											/>
										</div>
										
										<div className="space-y-2">
											<Label htmlFor={`dns-type-${index}`}>{t`Record Type`}</Label>
											<Select value={target.type} onValueChange={(value) => updateTargetType(index, value)}>
												<SelectTrigger>
													<SelectValue />
												</SelectTrigger>
												<SelectContent>
													{DNS_TYPES.map(type => (
														<SelectItem key={type.value} value={type.value}>
															{type.label}
														</SelectItem>
													))}
												</SelectContent>
											</Select>
										</div>
										
										<div className="space-y-2">
											<Label htmlFor={`dns-protocol-${index}`}>{t`Protocol`}</Label>
											<Select value={target.protocol || 'udp'} onValueChange={(value) => updateTargetProtocol(index, value as "udp" | "tcp" | "doh" | "dot")}>
												<SelectTrigger>
													<SelectValue />
												</SelectTrigger>
												<SelectContent>
													{DNS_PROTOCOLS.map(protocol => (
														<SelectItem key={protocol.value} value={protocol.value}>
															{protocol.label}
														</SelectItem>
													))}
												</SelectContent>
											</Select>
										</div>
										
										<div className="space-y-2">
											<Label htmlFor={`dns-timeout-${index}`}>{t`Timeout (seconds)`}</Label>
											<Input
												id={`dns-timeout-${index}`}
												type="number"
												min="1"
												max="30"
												value={target.timeout}
												onChange={(e) => updateTargetNumber(index, 'timeout', parseInt(e.target.value) || 5)}
											/>
										</div>
									</div>
								</CardContent>
							</Card>
						))}
					</div>
				)}
			</div>

			<Separator />


		</div>
	)
}

function HttpConfigTab({ 
	httpConfig, 
	setHttpConfig 
}: { 
	httpConfig: { targets: HttpTarget[], interval: string }
	setHttpConfig: (config: { targets: HttpTarget[], interval: string }) => void
}): JSX.Element {

	const addTarget = () => {
		setHttpConfig({
			...httpConfig,
			targets: [...httpConfig.targets, {
				url: "",
				friendly_name: "",
				timeout: 10
			}]
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
				i === index ? { ...target, [field]: value } : target
			)
		})
	}

	const updateTargetNumber = (index: number, field: 'timeout', value: number) => {
		setHttpConfig({
			...httpConfig,
			targets: httpConfig.targets.map((target, i) => 
				i === index ? { ...target, [field]: value } : target
			)
		})
	}



	return (
		<div className="space-y-6">
			{/* Check Interval at the top */}
			<div className="space-y-2">
				<Label htmlFor="http-interval">{t`Check Interval`}</Label>
				<Input
					id="http-interval"
					placeholder="*/2 * * * *"
					value={httpConfig.interval}
					onChange={(e) => setHttpConfig({ ...httpConfig, interval: e.target.value })}
				/>
				<p className="text-sm text-muted-foreground">
					{t`Cron expression (e.g., "*/2 * * * *" for every 2 minutes)`}
				</p>
			</div>

			<Separator />

			<div className="space-y-4">
				<div className="flex items-center justify-between">
					<h3 className="text-lg font-medium">{t`HTTP Targets`}</h3>
					<Button
						type="button"
						variant="outline"
						size="sm"
						onClick={addTarget}
						className="flex items-center gap-2"
					>
						<PlusIcon className="h-4 w-4" />
						{t`Add Target`}
					</Button>
				</div>

				{httpConfig.targets.length === 0 ? (
					<Card>
						<CardContent className="flex flex-col items-center justify-center py-8 text-center">
							<ActivityIcon className="h-12 w-12 text-muted-foreground mb-4" />
							<p className="text-muted-foreground mb-2">{t`No HTTP targets configured`}</p>
							<p className="text-sm text-muted-foreground">
								{t`Add HTTP targets to monitor website availability and response times.`}
							</p>
						</CardContent>
					</Card>
				) : (
					<div className="space-y-4">
						{httpConfig.targets.map((target, index) => (
							<Card key={index}>
								<CardContent className="p-4">
									<div className="flex items-center justify-between mb-4">
										<h4 className="font-medium">{t`Target ${index + 1}`}</h4>
										<Button
											onClick={() => removeTarget(index)}
											variant="ghost"
											size="sm"
											className="text-destructive hover:text-destructive"
										>
											<TrashIcon className="h-4 w-4" />
										</Button>
									</div>
									
									<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
										<div className="space-y-2">
											<Label htmlFor={`http-url-${index}`}>{t`URL`}</Label>
											<Input
												id={`http-url-${index}`}
												placeholder="https://example.com"
												value={target.url}
												onChange={(e) => updateTargetString(index, 'url', e.target.value)}
											/>
										</div>
										
										<div className="space-y-2">
											<Label htmlFor={`http-friendly-name-${index}`}>{t`Friendly Name`}</Label>
											<Input
												id={`http-friendly-name-${index}`}
												placeholder="Example Website"
												value={target.friendly_name || ""}
												onChange={(e) => updateTargetString(index, 'friendly_name', e.target.value)}
											/>
										</div>
										
										<div className="space-y-2">
											<Label htmlFor={`http-timeout-${index}`}>{t`Timeout (seconds)`}</Label>
											<Input
												id={`http-timeout-${index}`}
												type="number"
												min="1"
												max="60"
												value={target.timeout}
												onChange={(e) => updateTargetNumber(index, 'timeout', parseInt(e.target.value) || 10)}
											/>
										</div>
									</div>
								</CardContent>
							</Card>
						))}
					</div>
				)}
			</div>

			<Separator />


		</div>
	)
}

function SpeedtestConfigTab({ 
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
				<Label htmlFor="speedtest-interval">{t`Check Interval`}</Label>
				<Input
					id="speedtest-interval"
					value={speedtestConfig.interval}
					onChange={(e) => setSpeedtestConfig({ ...speedtestConfig, interval: e.target.value })}
					placeholder="0 */6 * * *"
				/>
				<p className="text-sm text-muted-foreground">
					{t`Cron expression for speedtest frequency (e.g., "0 */6 * * *" for every 6 hours)`}
				</p>
			</div>

			<Separator />

			<div className="space-y-4">
				<div className="flex items-center justify-between">
					<h3 className="text-lg font-medium">{t`Speedtest Targets`}</h3>
					<Button
						type="button"
						variant="outline"
						size="sm"
						onClick={addTarget}
						className="flex items-center gap-2"
					>
						<PlusIcon className="h-4 w-4" />
						{t`Add Target`}
					</Button>
				</div>

				{speedtestConfig.targets.length === 0 ? (
					<Card>
						<CardContent className="flex flex-col items-center justify-center py-8 text-center">
							<ActivityIcon className="h-12 w-12 text-muted-foreground mb-4" />
							<p className="text-muted-foreground mb-2">{t`No speedtest targets configured`}</p>
							<p className="text-sm text-muted-foreground">
								{t`Add speedtest targets to monitor network performance using Ookla speedtest CLI.`}
							</p>
						</CardContent>
					</Card>
				) : (
					<div className="space-y-4">
						{speedtestConfig.targets.map((target, index) => (
							<Card key={index}>
								<CardContent className="p-4">
									<div className="flex items-center justify-between mb-4">
										<h4 className="font-medium">{t`Target ${index + 1}`}</h4>
										<Button
											type="button"
											variant="ghost"
											size="sm"
											onClick={() => removeTarget(index)}
											className="text-destructive hover:text-destructive"
										>
											<TrashIcon className="h-4 w-4" />
										</Button>
									</div>
									<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
										<div className="space-y-2">
											<Label htmlFor={`speedtest-server-${index}`}>{t`Server ID`}</Label>
											<Input
												id={`speedtest-server-${index}`}
												value={target.server_id}
												onChange={(e) => updateTargetString(index, 'server_id', e.target.value)}
												placeholder="52365"
											/>
											<p className="text-sm text-muted-foreground">
												{t`Ookla speedtest server ID (leave empty for auto-selection)`}
											</p>
										</div>
										<div className="space-y-2">
											<Label htmlFor={`speedtest-friendly-${index}`}>{t`Friendly Name`}</Label>
											<Input
												id={`speedtest-friendly-${index}`}
												value={target.friendly_name || ''}
												onChange={(e) => updateTargetString(index, 'friendly_name', e.target.value)}
												placeholder="My ISP"
											/>
										</div>
										<div className="space-y-2">
											<Label htmlFor={`speedtest-timeout-${index}`}>{t`Timeout (seconds)`}</Label>
											<Input
												id={`speedtest-timeout-${index}`}
												type="number"
												value={target.timeout}
												onChange={(e) => updateTargetNumber(index, 'timeout', parseInt(e.target.value) || 60)}
												min="30"
												max="300"
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

export const SystemConfigDialog = memo(function SystemConfigDialog({ system }: { system: SystemRecord }) {
	// Unified state management
	const [pingConfig, setPingConfig] = useState<{ targets: PingTarget[], interval: string }>({ targets: [], interval: "*/3 * * * *" })
	const [dnsConfig, setDnsConfig] = useState<{ targets: DnsTarget[], interval: string }>({ targets: [], interval: "*/5 * * * *" })
	const [httpConfig, setHttpConfig] = useState<{ targets: HttpTarget[], interval: string }>({ targets: [], interval: "*/2 * * * *" })
	const [speedtestConfig, setSpeedtestConfig] = useState<{ targets: SpeedtestTarget[], interval: string }>({ targets: [], interval: "0 */6 * * *" })
	const [isLoading, setIsLoading] = useState(false)
	const [isConfigLoading, setIsConfigLoading] = useState(true)
	const [monitoringConfigId, setMonitoringConfigId] = useState<string | null>(null)

	// Load existing configs from the monitoring_config collection
	useEffect(() => {
		const loadMonitoringConfig = async () => {
			setIsConfigLoading(true)
			try {
				// Try to find existing monitoring config for this system
				console.log("🔍 Debug Config Dialog - Loading config for system:", system.id)
				
				// Add a small delay to ensure the system is properly loaded
				await new Promise(resolve => setTimeout(resolve, 100))
				
				const existingConfig = await pb.collection("monitoring_config").getFirstListItem(`system = "${system.id}"`)
				
				if (existingConfig) {
					setMonitoringConfigId(existingConfig.id)
					console.log("🔍 Debug Config Dialog - Found existing monitoring config:", existingConfig.id)
					
					// Parse ping configuration
					if (existingConfig.ping) {
						const pingData = typeof existingConfig.ping === 'string' ? JSON.parse(existingConfig.ping) : existingConfig.ping
						setPingConfig({
							targets: pingData.targets || [],
							interval: pingData.interval || "*/3 * * * *"
						})
					}
					
					// Parse DNS configuration
					if (existingConfig.dns) {
						const dnsData = typeof existingConfig.dns === 'string' ? JSON.parse(existingConfig.dns) : existingConfig.dns
						setDnsConfig({
							targets: dnsData.targets || [],
							interval: dnsData.interval || "*/5 * * * *"
						})
					}
					
					// Parse HTTP configuration
					if (existingConfig.http) {
						const httpData = typeof existingConfig.http === 'string' ? JSON.parse(existingConfig.http) : existingConfig.http
						setHttpConfig({
							targets: httpData.targets || [],
							interval: httpData.interval || "*/2 * * * *"
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
				// No existing config found, use defaults
				console.log("🔍 Debug Config Dialog - No existing monitoring config found, using defaults. Error:", error)
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
				toast({
					title: t`Invalid Configuration`,
					description: pingError || dnsError || httpError || speedtestError,
					variant: "destructive",
				})
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

			// Save to the monitoring_config collection
			console.log("🔍 Debug Config Dialog - Saving config. monitoringConfigId:", monitoringConfigId)
			if (monitoringConfigId) {
				// Update existing record
				console.log("🔍 Debug Config Dialog - Updating existing record:", monitoringConfigId)
				await pb.collection("monitoring_config").update(monitoringConfigId, monitoringConfigData)
			} else {
				// Create new record
				console.log("🔍 Debug Config Dialog - Creating new record")
				const newRecord = await pb.collection("monitoring_config").create(monitoringConfigData)
				setMonitoringConfigId(newRecord.id)
				console.log("🔍 Debug Config Dialog - Created new record with ID:", newRecord.id)
			}

			toast({
				title: t`Configuration Saved`,
				description: t`All monitoring configurations have been updated successfully. Remember to restart the agent for changes to take effect.`,
			})
		} catch (error) {
			console.error("Failed to save configs:", error)
			toast({
				title: t`Error`,
				description: t`Failed to save configuration. Please try again.`,
				variant: "destructive",
			})
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
						{t`Configure monitoring targets and intervals for ${system.name}. Note: The agent will need to be restarted for configuration changes to take effect.`}
					</DialogDescription>
				</DialogHeader>
				<Tabs defaultValue="speedtest" className="w-full">
					<TabsList className="grid w-full grid-cols-4">
						<TabsTrigger value="speedtest">{t`Speedtest`}</TabsTrigger>
						<TabsTrigger value="ping">{t`Ping`}</TabsTrigger>
						<TabsTrigger value="dns">{t`DNS`}</TabsTrigger>
						<TabsTrigger value="http">{t`HTTP`}</TabsTrigger>
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
