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

interface PingConfig {
	targets: PingTarget[]
	interval: string
}

interface DnsTarget {
	domain: string
	server: string
	type: string
	timeout: number
	friendly_name?: string
	protocol?: "udp" | "tcp" | "doh" | "dot"
}

interface DnsConfig {
	targets: DnsTarget[]
	interval: string
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

function PingConfigTab({ system }: { system: SystemRecord }): JSX.Element {
	const [pingConfig, setPingConfig] = useState<PingConfig>({ targets: [], interval: "*/3 * * * *" })
	const [isLoading, setIsLoading] = useState(false)

	// Load existing ping config
	useEffect(() => {
		if (system.ping_config && Array.isArray(system.ping_config.targets)) {
			setPingConfig({
				targets: system.ping_config.targets,
				interval: String(system.ping_config.interval || "*/3 * * * *")
			})
		} else {
			setPingConfig({ targets: [], interval: "*/3 * * * *" })
		}
	}, [system.ping_config])

	const addTarget = () => {
		setPingConfig(prev => ({
			...prev,
			targets: [
				...prev.targets,
				{
					host: '',
					friendly_name: '',
					count: 4,
					timeout: 5
				}
			]
		}))
	}

	const removeTarget = (index: number) => {
		setPingConfig(prev => ({
			...prev,
			targets: prev.targets.filter((_, i) => i !== index)
		}))
	}

	const updateTargetString = (index: number, field: 'host' | 'friendly_name', value: string) => {
		setPingConfig(prev => ({
			...prev,
			targets: prev.targets.map((target, i) => 
				i === index 
					? { ...target, [field]: value }
					: target
			)
		}))
	}

	const updateTargetNumber = (index: number, field: 'count' | 'timeout', value: number) => {
		setPingConfig(prev => ({
			...prev,
			targets: prev.targets.map((target, i) => 
				i === index 
					? { ...target, [field]: value }
					: target
			)
		}))
	}

	const validateCronExpression = (expression: string): string | null => {
		const parts = expression.split(' ')
		if (parts.length !== 5) {
			return "Invalid cron expression: Must have exactly 5 fields (minute hour day month weekday)"
		}
		
		if (parts[0] === '*' && parts[1] === '*') {
			return "Invalid cron expression: Running every minute is too frequent for ping tests"
		}
		
		return null
	}

	const savePingConfig = async () => {
		const cronError = validateCronExpression(pingConfig.interval)
		if (cronError) {
			toast({
				title: t`Invalid Cron Expression`,
				description: cronError,
				variant: "destructive"
			})
			return
		}
		setIsLoading(true)
		try {
			const validTargets = pingConfig.targets.filter(target => 
				target.host.trim() !== '' && 
				target.count > 0 && 
				target.timeout > 0
			)

			if (validTargets.length === 0) {
				toast({
					title: t`No Valid Targets`,
					description: t`Please add at least one valid ping target.`,
					variant: "destructive"
				})
				return
			}

			await pb.collection("systems").update(system.id, {
				ping_config: {
					targets: validTargets,
					interval: pingConfig.interval
				}
			})

			toast({
				title: t`Ping Configuration Saved`,
				description: t`Ping monitoring configuration has been updated successfully.`,
			})
		} catch (error) {
			console.error("Failed to save ping config:", error)
			toast({
				title: t`Error`,
				description: t`Failed to save ping configuration. Please try again.`,
				variant: "destructive"
			})
		} finally {
			setIsLoading(false)
		}
	}

	const hasConfig = pingConfig.targets.length > 0

	return (
		<div className="space-y-6">
			{/* Check Interval at the top */}
			<div className="space-y-2">
				<Label htmlFor="ping-interval">{t`Check Interval`}</Label>
				<Input
					id="ping-interval"
					placeholder="*/3 * * * *"
					value={pingConfig.interval}
					onChange={(e) => setPingConfig(prev => ({ ...prev, interval: e.target.value }))}
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

			<div className="flex justify-end gap-2">
				<Button
					type="button"
					variant="outline"
					onClick={() => setPingConfig({ targets: [], interval: "*/3 * * * *" })}
					disabled={isLoading}
				>
					{t`Reset`}
				</Button>
				<Button
					type="button"
					onClick={savePingConfig}
					disabled={isLoading || !hasConfig}
				>
					{isLoading ? t`Saving...` : t`Save Ping Configuration`}
				</Button>
			</div>
		</div>
	)
}

function DnsConfigTab({ system }: { system: SystemRecord }): JSX.Element {
	const [dnsConfig, setDnsConfig] = useState<DnsConfig>({ targets: [], interval: "*/5 * * * *" })
	const [isLoading, setIsLoading] = useState(false)

	// Load existing DNS config
	useEffect(() => {
		if (system.dns_config && Array.isArray(system.dns_config.targets)) {
			setDnsConfig({
				targets: system.dns_config.targets,
				interval: String(system.dns_config.interval || "*/5 * * * *")
			})
		} else {
			setDnsConfig({ targets: [], interval: "*/5 * * * *" })
		}
	}, [system.dns_config])

	const addTarget = () => {
		setDnsConfig(prev => ({
			...prev,
			targets: [
				...prev.targets,
				{
					domain: '',
					server: '8.8.8.8',
					type: 'A',
					timeout: 5,
					friendly_name: '',
					protocol: 'udp'
				}
			]
		}))
	}

	const removeTarget = (index: number) => {
		setDnsConfig(prev => ({
			...prev,
			targets: prev.targets.filter((_, i) => i !== index)
		}))
	}

	const updateTargetString = (index: number, field: 'domain' | 'server' | 'friendly_name', value: string) => {
		setDnsConfig(prev => ({
			...prev,
			targets: prev.targets.map((target, i) => 
				i === index 
					? { ...target, [field]: value }
					: target
			)
		}))
	}

	const updateTargetType = (index: number, value: string) => {
		setDnsConfig(prev => ({
			...prev,
			targets: prev.targets.map((target, i) => 
				i === index 
					? { ...target, type: value }
					: target
			)
		}))
	}

	const updateTargetProtocol = (index: number, value: "udp" | "tcp" | "doh" | "dot") => {
		setDnsConfig(prev => ({
			...prev,
			targets: prev.targets.map((target, i) => 
				i === index 
					? { ...target, protocol: value }
					: target
			)
		}))
	}

	const updateTargetNumber = (index: number, field: 'timeout', value: number) => {
		setDnsConfig(prev => ({
			...prev,
			targets: prev.targets.map((target, i) => 
				i === index 
					? { ...target, [field]: value }
					: target
			)
		}))
	}

	const validateCronExpression = (expression: string): string | null => {
		const parts = expression.split(' ')
		if (parts.length !== 5) {
			return "Invalid cron expression: Must have exactly 5 fields (minute hour day month weekday)"
		}
		
		if (parts[0] === '*' && parts[1] === '*') {
			return "Invalid cron expression: Running every minute is too frequent for DNS tests"
		}
		
		return null
	}

	const saveDnsConfig = async () => {
		const cronError = validateCronExpression(dnsConfig.interval)
		if (cronError) {
			toast({
				title: t`Invalid Cron Expression`,
				description: cronError,
				variant: "destructive"
			})
			return
		}
		setIsLoading(true)
		try {
			const validTargets = dnsConfig.targets.filter(target => 
				target.domain.trim() !== '' && 
				target.server.trim() !== '' && 
				target.type.trim() !== '' && 
				target.timeout > 0
			)

			if (validTargets.length === 0) {
				toast({
					title: t`No Valid Targets`,
					description: t`Please add at least one valid DNS target.`,
					variant: "destructive"
				})
				return
			}

			await pb.collection("systems").update(system.id, {
				dns_config: {
					targets: validTargets,
					interval: dnsConfig.interval
				}
			})

			toast({
				title: t`DNS Configuration Saved`,
				description: t`DNS monitoring configuration has been updated successfully.`,
			})
		} catch (error) {
			console.error("Failed to save DNS config:", error)
			toast({
				title: t`Error`,
				description: t`Failed to save DNS configuration. Please try again.`,
				variant: "destructive"
			})
		} finally {
			setIsLoading(false)
		}
	}

	const hasConfig = dnsConfig.targets.length > 0

	return (
		<div className="space-y-6">
			{/* Check Interval at the top */}
			<div className="space-y-2">
				<Label htmlFor="dns-interval">{t`Check Interval`}</Label>
				<Input
					id="dns-interval"
					placeholder="*/5 * * * *"
					value={dnsConfig.interval}
					onChange={(e) => setDnsConfig(prev => ({ ...prev, interval: e.target.value }))}
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

			<div className="flex justify-end gap-2">
				<Button
					type="button"
					variant="outline"
					onClick={() => setDnsConfig({ targets: [], interval: "*/5 * * * *" })}
					disabled={isLoading}
				>
					{t`Reset`}
				</Button>
				<Button
					type="button"
					onClick={saveDnsConfig}
					disabled={isLoading || !hasConfig}
				>
					{isLoading ? t`Saving...` : t`Save DNS Configuration`}
				</Button>
			</div>
		</div>
	)
}

export const SystemConfigDialog = memo(function SystemConfigDialog({ system }: { system: SystemRecord }) {
	const hasPingConfig = system.ping_config && Array.isArray(system.ping_config.targets) && system.ping_config.targets.length > 0
	const hasDnsConfig = system.dns_config && Array.isArray(system.dns_config.targets) && system.dns_config.targets.length > 0
	const hasAnyConfig = hasPingConfig || hasDnsConfig

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
						{t`Configure monitoring targets and intervals for ${system.name}.`}
					</DialogDescription>
				</DialogHeader>
				<Tabs defaultValue="ping" className="w-full">
					<TabsList className="grid w-full grid-cols-2">
						<TabsTrigger value="ping">{t`Ping`}</TabsTrigger>
						<TabsTrigger value="dns">{t`DNS`}</TabsTrigger>
					</TabsList>
					<TabsContent value="ping" className="mt-6">
						<PingConfigTab system={system} />
					</TabsContent>
					<TabsContent value="dns" className="mt-6">
						<DnsConfigTab system={system} />
					</TabsContent>
				</Tabs>
			</DialogContent>
		</Dialog>
	)
})
