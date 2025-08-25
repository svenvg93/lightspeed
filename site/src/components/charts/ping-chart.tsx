import { Area, CartesianGrid, YAxis, Line, ComposedChart } from "recharts"
import { ChartContainer, ChartTooltip, ChartTooltipContent, ChartLegend, ChartLegendContent, xAxis } from "@/components/ui/chart"
import { useYAxisWidth, cn, formatShortDate, chartMargin, toFixedFloat, decimalString } from "@/lib/utils"
import { ChartData } from "@/types"
import { useMemo } from "react"
import { t } from "@lingui/core/macro"
import { Trans } from "@lingui/react/macro"

interface PingChartProps {
	chartData: ChartData
	host: string
}

export default function PingChart({ chartData, host }: PingChartProps) {
	const { yAxisWidth, updateYAxisWidth } = useYAxisWidth()
	
	// Filter ping data for this specific host and handle gaps
	const pingData = chartData.pingData?.filter(data => data && (data.created === null || data[host])) || []

	if (pingData.length === 0) {
		return (
			<div className="flex items-center justify-center h-full text-muted-foreground">
				<Trans>No ping data available for {host}</Trans>
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
					<ComposedChart accessibilityLayer data={pingData} margin={chartMargin}>
						<CartesianGrid vertical={false} />
						{/* Left Y-axis for RTT */}
						<YAxis
							yAxisId="left"
							orientation="left"
							className="tracking-tighter"
							width={yAxisWidth}
							tickFormatter={(value) => updateYAxisWidth(toFixedFloat(value, 1) + " ms")}
							tickLine={false}
							axisLine={false}
						/>
						{/* Right Y-axis for packet loss */}
						<YAxis
							yAxisId="right"
							orientation="right"
							className="tracking-tighter"
							width={yAxisWidth}
							tickFormatter={(value) => toFixedFloat(value, 1) + " %"}
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
										if (name === t`Packet Loss`) {
											return toFixedFloat(value, 1) + " %"
										}
										return decimalString(value, 1) + " ms"
									}}
								/>
							}
						/>
						{/* Gradient definitions */}
						<defs>
							<linearGradient id={`fillAvgRtt-${host}`} x1="0" y1="0" x2="0" y2="1">
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
							<linearGradient id={`fillMinRtt-${host}`} x1="0" y1="0" x2="0" y2="1">
								<stop
									offset="5%"
									stopColor="hsl(var(--chart-2))"
									stopOpacity={0.6}
								/>
								<stop
									offset="95%"
									stopColor="hsl(var(--chart-2))"
									stopOpacity={0.2}
								/>
							</linearGradient>
							<linearGradient id={`fillMaxRtt-${host}`} x1="0" y1="0" x2="0" y2="1">
								<stop
									offset="5%"
									stopColor="hsl(var(--chart-3))"
									stopOpacity={0.6}
								/>
								<stop
									offset="95%"
									stopColor="hsl(var(--chart-3))"
									stopOpacity={0.2}
								/>
							</linearGradient>
						</defs>
						{/* RTT Areas with gradients */}
						<Area
							yAxisId="left"
							dataKey={(data) => data[host]?.avg_rtt ?? null}
							name={t`Average RTT`}
							type="monotoneX"
							fill={`url(#fillAvgRtt-${host})`}
							fillOpacity={0.4}
							stroke="hsl(var(--chart-1))"
							isAnimationActive={false}
						/>
						<Area
							yAxisId="left"
							dataKey={(data) => data[host]?.min_rtt ?? null}
							name={t`Min RTT`}
							type="monotoneX"
							fill={`url(#fillMinRtt-${host})`}
							fillOpacity={0.3}
							stroke="hsl(var(--chart-2))"
							isAnimationActive={false}
						/>
						<Area
							yAxisId="left"
							dataKey={(data) => data[host]?.max_rtt ?? null}
							name={t`Max RTT`}
							type="monotoneX"
							fill={`url(#fillMaxRtt-${host})`}
							fillOpacity={0.3}
							stroke="hsl(var(--chart-3))"
							isAnimationActive={false}
						/>
						{/* Packet Loss Line */}
						<Line
							yAxisId="right"
							dataKey={(data) => data[host]?.packet_loss ?? null}
							name={t`Packet Loss`}
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
	}, [pingData, yAxisWidth, host])
}
