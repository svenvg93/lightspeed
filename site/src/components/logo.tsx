export function Logo({ className }: { className?: string }) {
	return (
		<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 60" className={className}>
			<defs>
				<linearGradient id="apexGradient" x1="0%" y1="0%" x2="100%" y2="100%">
					<stop offset="0%" style={{ stopColor: "#3b82f6" }} />
					<stop offset="100%" style={{ stopColor: "#1e40af" }} />
				</linearGradient>
			</defs>
			{/* Triangle icon representing "peak/apex" */}
			<path
				fill="url(#apexGradient)"
				d="M8 45L20 15L32 45H8Z"
				strokeWidth="2"
				stroke="currentColor"
				opacity="0.9"
			/>
			{/* A */}
			<path
				fill="currentColor"
				d="M50 45V44L62 15h6l12 29v1h-5l-2.5-6h-13l-2.5 6h-5zm8.5-10h9l-4.5-11-4.5 11z"
			/>
			{/* P */}
			<path
				fill="currentColor"
				d="M85 45V15h12c3 0 5.5 1 7 2.5s2.5 3.5 2.5 6c0 2.5-.8 4.5-2.5 6s-4 2.5-7 2.5h-7V45h-5zm5-14h7c1.5 0 2.7-.4 3.5-1.2s1.2-1.8 1.2-3.3c0-1.5-.4-2.5-1.2-3.3s-2-1.2-3.5-1.2h-7v9z"
			/>
			{/* E */}
			<path
				fill="currentColor"
				d="M115 45V15h18v4h-13v8h12v4h-12v10h13v4h-18z"
			/>
			{/* X */}
			<path
				fill="currentColor"
				d="M145 45L155 30L145 15h6l7 11 7-11h6l-10 15 10 15h-6l-7-11-7 11h-6z"
			/>
		</svg>
	)
}
