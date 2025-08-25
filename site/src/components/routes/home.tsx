import { Suspense, lazy, memo, useEffect, useMemo } from "react"
import { $systems, pb } from "@/lib/stores"
import { useStore } from "@nanostores/react"
import { GithubIcon } from "lucide-react"
import { Separator } from "../ui/separator"
import { updateRecordList, updateSystemList } from "@/lib/utils"
import { SystemRecord } from "@/types"
import { useLingui } from "@lingui/react/macro"

const SystemsTable = lazy(() => import("../systems-table/systems-table"))

export const Home = memo(() => {
	const systems = useStore($systems)
	const { t } = useLingui()

	useEffect(() => {
		document.title = t`Dashboard` + " / Beszel"
	}, [t])

	useEffect(() => {
		// make sure we have the latest list of systems
		updateSystemList()

		// subscribe to real time updates for systems / alerts / system_averages
		pb.collection<SystemRecord>("systems").subscribe("*", (e) => {
			updateRecordList(e, $systems)
		})
		pb.collection("system_averages").subscribe("*", () => {
			// When system_averages are updated, refresh the system list to get latest averages
			updateSystemList()
		})
		return () => {
			pb.collection("systems").unsubscribe("*")
			pb.collection("system_averages").unsubscribe("*")
			// pb.collection('alerts').unsubscribe('*')
		}
	}, [])

	return useMemo(
		() => (
			<>
				<Suspense>
					<SystemsTable />
				</Suspense>

				<div className="flex gap-1.5 justify-end items-center pe-3 sm:pe-6 mt-3.5 text-xs opacity-80">
					<a
						href="https://github.com/henrygd/beszel"
						target="_blank"
						className="flex items-center gap-0.5 text-muted-foreground hover:text-foreground duration-75"
					>
						<GithubIcon className="h-3 w-3" /> GitHub
					</a>
					<Separator orientation="vertical" className="h-2.5 bg-muted-foreground opacity-70" />
					<a
						href="https://github.com/henrygd/beszel/releases"
						target="_blank"
						className="text-muted-foreground hover:text-foreground duration-75"
					>
						Beszel {globalThis.BESZEL.HUB_VERSION}
					</a>
				</div>
			</>
		),
		[]
	)
})

