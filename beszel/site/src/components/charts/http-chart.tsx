import { Area, CartesianGrid, YAxis, Line, ComposedChart } from "recharts"
import { ChartContainer, ChartTooltip, ChartTooltipContent, ChartLegend, ChartLegendContent, xAxis } from "@/components/ui/chart"
import { useYAxisWidth, cn, formatShortDate, chartMargin, toFixedFloat, decimalString } from "@/lib/utils"
import { ChartData } from "@/types"
import { useMemo } from "react"
import { t } from "@lingui/core/macro"
import { Trans } from "@lingui/react/macro"

interface HttpChartProps {
	chartData: ChartData
	targetKey: string // HTTP target key (URL)
}

export default function HttpChart({ chartData, targetKey }: HttpChartProps) {
	const { yAxisWidth, updateYAxisWidth } = useYAxisWidth()
	
	// Use all HTTP data including gaps
	const httpData = chartData.httpData || []

	if (httpData.length === 0) {
		return (
			<div className="flex items-center justify-center h-full text-muted-foreground">
				<Trans>No HTTP data available for {targetKey}</Trans>
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
					<ComposedChart accessibilityLayer data={httpData} margin={chartMargin}>
						<CartesianGrid vertical={false} />
						{/* Left Y-axis for Response Time */}
						<YAxis
							yAxisId="left"
							orientation="left"
							className="tracking-tighter"
							width={yAxisWidth}
							tickFormatter={(value) => updateYAxisWidth(toFixedFloat(value, 1) + "ms")}
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
								return decimalString(value, 1) + "ms"
							}}
								/>
							}
						/>
						{/* Gradient definitions */}
						<defs>
							<linearGradient id={`fillResponseTime-${targetKey}`} x1="0" y1="0" x2="0" y2="1">
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
						{/* Response Time Area */}
						<Area
							yAxisId="left"
							dataKey={(data) => data[targetKey]?.response_time ?? null}
							name={t`Response Time`}
							type="monotoneX"
							fill={`url(#fillResponseTime-${targetKey})`}
							fillOpacity={0.4}
							stroke="hsl(var(--chart-1))"
							strokeWidth={2}
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
	}, [httpData, yAxisWidth, targetKey])
}
