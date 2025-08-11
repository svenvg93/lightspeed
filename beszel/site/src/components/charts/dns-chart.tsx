import { Area, CartesianGrid, YAxis, Line, ComposedChart } from "recharts"
import { ChartContainer, ChartTooltip, ChartTooltipContent, ChartLegend, ChartLegendContent, xAxis } from "@/components/ui/chart"
import { useYAxisWidth, cn, formatShortDate, chartMargin, toFixedFloat, decimalString } from "@/lib/utils"
import { ChartData } from "@/types"
import { useMemo } from "react"
import { t } from "@lingui/core/macro"
import { Trans } from "@lingui/react/macro"

interface DnsChartProps {
	chartData: ChartData
	targetKey: string // DNS target key (domain@server#type)
}

export default function DnsChart({ chartData, targetKey }: DnsChartProps) {
	const { yAxisWidth, updateYAxisWidth } = useYAxisWidth()
	
	// Use all DNS data including gaps
	const dnsData = chartData.dnsData || []

	if (dnsData.length === 0) {
		return (
			<div className="flex items-center justify-center h-full text-muted-foreground">
				<Trans>No DNS data available for {targetKey}</Trans>
			</div>
		)
	}

	return useMemo(() => {
		return (
			<div>
				<ChartContainer
					className={cn("h-full w-full absolute aspect-auto bg-card opacity-0 transition-opacity", {
						"opacity-100": yAxisWidth,
					})}
				>
					<ComposedChart accessibilityLayer data={dnsData} margin={chartMargin}>
						<CartesianGrid vertical={false} />
						{/* Left Y-axis for Lookup Time */}
						<YAxis
							yAxisId="left"
							orientation="left"
							className="tracking-tighter"
							width={yAxisWidth}
							tickFormatter={(value) => updateYAxisWidth(toFixedFloat(value, 1) + "ms")}
							tickLine={false}
							axisLine={false}
						/>
						{/* Right Y-axis for Error Codes */}
						<YAxis
							yAxisId="right"
							orientation="right"
							className="tracking-tighter"
							width={yAxisWidth}
							tickFormatter={(value) => value.toString()}
							tickLine={false}
							axisLine={false}
						/>
						{xAxis(chartData)}
						<ChartTooltip
							animationEasing="ease-out"
							animationDuration={150}
							content={
								<ChartTooltipContent
									labelFormatter={(_, data) => formatShortDate(data[0].payload.created)}
									contentFormatter={({ value, name }) => {
										if (name === t`Error Code`) {
											return value.toString()
										}
										return decimalString(value, 1) + "ms"
									}}
								/>
							}
						/>
						{/* Gradient definitions */}
						<defs>
							<linearGradient id={`fillLookupTime-${targetKey}`} x1="0" y1="0" x2="0" y2="1">
								<stop
									offset="5%"
									stopColor="hsl(var(--chart-1))"
									stopOpacity={0.8}
								/>
								<stop
									offset="95%"
									stopColor="hsl(var(--chart-1))"
									stopOpacity={0.1}
								/>
							</linearGradient>
						</defs>
						{/* Lookup Time Area */}
						<Area
							yAxisId="left"
							dataKey={(data) => data[targetKey]?.lookup_time ?? null}
							name={t`Lookup Time`}
							type="monotoneX"
							fill={`url(#fillLookupTime-${targetKey})`}
							fillOpacity={0.4}
							stroke="hsl(var(--chart-1))"
							strokeWidth={2}
							isAnimationActive={false}
						/>
						{/* Error Code Line (will show error codes as numeric values) */}
						<Line
							yAxisId="right"
							dataKey={(data) => {
								const dnsResult = data[targetKey]
								if (!dnsResult) return null
								// Convert error codes to numeric values for display
								const errorCode = dnsResult.error_code
								if (errorCode === "SERVFAIL") return 1
								if (errorCode === "NXDOMAIN") return 2
								if (errorCode === "REFUSED") return 3
								if (errorCode === "TIMEOUT") return 4
								if (errorCode !== "") return 5 // Other errors
								return 0 // Success
							}}
							name={t`Error Code`}
							type="monotoneX"
							stroke="hsl(0 84% 60%)"
							strokeWidth={2}
							dot={false}
							isAnimationActive={false}
						/>
						{/* Legend */}
						<ChartLegend
							content={<ChartLegendContent />}
							verticalAlign="bottom"
						/>
					</ComposedChart>
				</ChartContainer>
			</div>
		)
	}, [dnsData, yAxisWidth, targetKey])
}
