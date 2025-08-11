import { SystemRecord } from "@/types"
import { CellContext, ColumnDef, HeaderContext } from "@tanstack/react-table"
import { ClassValue } from "clsx"
import {
	ArrowUpDownIcon,
	CopyIcon,
	ClockArrowUp,
	MoreHorizontalIcon,
	PauseCircleIcon,
	PenBoxIcon,
	PlayCircleIcon,
	ServerIcon,
	Trash2Icon,
	WifiIcon,
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
	isReadOnlyUser,
} from "@/lib/utils"
import { pb } from "@/lib/stores"
import { Trans, useLingui } from "@lingui/react/macro"
import { useMemo, useRef, useState } from "react"
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
	return [
		{
			size: 200,
			minSize: 0,
			accessorKey: "name",
			id: "system",
			name: () => t`System`,
			filterFn: (() => {
				let filterInput = ""
				let filterInputLower = ""
				const nameCache = new Map<string, string>()
				const statusTranslations = {
					up: t`Up`.toLowerCase(),
					down: t`Down`.toLowerCase(),
					paused: t`Paused`.toLowerCase(),
				} as const

				// match filter value against name or translated status
				return (row, _, newFilterInput) => {
					const { name, status } = row.original
					if (newFilterInput !== filterInput) {
						filterInput = newFilterInput
						filterInputLower = newFilterInput.toLowerCase()
					}
					let nameLower = nameCache.get(name)
					if (nameLower === undefined) {
						nameLower = name.toLowerCase()
						nameCache.set(name, nameLower)
					}
					if (nameLower.includes(filterInputLower)) {
						return true
					}
					const statusLower = statusTranslations[status as keyof typeof statusTranslations]
					return statusLower?.includes(filterInputLower) || false
				}
			})(),
			enableHiding: false,
			invertSorting: false,
			Icon: ServerIcon,
			cell: (info) => (
				<span className="flex gap-2 items-center font-medium text-sm text-nowrap md:ps-1 md:pe-5">
					<IndicatorDot system={info.row.original} />
					{info.getValue() as string}
				</span>
			),
			header: sortableHeader,
		},

		{
			accessorFn: ({ info }) => info.ap,
			id: "ap",
			name: () => t`ICMP`,
			size: 50,
			Icon: ClockArrowUp,
						header: (context) => (
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
			cell({ getValue }) {
				const ap = getValue() as number
				return (
					<span className="tabular-nums">
						{ap.toFixed(1)} ms
					</span>
				)
			},
		},
		{
			accessorFn: ({ info }) => info.ad,
			id: "ad",
			name: () => t`DNS`,
			size: 50,
			Icon: ServerIcon,
			header: (context) => (
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
			cell({ getValue }) {
				const ad = getValue() as number
				if (!ad || ad === 0) {
					return null
				}
				return (
					<span className="tabular-nums">
						{ad.toFixed(1)} ms
					</span>
				)
			},
		},
		{
			accessorFn: ({ info }) => info.v,
			id: "agent",
			name: () => t`Agent`,
			// invertSorting: true,
			size: 50,
			Icon: WifiIcon,
			hideSort: true,
			header: sortableHeader,
			cell(info) {
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
			name: () => t({ message: "Actions", comment: "Table column" }),
			size: 50,
			cell: ({ row }) => (
				<div className="flex justify-end items-center gap-1 -ms-3">
											<SystemConfigDialog system={row.original} />
					<AlertButton system={row.original} />
					<ActionsButton system={row.original} />
				</div>
			),
		},
	] as ColumnDef<SystemRecord>[]
}

function sortableHeader(context: HeaderContext<SystemRecord, unknown>) {
	const { column } = context
	// @ts-ignore
	const { Icon, hideSort, name }: { Icon: React.ElementType; name: () => string; hideSort: boolean } = column.columnDef
	return (
		<Button
			variant="ghost"
			className="h-9 px-3 flex"
			onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
		>
			{Icon && <Icon className="me-2 size-4" />}
			{name()}
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
	let editOpened = useRef(false)
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
						{!isReadOnlyUser() && (
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
						<DropdownMenuItem
							className={cn(isReadOnlyUser() && "hidden")}
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
						<DropdownMenuItem onClick={() => copyToClipboard(name)}>
							<CopyIcon className="me-2.5 size-4" />
							<Trans>Copy name</Trans>
						</DropdownMenuItem>
						<DropdownMenuItem onClick={() => copyToClipboard(host)}>
							<CopyIcon className="me-2.5 size-4" />
							<Trans>Copy host</Trans>
						</DropdownMenuItem>
						<DropdownMenuSeparator className={cn(isReadOnlyUser() && "hidden")} />
						<DropdownMenuItem className={cn(isReadOnlyUser() && "hidden")} onSelect={() => setDeleteOpen(true)}>
							<Trash2Icon className="me-2.5 size-4" />
							<Trans>Delete</Trans>
						</DropdownMenuItem>
					</DropdownMenuContent>
				</DropdownMenu>
				{/* edit dialog */}
				<Dialog open={editOpen} onOpenChange={setEditOpen}>
					{editOpened.current && <SystemDialog system={system} setOpen={setEditOpen} />}
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
	}, [id, status, host, name, t, deleteOpen, editOpen])
})
