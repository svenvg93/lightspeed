import { RecordModel } from "pocketbase"
import { Unit, Os } from "./lib/enums"

// global window properties
declare global {
	var BESZEL: {
		BASE_PATH: string
		HUB_VERSION: string
		HUB_URL: string
	}
}

export interface FingerprintRecord extends RecordModel {
	id: string
	system: string
	fingerprint: string
	token: string
	expand: {
		system: {
			name: string
		}
	}
}

export interface SystemRecord extends RecordModel {
	name: string
	host: string
	status: "up" | "down" | "paused" | "pending"
	port: string
	info: SystemInfo
	v: string
	ping_config?: {
		targets: {
			host: string
			friendly_name?: string
			count: number
			timeout: number
		}[]
		interval: string | number // Cron expression or seconds for all ping tests
	}
}

export interface SystemInfo {
	/** hostname */
	h: string
	/** agent version */
	v: string
	/** network interface speed (mbps) */
	ns?: number
	/** public ip address */
	ip?: string
	/** internet service provider */
	isp?: string
	/** autonomous system number */
	asn?: string
	/** average ping across all targets (ms) */
	ap?: number
}


export interface SystemStatsRecord extends RecordModel {
	system: string
	stats: SystemStats
	created: string | number
}

export interface AlertRecord extends RecordModel {
	id: string
	system: string
	name: string
	triggered: boolean
	sysname?: string
	// user: string
}

export interface AlertsHistoryRecord extends RecordModel {
	alert: string
	user: string
	system: string
	name: string
	val: number
	created: string
	resolved?: string | null
}

export type ChartTimes = "1h" | "12h" | "24h" | "1w" | "30d"

export interface ChartTimeData {
	[key: string]: {
		type: "1m" | "10m" | "20m" | "120m" | "480m"
		expectedInterval: number
		label: () => string
		ticks?: number
		format: (timestamp: string) => string
		getOffset: (endTime: Date) => Date
	}
}

export interface UserSettings {
	chartTime: ChartTimes
	emails?: string[]
	webhooks?: string[]
	unitTemp?: Unit
	unitNet?: Unit
	unitDisk?: Unit
	colorWarn?: number
	colorCrit?: number
}

type ChartDataContainer = {
	created: number | null
} & {
	[key: string]: key extends "created" ? never : ContainerStats
}

export interface SemVer {
	major: number
	minor: number
	patch: number
}

export interface PingStatsRecord extends RecordModel {
	system: string
	host: string
	packet_loss: number
	min_rtt: number
	max_rtt: number
	avg_rtt: number
	created: string | number
}

type ChartDataPing = {
	created: number | null
} & {
	[key: string]: key extends "created" ? never : {
		host: string
		packet_loss: number
		min_rtt: number
		max_rtt: number
		avg_rtt: number
	} | null // Allow null for gap data points
}

export interface ChartData {
	agentVersion: SemVer
	systemStats: SystemStatsRecord[]
	containerData: ChartDataContainer[]
	pingData?: ChartDataPing[] // Made optional since it's only used for ping charts
	orientation: "right" | "left"
	ticks: number[]
	domain: number[]
	chartTime: ChartTimes
}

interface AlertInfo {
	name: () => string
	unit: string
	icon: any
	desc: () => string
	max?: number
	min?: number
	step?: number
	start?: number
	/** Single value description (when there's only one value, like status) */
	singleDesc?: () => string
}
