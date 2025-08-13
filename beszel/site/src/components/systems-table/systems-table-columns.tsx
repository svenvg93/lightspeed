import { SystemRecord } from "@/types"
import { ColumnDef, HeaderContext } from "@tanstack/react-table"


import { ClassValue } from "clsx"
import {
	ArrowUpDownIcon,
	CopyIcon,
	ClockArrowUp,
	GlobeIcon,
	GitPullRequestIcon,
	MoreHorizontalIcon,
	PauseCircleIcon,
	MapPinHouseIcon,
	PenBoxIcon,
	PlayCircleIcon,
	ServerIcon,
	Trash2Icon,
	DownloadIcon,
	UploadIcon,
	TagsIcon,
	ActivityIcon,
} from "lucide-react"
import { Button } from "../ui/button"
import {
	Tooltip,
	TooltipContent,
	TooltipProvider,
	TooltipTrigger,
} from "../ui/tooltip"
import {
	cn,
	copyToClipboard,
	generateToken,
	getHubURL,
	isAdmin,
	tokenMap,
} from "@/lib/utils"
import { pb } from "@/lib/stores"
import { Trans, useLingui } from "@lingui/react/macro"
import { useMemo, useRef, useState, useEffect } from "react"
import { memo } from "react"
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "../ui/dropdown-menu"
import AlertButton from "../alerts/alert-button"
import { SystemConfigDialog } from "../system-config/system-config-dialog"
import { Dialog } from "../ui/dialog"
import { SystemDialog } from "../add-system"
import { AlertDialog } from "../ui/alert-dialog"
import {
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "../ui/alert-dialog"
import { buttonVariants } from "../ui/button"
import { t } from "@lingui/core/macro"



const STATUS_COLORS = {
	up: "bg-green-500",
	down: "bg-red-500",
	paused: "bg-primary/40",
	pending: "bg-yellow-500",
} as const

/**
 * @param viewMode - "table" or "grid"
 * @returns - Column definitions for the systems table
 */
export default function SystemsTableColumns(viewMode: "table" | "grid"): ColumnDef<SystemRecord>[] {
	// @ts-ignore - Complex table configuration with implicit types
	return [
		{
			size: 200,
			minSize: 0,
			accessorKey: "name",
			id: "system",
			filterFn: (() => {
				let filterInput = ""
				let filterInputLower = ""
				const nameCache = new Map<string, string>()
				const statusTranslations = {
					up: t`Up`.toLowerCase(),
					down: t`Down`.toLowerCase(),
					paused: t`Paused`.toLowerCase(),
				} as const

				// match filter value against name, tags, or translated status
				return (row: any, _: any, newFilterInput: any) => {
					const { name, status, tags } = row.original
					if (newFilterInput !== filterInput) {
						filterInput = newFilterInput
						filterInputLower = newFilterInput.toLowerCase()
					}
					let nameLower = nameCache.get(name)
					if (nameLower === undefined) {
						nameLower = name?.toLowerCase() || ""
						if (name) nameCache.set(name, nameLower || "")
					}
					if (nameLower && nameLower.includes(filterInputLower)) {
						return true
					}
					// Check tags
					if (tags && Array.isArray(tags)) {
						for (const tag of tags) {
							if (tag.toLowerCase().includes(filterInputLower)) {
								return true
							}
						}
					}
					const statusLower = statusTranslations[status as keyof typeof statusTranslations]
					return statusLower?.includes(filterInputLower) || false
				}
			})(),
			enableHiding: false,
			invertSorting: false,
			Icon: ServerIcon,
			cell: (info: any) => (
				<span className="flex gap-2 items-center font-medium text-sm text-nowrap md:ps-1 md:pe-5">
					<IndicatorDot system={info.row.original} />
					{info.getValue() as string}
				</span>
			),
			header: sortableHeader,
		},
		{
			accessorFn: ({ tags }: { tags?: string[] }) => tags || [],
			id: "tags",
			name: () => t`Tags`,
			size: 120,
			Icon: TagsIcon,
			header: (context: any) => (
				<TooltipProvider>
					<Tooltip>
						<TooltipTrigger asChild>
							{sortableHeader(context)}
						</TooltipTrigger>
						<TooltipContent>
							{t`Tags for filtering and organization`}
						</TooltipContent>
					</Tooltip>
				</TooltipProvider>
			),
			cell({ getValue }: { getValue: () => any }) {
				const tags = getValue() as string[]
				if (!tags || tags.length === 0) {
					return null
				}
				
				// Show only first tag, with +X more indicator
				const firstTag = tags[0]
				const remainingCount = tags.length - 1
				
				return (
					<div className="flex items-center gap-1">
						<span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200">
							{firstTag}
						</span>
						{remainingCount > 0 && (
							<span className="text-xs text-muted-foreground">
								+{remainingCount}
							</span>
						)}
					</div>
				)
			},
		},
		{
			accessorFn: ({ averages }: { averages?: any }) => averages?.adl || 0,
			id: "adl",
			name: () => t`Download`,
			size: 140,
			Icon: DownloadIcon,
			header: (context: any) => (
				<TooltipProvider>
					<Tooltip>
						<TooltipTrigger asChild>
							{sortableHeader(context)}
						</TooltipTrigger>
						<TooltipContent>
							{t`Average download speed across all speedtest targets`}
						</TooltipContent>
					</Tooltip>
				</TooltipProvider>
			),
			cell: ({ getValue, row, column }: { getValue: () => any; row: any; column: any }) => <SpeedMeterCell getValue={getValue} row={row} column={column} />,
		},
		{
			accessorFn: ({ averages }: { averages?: any }) => averages?.aul || 0,
			id: "aul",
			name: () => t`Upload`,
			size: 140,
			Icon: UploadIcon,
			header: (context: any) => (
				<TooltipProvider>
					<Tooltip>
						<TooltipTrigger asChild>
							{sortableHeader(context)}
						</TooltipTrigger>
						<TooltipContent>
							{t`Average upload speed across all speedtest targets`}
						</TooltipContent>
					</Tooltip>
				</TooltipProvider>
			),
			cell: ({ getValue, row, column }: { getValue: () => any; row: any; column: any }) => <SpeedMeterCell getValue={getValue} row={row} column={column} />,
		},
		{
			accessorFn: ({ averages }: { averages?: any }) => averages?.ap || 0,
			id: "ap",
			name: () => t`ICMP`,
			size: 50,
			Icon: ClockArrowUp,
						header: (context: any) => (
				<TooltipProvider>
					<Tooltip>
						<TooltipTrigger asChild>
							{sortableHeader(context)}
						</TooltipTrigger>
						<TooltipContent>
							{t`Overall average latency across all ping targets`}
						</TooltipContent>
					</Tooltip>
				</TooltipProvider>
			),
			cell: ({ getValue, row, column }: { getValue: () => any; row: any; column: any }) => <PerformanceDotCell getValue={getValue} row={row} column={column} />,
		},
		{
			accessorFn: ({ averages }: { averages?: any }) => averages?.ad || 0,
			id: "ad",
			name: () => t`DNS`,
			size: 50,
			Icon: MapPinHouseIcon,
			header: (context: any) => (
				<TooltipProvider>
					<Tooltip>
						<TooltipTrigger asChild>
							{sortableHeader(context)}
						</TooltipTrigger>
						<TooltipContent>
							{t`Overall average lookup time across all DNS targets`}
						</TooltipContent>
					</Tooltip>
				</TooltipProvider>
			),
			cell: ({ getValue, row, column }: { getValue: () => any; row: any; column: any }) => <PerformanceDotCell getValue={getValue} row={row} column={column} />,
		},
		{
			accessorFn: ({ averages }: { averages?: any }) => averages?.ah || 0,
			id: "ah",
			name: () => t`HTTP`,
			size: 50,
			Icon: GlobeIcon,
			header: (context: any) => (
				<TooltipProvider>
					<Tooltip>
						<TooltipTrigger asChild>
							{sortableHeader(context)}
						</TooltipTrigger>
						<TooltipContent>
							{t`Overall average response time across all HTTP targets`}
						</TooltipContent>
					</Tooltip>
				</TooltipProvider>
			),
			cell: ({ getValue, row, column }: { getValue: () => any; row: any; column: any }) => <PerformanceDotCell getValue={getValue} row={row} column={column} />,
		},
		{
			accessorFn: ({ info }: { info: any }) => info.v,
			id: "agent",
			name: () => t`Agent`,
			// invertSorting: true,
			size: 50,
			Icon: 	GitPullRequestIcon,				
			hideSort: true,
			header: sortableHeader,
			cell(info: any) {
				const version = info.getValue() as string
				if (!version) {
					return null
				}
				const system = info.row.original
				return (
					<span className={cn("flex gap-2 items-center md:pe-5 tabular-nums", viewMode === "table" && "ps-0.5")}>
						<IndicatorDot
							system={system}
							className={
								(system.status !== "up" && STATUS_COLORS.paused) ||
								(version === globalThis.BESZEL.HUB_VERSION && STATUS_COLORS.up) ||
								STATUS_COLORS.pending
							}
						/>
						<span className="truncate max-w-14">{info.getValue() as string}</span>
					</span>
				)
			},
		},
		{
			id: "actions",
			// @ts-ignore
			size: 50,
			cell: ({ row }: { row: any }) => (
				<div className="flex justify-end items-center gap-1 -ms-3">
					{isAdmin() && <SystemConfigDialog system={row.original} />}
					{isAdmin() && <AlertButton system={row.original} />}
					<ActionsButton system={row.original} />
				</div>
			),
		},
	] as unknown as ColumnDef<SystemRecord>[]
}

function sortableHeader(context: HeaderContext<SystemRecord, unknown>) {
	const { column } = context
	// @ts-ignore
	const { Icon, hideSort }: { Icon: React.ElementType; hideSort: boolean } = column.columnDef
	
	// Get display name from column ID
	const getDisplayName = (id: string) => {
		switch (id) {
			case "system": return t`System`
			case "adl": return t`Download`
			case "aul": return t`Upload`
			case "ap": return t`ICMP`
			case "ad": return t`DNS`
			case "ah": return t`HTTP`
			case "tags": return t`Tags`
			case "agent": return t`Agent`
			default: return id
		}
	}
	
	return (
		<Button
			variant="ghost"
			className="h-9 px-3 flex"
			onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
		>
			{Icon && <Icon className="me-2 size-4" />}
			{getDisplayName(column.id)}
			{hideSort || <ArrowUpDownIcon className="ms-2 size-4" />}
		</Button>
	)
}

export function IndicatorDot({ system, className }: { system: SystemRecord; className?: ClassValue }) {
	className ||= STATUS_COLORS[system.status as keyof typeof STATUS_COLORS] || ""
	return (
		<span
			className={cn("flex-shrink-0 size-2 rounded-full", className)}
			// style={{ marginBottom: "-1px" }}
		/>
	)
}

export const ActionsButton = memo(({ system }: { system: SystemRecord }) => {
	const [deleteOpen, setDeleteOpen] = useState(false)
	const [editOpen, setEditOpen] = useState(false)
	const [configOpen, setConfigOpen] = useState(false)
	let editOpened = useRef(false)
	let configOpened = useRef(false)
	const { t } = useLingui()
	const { id, status, host, name } = system

	return useMemo(() => {
		return (
			<>
				<DropdownMenu>
					<DropdownMenuTrigger asChild>
						<Button variant="ghost" size={"icon"} data-nolink>
							<span className="sr-only">
								<Trans>Open menu</Trans>
							</span>
							<MoreHorizontalIcon className="w-5" />
						</Button>
					</DropdownMenuTrigger>
					<DropdownMenuContent align="end">
						{isAdmin() && (
							<DropdownMenuItem
								onSelect={() => {
									editOpened.current = true
									setEditOpen(true)
								}}
							>
								<PenBoxIcon className="me-2.5 size-4" />
								<Trans>Edit</Trans>
							</DropdownMenuItem>
						)}

						{isAdmin() && (
							<DropdownMenuItem
								onClick={() => {
									pb.collection("systems").update(id, {
										status: status === "paused" ? "pending" : "paused",
									})
								}}
							>
								{status === "paused" ? (
									<>
										<PlayCircleIcon className="me-2.5 size-4" />
										<Trans>Resume</Trans>
									</>
								) : (
									<>
										<PauseCircleIcon className="me-2.5 size-4" />
										<Trans>Pause</Trans>
									</>
								)}
							</DropdownMenuItem>
						)}
						<DropdownMenuItem onClick={() => copyToClipboard(name)}>
							<CopyIcon className="me-2.5 size-4" />
							<Trans>Copy name</Trans>
						</DropdownMenuItem>
						<DropdownMenuItem onClick={() => copyToClipboard(host)}>
							<CopyIcon className="me-2.5 size-4" />
							<Trans>Copy host</Trans>
						</DropdownMenuItem>
						{isAdmin() && (
							<>
								<DropdownMenuSeparator />
								<DropdownMenuItem onSelect={() => setDeleteOpen(true)}>
									<Trash2Icon className="me-2.5 size-4" />
									<Trans>Delete</Trans>
								</DropdownMenuItem>
							</>
						)}
					</DropdownMenuContent>
				</DropdownMenu>
				{/* edit dialog */}
				<Dialog open={editOpen} onOpenChange={setEditOpen}>
					{editOpened.current && <SystemDialog system={system} setOpen={setEditOpen} />}
				</Dialog>
				{/* config dialog */}
				<Dialog open={configOpen} onOpenChange={setConfigOpen}>
					{configOpened.current && <SystemConfigDialog system={system} />}
				</Dialog>
				{/* deletion dialog */}
				<AlertDialog open={deleteOpen} onOpenChange={(open) => setDeleteOpen(open)}>
					<AlertDialogContent>
						<AlertDialogHeader>
							<AlertDialogTitle>
								<Trans>Are you sure you want to delete {name}?</Trans>
							</AlertDialogTitle>
							<AlertDialogDescription>
								<Trans>
									This action cannot be undone. This will permanently delete all current records for {name} from the
									database.
								</Trans>
							</AlertDialogDescription>
						</AlertDialogHeader>
						<AlertDialogFooter>
							<AlertDialogCancel>
								<Trans>Cancel</Trans>
							</AlertDialogCancel>
							<AlertDialogAction
								className={cn(buttonVariants({ variant: "destructive" }))}
								onClick={() => pb.collection("systems").delete(id)}
							>
								<Trans>Continue</Trans>
							</AlertDialogAction>
						</AlertDialogFooter>
					</AlertDialogContent>
				</AlertDialog>
			</>
		)
	}, [id, status, host, name, t, deleteOpen, editOpen, configOpen])
})

// SpeedMeterCell component for displaying speed meters
function SpeedMeterCell({ getValue, row, column }: { getValue: () => any; row: any; column: any }) {
	const speed = getValue() as number
	const system = row.original as SystemRecord
	
	// Get expected speed directly from system record
	const expectedSpeed = column.id === "adl" 
		? system.expected_performance?.download_speed
		: system.expected_performance?.upload_speed
	
		// Debug logging
	console.log(`SpeedMeterCell ${column.id}:`, { 
		speed, 
		expectedSpeed, 
		systemId: system.id, 
		systemStatus: system.status,
		hasSpeed: speed > 0
	})
	
	if (!speed || speed === 0) {
		return null
	}

	// If no expected speed is set, just show the value
	if (!expectedSpeed) {
		return (
					<span className="tabular-nums">
			{speed.toFixed(2)} Mbps
		</span>
		)
	}

	// Calculate percentage of expected speed
	const percentage = Math.min((speed / expectedSpeed) * 100, 100)
	
	// Determine meter state based on percentage
	let meterState = "bg-green-500"
	if (percentage < 50) {
		meterState = "bg-red-600"
	} else if (percentage < 80) {
		meterState = "bg-yellow-500"
	}

	return (
		<div className="flex gap-2 items-center tabular-nums tracking-tight">
			<span className="min-w-12">{speed.toFixed(2)} Mbps</span>
			<span className="grow min-w-8 block bg-muted h-[1em] relative rounded-sm overflow-hidden">
				<span
					className={cn(
						"absolute inset-0 w-full h-full origin-left",
						(system.status !== "up" && "bg-primary/30") ||
						meterState
					)}
					style={{
						transform: `scalex(${percentage / 100})`,
					}}
				></span>
			</span>
		</div>
	)
}

// PerformanceDotCell component for displaying performance dots
function PerformanceDotCell({ getValue, row, column }: { getValue: () => any; row: any; column: any }) {
	const value = getValue() as number
	const system = row.original as SystemRecord
	
	// Get expected value based on column type
	let expectedValue: number | undefined
	switch (column.id) {
		case "ap": // ping
			expectedValue = system.expected_performance?.ping_latency
			break
		case "ad": // DNS
			expectedValue = system.expected_performance?.dns_lookup_time
			break
		case "ah": // HTTP
			expectedValue = system.expected_performance?.http_response_time
			break
		default:
			expectedValue = undefined
	}

	if (!value || value === 0) {
		return null
	}

	if (!expectedValue) {
		return (
			<span className="tabular-nums">
				{value.toFixed(1)} ms
			</span>
		)
	}

	// For latency metrics, lower is better (opposite of speed)
	const percentage = Math.min((expectedValue / value) * 100, 100)

	let dotColor = "bg-green-500"
	if (percentage < 50) {
		dotColor = "bg-red-600"
	} else if (percentage < 80) {
		dotColor = "bg-yellow-500"
	}

	return (
		<div className="flex gap-2 items-center tabular-nums tracking-tight">
			<span className={cn(
				"size-2 rounded-full",
				(system.status !== "up" && "bg-primary/30") ||
				dotColor
			)} />
			<span className="min-w-8">{value.toFixed(1)} ms</span>
		</div>
	)
}
