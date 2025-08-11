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
import { ActivityIcon, PlusIcon, TrashIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { SystemRecord } from "@/types"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { toast } from "@/components/ui/use-toast"
import { pb } from "@/lib/stores"
import { Card, CardContent } from "@/components/ui/card"

interface PingTarget {
	host: string
	friendly_name?: string
	count: number
	timeout: number
}

interface PingConfig {
	targets: PingTarget[]
	interval: string // Cron expression (e.g., "*/3 * * * *" for every 3 minutes)
}

export default memo(function PingConfigButton({ system }: { system: SystemRecord }) {
	const [opened, setOpened] = useState(false)

	const hasPingConfig = system.ping_config && Array.isArray(system.ping_config?.targets) && system.ping_config.targets.length > 0

	return useMemo(
		() => (
			<Dialog>
				<DialogTrigger asChild>
					<Button variant="ghost" size="icon" aria-label={t`Ping Configuration`} data-nolink onClick={() => setOpened(true)}>
						<ActivityIcon
							className={`h-[1.2em] w-[1.2em] pointer-events-none ${hasPingConfig ? 'text-primary' : ''}`}
						/>
					</Button>
				</DialogTrigger>
				<DialogContent className="max-h-full sm:max-h-[95svh] overflow-auto max-w-[37rem]">
					{opened && <PingConfigDialogContent key={`${system.id}-${Date.now()}`} system={system} />}
				</DialogContent>
			</Dialog>
		),
		[opened, hasPingConfig]
	)
})

function PingConfigDialogContent({ system }: { system: SystemRecord }): JSX.Element {
	const [pingConfig, setPingConfig] = useState<PingConfig>({ targets: [], interval: "*/3 * * * *" })
	const [isLoading, setIsLoading] = useState(false)

		// Load existing ping config
	useEffect(() => {
		if (system.ping_config && Array.isArray(system.ping_config.targets)) {
			// Convert old interval format to cron expression if needed
			let interval = system.ping_config.interval
			if (typeof interval === 'number') {
				interval = `*/${interval} * * * *`
			} else if (typeof interval !== 'string') {
				interval = "*/3 * * * *"
			}
			
			// Ensure all targets have the friendly_name field
			const targetsWithFriendlyName = system.ping_config.targets.map(target => ({
				...target,
				friendly_name: target.friendly_name || ''
			}))
			
			setPingConfig({
				targets: targetsWithFriendlyName,
				interval: interval
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

	// Validate cron expression to catch common mistakes
	const validateCronExpression = (expression: string): string | null => {
		// Check for basic format
		const parts = expression.split(' ')
		if (parts.length !== 5) {
			return "Invalid cron expression: Must have exactly 5 fields (minute hour day month weekday)"
		}
		
		// Check for continuous execution patterns that might be too frequent
		if (parts[0] === '*' && parts[1] === '*') {
			return "Invalid cron expression: Running every minute is too frequent for ping tests"
		}
		
		return null
	}

	const savePingConfig = async () => {
		// Validate cron expression
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
			// Validate targets
			const validTargets = pingConfig.targets.filter(target => 
				target.host.trim() !== '' && 
				target.count > 0 && 
				target.timeout > 0
			)

			// Update the system record with the ping config
			await pb.collection("systems").update(system.id, {
				ping_config: {
					targets: validTargets.map(target => ({
						...target,
						host: target.host.trim()
					})),
					interval: pingConfig.interval
				}
			})

			toast({
				title: t`Ping configuration saved`,
				description: t`Ping targets have been updated for ${system.name}.`,
			})
		} catch (error: any) {
			toast({
				title: t`Failed to save ping configuration`,
				description: error.message || t`Please check logs for more details.`,
				variant: "destructive",
			})
		} finally {
			setIsLoading(false)
		}
	}

	return (
		<>
			<DialogHeader>
				<DialogTitle className="text-xl">
					<Trans>Ping Configuration</Trans>
				</DialogTitle>
				<DialogDescription>
					<Trans>Configure ping monitoring targets for {system.name}. Agent need to be restarted for changes to take effect.</Trans>
				</DialogDescription>
			</DialogHeader>

			<div className="space-y-6">
				{/* Global Interval Setting */}
				<div className="space-y-2">
					<Label htmlFor="global-interval">
						<Trans>Cron Expression</Trans>
					</Label>
					<Input
						id="global-interval"
						type="text"
						placeholder="*/3 * * * *"
						value={pingConfig.interval}
						onChange={(e) => setPingConfig(prev => ({ ...prev, interval: e.target.value }))}
						className="max-w-64"
					/>
					<p className="text-sm text-muted-foreground">
						<Trans>5-field cron expression for ping scheduling (minute hour day month weekday)</Trans> •{" "}
						<a 
							href="https://crontab.guru" 
							target="_blank" 
							rel="noopener noreferrer"
							className="text-primary hover:underline"
						>
							Cron expression generator
						</a>
					</p>
				</div>

				<Separator />

				{pingConfig.targets.length === 0 ? (
					<div className="text-center py-8 text-muted-foreground">
						<ActivityIcon className="h-12 w-12 mx-auto mb-4 opacity-50" />
						<p><Trans>No ping targets configured.</Trans></p>
						<p className="text-sm"><Trans>Add targets to monitor network connectivity.</Trans></p>
					</div>
				) : (
					<div className="space-y-3">
						{pingConfig.targets.map((target, index) => (
							<Card key={index} className="relative">
								<CardContent className="p-4">
									<Button
										variant="ghost"
										size="icon"
										className="absolute top-2 right-2 h-8 w-8 text-muted-foreground hover:text-destructive"
										onClick={() => removeTarget(index)}
									>
										<TrashIcon className="h-4 w-4" />
									</Button>
									
									<div className="grid gap-4 pr-10">
										<div className="space-y-2">
											<Label htmlFor={`friendly-name-${index}`}>
												<Trans>Friendly Name (Optional)</Trans>
											</Label>
											<Input
												id={`friendly-name-${index}`}
												placeholder="e.g., Google DNS"
												value={target.friendly_name || ''}
												onChange={(e) => updateTargetString(index, 'friendly_name', e.target.value)}
											/>
										</div>
										<div className="space-y-2">
											<Label htmlFor={`host-${index}`}>
												<Trans>Host</Trans>
											</Label>
											<Input
												id={`host-${index}`}
												placeholder="example.com or 8.8.8.8"
												value={target.host}
												onChange={(e) => updateTargetString(index, 'host', e.target.value)}
											/>
										</div>
									
										
										<div className="grid grid-cols-2 gap-3">
											<div className="space-y-2">
												<Label htmlFor={`count-${index}`}>
													<Trans>Count</Trans>
												</Label>
												<Input
													id={`count-${index}`}
													type="number"
													min="1"
													max="20"
													value={target.count}
													onChange={(e) => updateTargetNumber(index, 'count', parseInt(e.target.value) || 1)}
												/>
											</div>
											
											<div className="space-y-2">
												<Label htmlFor={`timeout-${index}`}>
													<Trans>Timeout (s)</Trans>
												</Label>
												<Input
													id={`timeout-${index}`}
													type="number"
													min="1"
													max="30"
													value={target.timeout}
													onChange={(e) => updateTargetNumber(index, 'timeout', parseInt(e.target.value) || 5)}
												/>
											</div>
										</div>
									</div>
								</CardContent>
							</Card>
						))}
					</div>
				)}

				<div className="flex gap-2">
					<Button variant="outline" onClick={addTarget} className="flex-1">
						<PlusIcon className="h-4 w-4 mr-2" />
						<Trans>Add Target</Trans>
					</Button>
				</div>

				<Separator />

				<div className="flex gap-2">
					<Button onClick={savePingConfig} disabled={isLoading} className="flex-1">
						{isLoading ? t`Saving...` : t`Save Configuration`}
					</Button>
				</div>

				<div className="text-sm text-muted-foreground space-y-2">
					<p><Trans>• Host: Domain name or IP address to ping</Trans></p>
					<p><Trans>• Count: Number of ping packets to send per test</Trans></p>
					<p><Trans>• Cron Expression: Schedule for ping tests</Trans></p>
					<p><Trans>• Timeout: Maximum seconds to wait for response</Trans></p>
				</div>
			</div>
		</>
	)
}
