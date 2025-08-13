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
	info: SystemInfo
	averages?: {
		ap?: number   // Average ping latency
		apl?: number  // Average ping packet loss
		ad?: number   // Average DNS lookup time
		adf?: number  // Average DNS failure rate
		ah?: number   // Average HTTP response time
		ahf?: number  // Average HTTP failure rate
		adl?: number  // Average download
		aul?: number  // Average upload
	}
	expected_performance?: {
		ping_latency?: number      // Expected ping latency in ms
		dns_lookup_time?: number   // Expected DNS lookup time in ms
		http_response_time?: number // Expected HTTP response time in ms
		download_speed?: number    // Expected download speed in Mbps
		upload_speed?: number      // Expected upload speed in Mbps
	}
	tags?: string[]  // Array of tags for filtering and organization
	v: string
	
	// Unified monitoring configuration
	monitoring_config?: {
		enabled: {
			ping: boolean
			dns: boolean
			http?: boolean
			speedtest?: boolean
		}
		global_interval?: string | number // Default interval for all monitoring types
		ping?: {
			targets: {
				host: string
				friendly_name?: string
				count: number
				timeout: number
			}[]
			interval?: string | number // Override global interval
			expected_latency?: number // Expected ping latency in ms
		}
		dns?: {
			targets: {
				domain: string
				server: string
				type: string
				timeout: number
				friendly_name?: string
				protocol?: "udp" | "tcp" | "doh" | "dot"
			}[]
			interval?: string | number // Override global interval
			expected_lookup_time?: number // Expected DNS lookup time in ms
		}
		http?: {
			targets: {
				url: string
				friendly_name?: string
				method?: "GET" | "POST" | "PUT" | "DELETE" | "HEAD"
				timeout: number
				expected_status?: number[]
				headers?: Record<string, string>
			}[]
			interval?: string | number // Override global interval
			expected_response_time?: number // Expected HTTP response time in ms
		}
		speedtest?: {
			targets: {
				server_url: string
				friendly_name?: string
				timeout: number
			}[]
			interval?: string | number // Override global interval
			expected_download_speed?: number // Expected download speed in Mbps
			expected_upload_speed?: number   // Expected upload speed in Mbps
		}
	}
}

export interface SystemInfo {
	/** hostname */
	h: string
	/** agent version */
	v: string

	/** public ip address */
	ip?: string
	/** internet service provider */
	isp?: string
	/** autonomous system number */
	asn?: string
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
		format: (timestamp: string | number) => string
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

export interface DnsStatsRecord extends RecordModel {
	system: string
	domain: string
	server: string
	type: string
	status: string
	lookup_time: number
	error_code: string
	created: string | number
}

export interface HttpStatsRecord extends RecordModel {
	system: string
	url: string
	status: string
	response_time: number
	status_code: number
	error_code: string
	created: string | number
}

export interface SpeedtestStatsRecord extends RecordModel {
	system: string
	server_id: string
	server_name: string
	server_location: string
	server_country: string
	status: string
	download_speed: number
	upload_speed: number
	latency: number
	packet_loss: number
	error_code: string
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

type ChartDataDns = {
	created: number | null
} & {
	[key: string]: key extends "created" ? never : {
		domain: string
		server: string
		type: string
		status: string
		lookup_time: number
		error_code: string
	} | null // Allow null for gap data points
}

type ChartDataHttp = {
	created: number | null
} & {
	[key: string]: key extends "created" ? never : {
		url: string
		status: string
		response_time: number
		status_code: number
		error_code: string
	} | null // Allow null for gap data points
}

type ChartDataSpeedtest = {
	created: number | null
} & {
	[key: string]: key extends "created" ? never : {
		server_id: string
		status: string
		download_speed: number
		upload_speed: number
		latency: number
		packet_loss: number
		error_code: string
	} | null // Allow null for gap data points
}

export interface ChartData {
	agentVersion: SemVer
	systemStats: SystemStatsRecord[]
	containerData: ChartDataContainer[]
	pingData?: ChartDataPing[] // Made optional since it's only used for ping charts
	dnsData?: ChartDataDns[] // Made optional since it's only used for DNS charts
	httpData?: ChartDataHttp[] // Made optional since it's only used for HTTP charts
	speedtestData?: ChartDataSpeedtest[] // Made optional since it's only used for speedtest charts
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
