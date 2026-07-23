import { useCallback, useRef, useState } from "react"
import { Barcode, FlaskConical, Loader2, Search } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ShowErrorToast } from "@/components/showToast"

type LabBxRow = {
  serial?: string
  compressor?: string
  bx_model?: string
  line_num?: string
  point_num?: string
  start_time?: string
  stop_time?: string
  test_time?: string
  bx_result?: string
}

type LabInfoResponse = {
  serial?: string
  rows?: LabBxRow[]
}

export default function LabPage() {
  const inputRef = useRef<HTMLInputElement>(null)
  const [serial, setSerial] = useState("")
  const [loading, setLoading] = useState(false)
  const [searched, setSearched] = useState(false)
  const [rows, setRows] = useState<LabBxRow[]>([])
  const [queriedSerial, setQueriedSerial] = useState("")

  const search = useCallback(
    async (value?: string) => {
      const q = (value ?? serial).trim()
      if (!q) {
        ShowErrorToast("Serial kiriting")
        inputRef.current?.focus()
        return
      }

      setLoading(true)
      setSearched(true)
      const result = await Backend_Request<LabInfoResponse>({ serial: q }, "/api/lab/info")
      setLoading(false)

      if (result.result !== "ok") {
        setRows([])
        setQueriedSerial(q)
        ShowErrorToast(result.error || "Laboratoriya serveriga ulanib bo'lmadi")
        return
      }

      setQueriedSerial(result.data?.serial || q)
      setSerial(q)
      setRows(result.data?.rows ?? [])
    },
    [serial],
  )

  function onSubmit(event: React.FormEvent) {
    event.preventDefault()
    void search()
  }

  return (
    <PageContainer
      title="Laboratoriya"
      description="VTM laboratoriyasi — serial bo'yicha BxData test natijalari"
    >
      <Panel className="overflow-hidden border-primary/20 bg-gradient-to-br from-primary/5 via-card to-card p-0">
        <form onSubmit={onSubmit} className="flex flex-col gap-4 p-5 sm:flex-row sm:items-end">
          <div className="flex-1 space-y-2">
            <label
              htmlFor="lab-serial-search"
              className="flex items-center gap-2 text-sm font-medium text-foreground"
            >
              <Barcode className="size-4 text-primary" />
              Serial raqam (BxNo)
            </label>
            <Input
              id="lab-serial-search"
              ref={inputRef}
              value={serial}
              onChange={(event) => setSerial(event.target.value)}
              placeholder="Serial skanerlang yoki kiriting"
              className="h-12 rounded-xl border-border/70 bg-background/80 text-base font-mono shadow-inner"
              autoComplete="off"
              spellCheck={false}
            />
          </div>
          <Button
            type="submit"
            size="lg"
            className="h-12 rounded-xl px-6 shadow-md shadow-primary/20"
            disabled={loading}
          >
            {loading ? <Loader2 className="size-4 animate-spin" /> : <Search className="size-4" />}
            Qidirish
          </Button>
        </form>
      </Panel>

      {loading ? (
        <div className="flex items-center justify-center gap-2 rounded-2xl border border-dashed border-border/70 bg-muted/20 py-16 text-muted-foreground">
          <Loader2 className="size-5 animate-spin" />
          Ma'lumot yuklanmoqda…
        </div>
      ) : null}

      {!loading && searched && rows.length === 0 ? (
        <Panel>
          <div className="flex flex-col items-center justify-center gap-2 py-14 text-center text-muted-foreground">
            <FlaskConical className="size-8 opacity-50" />
            <div className="font-medium text-foreground">Natija topilmadi</div>
            <div className="text-sm">
              <span className="font-mono">{queriedSerial || "—"}</span> uchun BxData yozuvi yo‘q
            </div>
          </div>
        </Panel>
      ) : null}

      {!loading && rows.length > 0 ? (
        <Panel
          title={`Natijalar: ${queriedSerial}`}
          description={`${rows.length} ta yozuv`}
          noPadding
        >
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Serial</TableHead>
                  <TableHead>Compressor</TableHead>
                  <TableHead>BxModel</TableHead>
                  <TableHead>Line</TableHead>
                  <TableHead>Point</TableHead>
                  <TableHead>Start</TableHead>
                  <TableHead>Stop</TableHead>
                  <TableHead>TestTime</TableHead>
                  <TableHead>Result</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((row, index) => (
                  <TableRow key={`${row.serial}-${row.stop_time}-${row.point_num}-${index}`}>
                    <TableCell className="font-mono text-xs">{row.serial || "—"}</TableCell>
                    <TableCell className="font-mono text-xs">{row.compressor || "—"}</TableCell>
                    <TableCell>{row.bx_model || "—"}</TableCell>
                    <TableCell>{row.line_num || "—"}</TableCell>
                    <TableCell>{row.point_num || "—"}</TableCell>
                    <TableCell className="whitespace-nowrap text-xs">{row.start_time || "—"}</TableCell>
                    <TableCell className="whitespace-nowrap text-xs">{row.stop_time || "—"}</TableCell>
                    <TableCell>{row.test_time || "—"}</TableCell>
                    <TableCell className="font-medium">{row.bx_result || "—"}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </Panel>
      ) : null}
    </PageContainer>
  )
}
