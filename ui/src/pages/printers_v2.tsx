import { Backend_Request } from "@/services/backend"
import { useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { Eye, Loader2, RefreshCcw } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  createColumnHelper,
} from "@tanstack/react-table"

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { X } from "lucide-react"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"

type LineItem = { line_id: number; name: string }

type LabelTemplateItem = {
  id: number
  name: string
  line_id: number
  line_name?: string
}

type PrinterV2Item = {
  id: number
  line_id: number
  line_name: string
  printer_name: string
  address: string
  label_template_id: number
  label_template_name: string
  print_language: string
  language_hint?: string
}

type DetectLanguageResponse = {
  print_language: string
  driver_name: string
  detect_reason: string
}

const PRINT_LANGUAGE_LABELS: Record<string, string> = {
  gdi: "GDI (Windows)",
  tspl: "TSPL",
  zpl: "ZPL",
}

function printLanguageShortHint(lang: string, printerName?: string): string {
  const name = (printerName || "").toUpperCase()
  if (name.startsWith("AC-FILE")) {
    return "Fayl preview → print_preview/*.png (+ .zpl/.tspl)"
  }
  switch (lang) {
    case "zpl":
      return "Zebra RAW — density/speed shablondan"
    case "tspl":
      return "TSPL RAW — density/speed/gap shablondan"
    default:
      return "Windows Printing Preferences (fallback)"
  }
}

type LocalPrinterItem = {
  name: string
  is_default: boolean
  configured: boolean
  line_name?: string
}

type LocalPrintersResponse = {
  hostname: string
  printers: LocalPrinterItem[]
}

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

export default function PrintersV2Page() {
  const [printers, setPrinters] = useState<PrinterV2Item[]>([])
  const [lines, setLines] = useState<LineItem[]>([])
  const [templates, setTemplates] = useState<LabelTemplateItem[]>([])
  const [localPrinters, setLocalPrinters] = useState<LocalPrinterItem[]>([])
  const [loadingLocal, setLoadingLocal] = useState(false)
  const [loadingQueueId, setLoadingQueueId] = useState<number | null>(null)
  const [queuePrinterName, setQueuePrinterName] = useState("")
  const [queueJobs, setQueueJobs] = useState<PrinterQueueJob[]>([])
  const [selectedLine, setSelectedLine] = useState<LineItem | null>(null)
  const [selectedTemplateId, setSelectedTemplateId] = useState("")
  const [printerName, setPrinterName] = useState("")
  const [detectedLanguage, setDetectedLanguage] = useState("gdi")
  const [selectedLanguage, setSelectedLanguage] = useState("gdi")
  const [driverName, setDriverName] = useState("")
  const [detectReason, setDetectReason] = useState("")
  const [detectingLanguage, setDetectingLanguage] = useState(false)
  const [updatingLanguageId, setUpdatingLanguageId] = useState<number | null>(null)
  const [testingPrintId, setTestingPrintId] = useState<number | null>(null)

  const canAdd = Boolean(selectedLine && printerName)

  function showOkToast(text: string) {
    toast(text, {
      style: { backgroundColor: "rgba(8, 113, 8, 0.5)", color: "white" },
      position: "top-right",
    })
  }

  function showErrorToast(text: string) {
    toast(text, {
      style: { backgroundColor: "rgba(31, 41, 55, 0.9)", color: "white", justifyContent: "center" },
      position: "top-center",
    })
  }

  async function printersGetAll() {
    const result = await Backend_Request<PrinterV2Item[]>({}, "/api/tech/printers-v2/all")
    if (result.result === "ok") {
      setPrinters(result.data ?? [])
    } else {
      showErrorToast(result.error || "Printerlar yuklanmadi")
    }
  }

  async function loadLocalPrinters() {
    setLoadingLocal(true)
    const result = await Backend_Request<LocalPrintersResponse>({}, "/api/tech/printers-v2/local")
    setLoadingLocal(false)

    if (result.result === "ok" && result.data) {
      setLocalPrinters(result.data.printers ?? [])
    } else {
      showErrorToast(result.error || "Mahalliy printerlar yuklanmadi")
    }
  }

  async function linesGetAll() {
    const result = await Backend_Request<LineItem[]>({}, "/api/lines/all")
    if (result.result === "ok") {
      setLines(result.data ?? [])
    } else {
      showErrorToast(result.error)
    }
  }

  async function templatesGetAll() {
    const result = await Backend_Request<LabelTemplateItem[]>({}, "/api/tech/label-templates/all")
    if (result.result === "ok") {
      setTemplates(result.data ?? [])
    }
  }

  async function deletePrinter(id: number) {
    const result = await Backend_Request({ id }, "/api/tech/printers-v2/delete")
    if (result.result === "ok") {
      showOkToast("Ma'lumot o'chirildi")
      void printersGetAll()
      void loadLocalPrinters()
    } else {
      showErrorToast(result.error || "O'chirishda xatolik")
    }
  }

  async function detectPrinterLanguage(name: string) {
    if (!name) {
      setDetectedLanguage("gdi")
      setSelectedLanguage("gdi")
      setDriverName("")
      setDetectReason("")
      return
    }

    setDetectingLanguage(true)
    const result = await Backend_Request<DetectLanguageResponse>(
      { printer_name: name },
      "/api/tech/printers-v2/detect-language",
    )
    setDetectingLanguage(false)

    if (result.result === "ok" && result.data) {
      const lang = result.data.print_language || "gdi"
      setDetectedLanguage(lang)
      setSelectedLanguage(lang)
      setDriverName(result.data.driver_name || "")
      setDetectReason(result.data.detect_reason || "")
    } else {
      setDetectedLanguage("gdi")
      setSelectedLanguage("gdi")
      setDriverName("")
      setDetectReason("")
    }
  }

  async function addPrinter() {
    if (!selectedLine || !printerName) {
      return
    }

    const result = await Backend_Request(
      {
        line_id: selectedLine.line_id,
        address: "",
        printer_name: printerName,
        label_template_id: selectedTemplateId ? Number(selectedTemplateId) : 0,
        print_language: selectedLanguage || "gdi",
      },
      "/api/tech/printers-v2/add",
    )

    if (result.result === "ok") {
      showOkToast("Ma'lumot qo'shildi")
      setPrinterName("")
      setDetectedLanguage("gdi")
      setSelectedLanguage("gdi")
      setDriverName("")
      setDetectReason("")
      setSelectedLine(null)
      setSelectedTemplateId("")
      void printersGetAll()
      void loadLocalPrinters()
    } else {
      showErrorToast(result.error || "Qo'shishda xatolik")
    }
  }

  async function updatePrinterLanguage(id: number, printLanguage: string) {
    setUpdatingLanguageId(id)
    const result = await Backend_Request(
      { id, print_language: printLanguage },
      "/api/tech/printers-v2/update-language",
    )
    setUpdatingLanguageId(null)

    if (result.result === "ok") {
      showOkToast("Print tili yangilandi")
      void printersGetAll()
    } else {
      showErrorToast(result.error || "Print tilini yangilab bo'lmadi")
    }
  }

  async function testPrint(printer: PrinterV2Item) {
    setTestingPrintId(printer.id)
    const result = await Backend_Request<{ message?: string }>(
      { id: printer.id },
      "/api/tech/printers-v2/test-print",
    )
    setTestingPrintId(null)

    if (result.result === "ok") {
      showOkToast(result.data?.message || "Test print yuborildi")
    } else {
      showErrorToast(result.error || "Test print xatosi")
    }
  }

  async function loadPrinterQueue(printer: PrinterV2Item) {
    setLoadingQueueId(printer.id)
    const result = await Backend_Request<PrinterQueueResponse>(
      { printer_v2_id: printer.id },
      "/api/tech/printers-v2/jobs",
    )
    setLoadingQueueId(null)

    if (result.result === "ok" && result.data) {
      setQueuePrinterName(result.data.printer_name || printer.printer_name)
      setQueueJobs(result.data.jobs ?? [])
    } else {
      showErrorToast(result.error || "Print queue yuklanmadi")
    }
  }

  const lineTemplates = selectedLine
    ? templates.filter((t) => t.line_id === selectedLine.line_id)
    : templates

  const localNameSet = new Set(localPrinters.map((p) => p.name))

  function dropDownLines() {
    return (
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" className="h-9 w-full justify-between font-normal">
            <span className="truncate">{selectedLine?.name || "Liniyani tanlang"}</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-[var(--radix-dropdown-menu-trigger-width)]" align="start">
          <DropdownMenuGroup>
            {lines.map((line) => (
              <DropdownMenuItem
                key={line.line_id}
                onClick={() => {
                  setSelectedLine(line)
                  setSelectedTemplateId("")
                }}
              >
                {line.name}
              </DropdownMenuItem>
            ))}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    )
  }

  const columnHelper = createColumnHelper<PrinterV2Item>()

  const columns = [
    columnHelper.accessor("line_name", { header: "Liniya nomi" }),
    columnHelper.accessor("printer_name", { header: "Printer nomi" }),
    columnHelper.display({
      id: "installed",
      header: "Kompyuterda",
      cell: ({ row }) =>
        localNameSet.has(row.original.printer_name) ? (
          <span className="text-sm font-medium text-green-600">Ha</span>
        ) : (
          <span className="text-sm text-muted-foreground">Yo&apos;q</span>
        ),
    }),
    columnHelper.accessor("label_template_name", {
      header: "Etiketka shablon",
      cell: ({ getValue }) => getValue() || "—",
    }),
    columnHelper.accessor("print_language", {
      header: "Print tili",
      cell: ({ row }) => {
        const current = row.original.print_language || "gdi"
        const busy = updatingLanguageId === row.original.id
        const hint = row.original.language_hint || printLanguageShortHint(current, row.original.printer_name)
        return (
          <select
            className="flex h-8 w-full min-w-[130px] max-w-[160px] rounded-md border border-input bg-background px-2 text-sm"
            value={current}
            disabled={busy || updatingLanguageId !== null}
            onChange={(e) => {
              const next = e.target.value
              if (next === current) return
              void updatePrinterLanguage(row.original.id, next)
            }}
            title={hint}
          >
            <option value="gdi">{PRINT_LANGUAGE_LABELS.gdi}</option>
            <option value="tspl">{PRINT_LANGUAGE_LABELS.tspl}</option>
            <option value="zpl">{PRINT_LANGUAGE_LABELS.zpl}</option>
          </select>
        )
      },
    }),
    columnHelper.display({
      id: "actions",
      header: "Actions",
      cell: ({ row }) => {
        const isLoadingQueue = loadingQueueId === row.original.id
        const isTesting = testingPrintId === row.original.id
        return (
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => void testPrint(row.original)}
              disabled={testingPrintId !== null || !row.original.label_template_id}
              title="Test print (shablon sample)"
            >
              {isTesting ? <Loader2 className="size-4 animate-spin" /> : "Test"}
            </Button>
            <Button
              variant="outline"
              size="icon"
              onClick={() => void loadPrinterQueue(row.original)}
              disabled={loadingQueueId !== null}
              title="Aktiv queue"
            >
              {isLoadingQueue ? <Loader2 className="size-4 animate-spin" /> : <Eye className="size-4" />}
            </Button>
            <Button variant="outline" size="icon" onClick={() => void deletePrinter(row.original.id)}>
              <X style={{ color: "red" }} />
            </Button>
          </div>
        )
      },
    }),
  ]

  const table = useReactTable({
    data: printers,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  useEffect(() => {
    void printersGetAll()
    void linesGetAll()
    void templatesGetAll()
    void loadLocalPrinters()
  }, [])

  return (
    <PageContainer
      title="Printerlar V2"
      description="Zebra/Gprinter uchun ZPL/TSPL tavsiya. GDI — Windows fallback. AC-FILE* — test uchun PNG (print_preview/)."
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" asChild>
            <Link to="/printers-v2/metrics">Print metrics</Link>
          </Button>
          <Button variant="outline" className="gap-2" onClick={() => void loadLocalPrinters()} disabled={loadingLocal}>
            {loadingLocal ? <Loader2 className="size-4 animate-spin" /> : <RefreshCcw className="size-4" />}
            Printerlarni yangilash
          </Button>
        </div>
      }
    >
      <Panel title="Ro'yxat" noPadding>
        <div className="overflow-hidden rounded-b-2xl">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((hg) => (
                <TableRow key={hg.id}>
                  {hg.headers.map((header) => (
                    <TableHead key={header.id}>
                      {flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={columns.length} className="py-8 text-center text-muted-foreground">
                    V2 printerlar yo&apos;q
                  </TableCell>
                </TableRow>
              ) : (
                table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </Panel>

      <Panel title="Yangi printer" className="mt-4">
        <div className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">Liniya</Label>
              {dropDownLines()}
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">Printer</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                value={printerName}
                onChange={(e) => {
                  const name = e.target.value
                  setPrinterName(name)
                  void detectPrinterLanguage(name)
                }}
                disabled={loadingLocal}
              >
                <option value="">{loadingLocal ? "Yuklanmoqda..." : "Printer tanlang"}</option>
                <option value="AC-FILE-ZPL">AC-FILE-ZPL (PNG preview, ZPL)</option>
                <option value="AC-FILE-TSPL">AC-FILE-TSPL (PNG preview, TSPL)</option>
                {localPrinters.map((p) => (
                  <option key={p.name} value={p.name}>
                    {p.name}
                    {p.is_default ? " (default)" : ""}
                  </option>
                ))}
              </select>
              <p className="text-[10px] text-muted-foreground">
                AC-FILE* — Windows printer emas; chop natijasi <code>print_preview/</code> papkasiga PNG (+ .zpl/.tspl) yoziladi.
              </p>
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">Print tili</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                value={selectedLanguage}
                onChange={(e) => setSelectedLanguage(e.target.value)}
                disabled={!printerName || detectingLanguage}
              >
                <option value="gdi">{PRINT_LANGUAGE_LABELS.gdi}</option>
                <option value="tspl">{PRINT_LANGUAGE_LABELS.tspl}</option>
                <option value="zpl">{PRINT_LANGUAGE_LABELS.zpl}</option>
              </select>
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">Etiketka shablon</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                value={selectedTemplateId}
                onChange={(e) => setSelectedTemplateId(e.target.value)}
                disabled={!selectedLine}
              >
                <option value="">Tanlanmagan</option>
                {lineTemplates.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {printerName ? (
            <div className="rounded-lg border border-border bg-muted/30 px-3 py-2.5 text-sm">
              {detectingLanguage ? (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Loader2 className="size-3.5 animate-spin" />
                  Print tili aniqlanmoqda…
                </div>
              ) : (
                <div className="flex flex-col gap-1.5 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
                  <div className="min-w-0 space-y-1">
                    <div className="font-medium text-foreground">
                      {printLanguageShortHint(selectedLanguage, printerName)}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Avtomatik: {PRINT_LANGUAGE_LABELS[detectedLanguage] || detectedLanguage}
                      {selectedLanguage !== detectedLanguage ? " · qo‘lda o‘zgartirilgan" : ""}
                      {driverName ? ` · ${driverName}` : ""}
                    </div>
                  </div>
                  {detectReason ? (
                    <p className="max-w-md text-xs leading-relaxed text-muted-foreground sm:text-right">
                      {detectReason}
                    </p>
                  ) : null}
                </div>
              )}
            </div>
          ) : null}

          <div className="flex justify-end">
            <Button disabled={!canAdd} onClick={() => void addPrinter()}>
              Qo&apos;shish
            </Button>
          </div>
        </div>
      </Panel>

      <Panel title={queuePrinterName ? `Aktiv queue: ${queuePrinterName}` : "Aktiv queue"} className="mt-4">
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Document</TableHead>
                <TableHead>User</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Submitted</TableHead>
                <TableHead>Pages</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {queueJobs.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="py-6 text-center text-muted-foreground">
                    Queue bo&apos;sh yoki printer tanlanmagan
                  </TableCell>
                </TableRow>
              ) : (
                queueJobs.map((job) => (
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
    </PageContainer>
  )
}
