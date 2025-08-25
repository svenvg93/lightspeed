import { Area, CartesianGrid, YAxis, ComposedChart } from "recharts"
import { ChartContainer, ChartTooltip, ChartTooltipContent, ChartLegend, ChartLegendContent, xAxis } from "@/components/ui/chart"
import { useYAxisWidth, cn, formatShortDate, chartMargin, toFixedFloat, decimalString } from "@/lib/utils"
import { ChartData } from "@/types"
import { useMemo } from "react"
import { t } from "@lingui/core/macro"
import { Trans } from "@lingui/react/macro"

interface SpeedtestPacketLossChartProps {
	chartData: ChartData
	serverIds: string[]
	serverNames?: Record<string, string> // Map of server ID to friendly name
}

export default function SpeedtestPacketLossChart({ chartData, serverIds, serverNames }: SpeedtestPacketLossChartProps) {
	const { yAxisWidth, updateYAxisWidth } = useYAxisWidth()
	
	// Use all speedtest data including gaps
	const speedtestData = chartData.speedtestData || []

	if (speedtestData.length === 0) {
		return (
			<div className="flex items-center justify-center h-full text-muted-foreground">
				<Trans>No speedtest data available</Trans>
			</div>
		)
	}

	// Define colors for different servers
	const serverColors = [
		"hsl(var(--chart-1))",
		"hsl(var(--chart-2))", 
		"hsl(var(--chart-3))",
		"hsl(var(--chart-4))",
		"hsl(var(--chart-5))",
	]

	return useMemo(() => {
		return (
			<div>
				<ChartContainer
					className={cn("h-full w-full absolute aspect-auto bg-card opacity-0 transition-opacity", {
						"opacity-100": yAxisWidth,
					})}
				>
					<ComposedChart accessibilityLayer data={speedtestData} margin={chartMargin}>
						<CartesianGrid vertical={false} />
						{/* Y-axis for Packet Loss */}
						<YAxis
							yAxisId="left"
							orientation="left"
							className="tracking-tighter"
							width={yAxisWidth}
							tickFormatter={(value) => updateYAxisWidth(toFixedFloat(value, 1) + "%")}
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
									contentFormatter={({ value }) => {
										return decimalString(value, 1) + "%"
									}}
								/>
							}
						/>
						{/* Gradient definitions for each server */}
						<defs>
							{serverIds.map((serverId, index) => (
								<linearGradient 
									key={serverId}
									id={`fillPacketLoss-${serverId}`} 
									x1="0" 
									y1="0" 
									x2="0" 
									y2="1"
								>
									<stop
										offset="5%"
										stopColor={serverColors[index % serverColors.length]}
										stopOpacity={0.8}
									/>
									<stop
										offset="95%"
										stopColor={serverColors[index % serverColors.length]}
										stopOpacity={0.1}
									/>
								</linearGradient>
							))}
						</defs>
						{/* Packet Loss Areas for each server */}
						{serverIds.map((serverId, index) => (
							<Area
								key={serverId}
								yAxisId="left"
								dataKey={(data) => data[serverId]?.packet_loss ?? null}
								name={serverNames?.[serverId] || serverId}
								type="monotoneX"
								fill={`url(#fillPacketLoss-${serverId})`}
								fillOpacity={0.4}
								stroke={serverColors[index % serverColors.length]}
								strokeWidth={2}
								isAnimationActive={false}
							/>
						))}
						{/* Legend */}
						<ChartLegend
							content={<ChartLegendContent />}
							verticalAlign="bottom"
						/>
					</ComposedChart>
				</ChartContainer>
			</div>
		)
	}, [speedtestData, yAxisWidth, serverIds, serverNames])
}
