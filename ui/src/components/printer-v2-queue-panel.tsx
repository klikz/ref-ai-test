import { Backend_Request } from "@/services/backend"
import { useCallback, useEffect, useState } from "react"
import { Loader2, RefreshCcw } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Panel } from "@/components/layout/panel"
import { ShowErrorToast } from "@/components/showToast"

type PrinterQueueJob = {
  id: number
  document_name: string
  user_name: string
  submitted_time: string
  job_status: string
  total_pages: number
  size: number
}

type PrinterQueueResponse = {
  printer_v2_id: number
  printer_name: string
  jobs: PrinterQueueJob[]
}

type PrinterV2QueuePanelProps = {
  printerV2Id: number | null
  refreshKey?: number
  pollIntervalMs?: number
}

export function PrinterV2QueuePanel({
  printerV2Id,
  refreshKey = 0,
  pollIntervalMs = 3000,
}: PrinterV2QueuePanelProps) {
  const [jobs, setJobs] = useState<PrinterQueueJob[]>([])
  const [printerName, setPrinterName] = useState("")
  const [loading, setLoading] = useState(false)
  const [polling, setPolling] = useState(false)

  const loadQueue = useCallback(
    async (options?: { silent?: boolean }) => {
      const silent = options?.silent ?? false
      if (!printerV2Id) {
        setJobs([])
        setPrinterName("")
        return
      }

      if (silent) {
        setPolling(true)
      } else {
        setLoading(true)
      }

      try {
        const result = await Backend_Request<PrinterQueueResponse>(
          { printer_v2_id: printerV2Id },
          "/api/tech/printers-v2/jobs",
        )
        if (result.result === "ok" && result.data) {
          setPrinterName(result.data.printer_name)
          setJobs(result.data.jobs ?? [])
        } else if (!silent) {
          ShowErrorToast(result.error || "Print queue yuklanmadi")
        }
      } finally {
        if (silent) {
          setPolling(false)
        } else {
          setLoading(false)
        }
      }
    },
    [printerV2Id],
  )

  useEffect(() => {
    void loadQueue()
  }, [loadQueue, refreshKey])

  useEffect(() => {
    if (!printerV2Id || pollIntervalMs <= 0) {
      return
    }

    const interval = window.setInterval(() => {
      void loadQueue({ silent: true })
    }, pollIntervalMs)

    return () => window.clearInterval(interval)
  }, [printerV2Id, pollIntervalMs, loadQueue, refreshKey])

  return (
    <Panel
      title={printerName ? `Aktiv queue: ${printerName}` : "Aktiv queue"}
      noPadding
      action={
        <Button
          variant="outline"
          size="sm"
          className="gap-2"
          disabled={!printerV2Id || loading}
          onClick={() => void loadQueue()}
        >
          {loading || polling ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <RefreshCcw className="size-4" />
          )}
          Yangilash
        </Button>
      }
    >
      <div className="overflow-x-auto rounded-b-xl">
        <Table>
          <TableHeader>
            <TableRow className="border-b border-border/60 bg-muted/40 hover:bg-muted/40">
              <TableHead className="h-10 whitespace-nowrap px-3">ID</TableHead>
              <TableHead className="h-10 whitespace-nowrap px-3">Document</TableHead>
              <TableHead className="h-10 whitespace-nowrap px-3">User</TableHead>
              <TableHead className="h-10 whitespace-nowrap px-3">Status</TableHead>
              <TableHead className="h-10 whitespace-nowrap px-3">Submitted</TableHead>
              <TableHead className="h-10 whitespace-nowrap px-3">Pages</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {!printerV2Id ? (
              <TableRow>
                <TableCell colSpan={6} className="py-6 text-center text-muted-foreground">
                  Printer tanlanmagan
                </TableCell>
              </TableRow>
            ) : loading && jobs.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="py-6 text-center text-muted-foreground">
                  <Loader2 className="mx-auto size-5 animate-spin" />
                </TableCell>
              </TableRow>
            ) : jobs.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="py-6 text-center text-muted-foreground">
                  Queue bo&apos;sh
                </TableCell>
              </TableRow>
            ) : (
              jobs.map((job) => (
                <TableRow key={job.id}>
                  <TableCell>{job.id}</TableCell>
                  <TableCell>{job.document_name || "—"}</TableCell>
                  <TableCell>{job.user_name || "—"}</TableCell>
                  <TableCell>{job.job_status || "—"}</TableCell>
                  <TableCell>{job.submitted_time || "—"}</TableCell>
                  <TableCell>{job.total_pages || 0}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </Panel>
  )
}
