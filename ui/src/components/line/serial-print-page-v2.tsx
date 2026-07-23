import { useEffect, useMemo, useRef, useState, type ReactNode } from "react"
import { Link } from "react-router-dom"
import { CameraOff, Loader2, Printer } from "lucide-react"
import {
  createColumnHelper,
  getCoreRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
} from "@tanstack/react-table"
import { PanelVirtualTable } from "@/components/layout/panel-virtual-table"
import { Backend_Request } from "@/services/backend"
import { Global_Data } from "@/config/config"
import { ShowErrorToast, ShowOKToast, ShowWarningToast } from "@/components/showToast"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { ProductionPlanWidget } from "@/components/production_plan_widget"
import { LAST_RECORDS_PANEL_CLASS, LAST_RECORDS_TITLE } from "@/lib/last-records"
import { onInputFocusForKeyboard, useKeyboardSafeInput } from "@/lib/keyboard-safe-input"
import { playScanError, playScanOk } from "@/lib/scan-feedback"
import { cn } from "@/lib/utils"

type PrinterV2 = {
  id: number
  line_id: number
  printer_name: string
  label_template_id: number
  label_template_name: string
}

type LastProduct = {
  serial: string
  acc_serial?: string
  model: string
  model_nomi: string
}

type LastScanDisplay = {
  serial: string
  acc_serial?: string
  file_path?: string
  photo_error?: string
}

type SerialPrintPageV2Props = {
  title: string
  description?: string
  lineID: number
  lineLabel?: string
  printURL: string
  reprintURL?: string
  requireAccSerial?: boolean
  showScanPhoto?: boolean
  titleAddon?: ReactNode
}

const columnHelper = createColumnHelper<LastProduct>()

export function SerialPrintPageV2({
  title,
  description = "V2",
  lineID,
  lineLabel,
  printURL,
  reprintURL,
  requireAccSerial = false,
  showScanPhoto = false,
  titleAddon,
}: SerialPrintPageV2Props) {
  const serialRef = useRef<HTMLInputElement>(null)
  const accSerialRef = useRef<HTMLInputElement>(null)
  const submittingRef = useRef(false)

  useKeyboardSafeInput(serialRef)
  useKeyboardSafeInput(accSerialRef)
  const [lastProducts, setLastProducts] = useState<LastProduct[]>([])
  const [printers, setPrinters] = useState<PrinterV2[]>([])
  const [printersLoading, setPrintersLoading] = useState(true)
  const [selectedPrinter, setSelectedPrinter] = useState<PrinterV2 | null>(null)
  const [sorting, setSorting] = useState<SortingState>([])
  const [accSerialDraft, setAccSerialDraft] = useState("")
  const [printing, setPrinting] = useState(false)
  const [reprintingSerial, setReprintingSerial] = useState<string | null>(null)
  const [planRefreshKey, setPlanRefreshKey] = useState(0)
  const [lastScanDisplay, setLastScanDisplay] = useState<LastScanDisplay | null>(null)
  const [scanError, setScanError] = useState("")

  function bumpPlanRefresh() {
    setPlanRefreshKey((key) => key + 1)
  }

  function clearScanError() {
    if (scanError) {
      setScanError("")
    }
  }

  async function printersGetAll() {
    setPrintersLoading(true)
    const result = await Backend_Request<PrinterV2[]>({ line_id: lineID }, "/api/tech/printers-v2/by-line")
    if (result.result === "ok" && result.data) {
      setPrinters(result.data)
      setSelectedPrinter((current) => current ?? result.data[0] ?? null)
    } else {
      ShowErrorToast(result.error || "Printerlar yuklanmadi")
    }
    setPrintersLoading(false)
  }

  async function productsGetLast() {
    const result = await Backend_Request<LastProduct[]>({ line_id: lineID }, "/api/lines/last")
    if (result.result === "ok" && result.data) {
      setLastProducts(result.data)
    } else {
      ShowErrorToast(result.error || "Oxirgi mahsulotlar yuklanmadi")
    }
  }

  function focusSerialInput() {
    window.setTimeout(() => {
      const input = serialRef.current
      if (!input) {
        return
      }
      input.focus()
      input.select()
    }, 50)
  }

  function readSerialValue() {
    return serialRef.current?.value.trim() ?? ""
  }

  function readAccSerialValue() {
    if (requireAccSerial) {
      return accSerialDraft.trim()
    }
    return accSerialRef.current?.value.trim() ?? ""
  }

  function clearInputs() {
    if (serialRef.current) {
      serialRef.current.value = ""
    }
    if (accSerialRef.current) {
      accSerialRef.current.value = ""
    }
    setAccSerialDraft("")
  }

  function showScanError(message: string) {
    setScanError(message)
    playScanError()
    clearInputs()
    focusSerialInput()
  }

  function focusAccSerialInput() {
    const serial = readSerialValue()
    if (!serial) {
      showScanError("Serial nomer bo'sh")
      return
    }
    window.setTimeout(() => accSerialRef.current?.focus(), 0)
  }

  function submitScanner() {
    const serial = readSerialValue()
    const accSerial = readAccSerialValue()

    if (!serial) {
      showScanError("Serial nomer bo'sh")
      return
    }

    if (requireAccSerial && !accSerial) {
      focusAccSerialInput()
      return
    }

    void printLabel(serial, false, accSerial)
  }

  function handleScannerSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    submitScanner()
  }

  const activePrinter = selectedPrinter ?? printers[0] ?? null
  const canPrint = Boolean(activePrinter?.label_template_id) && !printersLoading

  async function printLabel(serial: string, reprint: boolean, accSerial = "") {
    if (submittingRef.current || printing || reprintingSerial) {
      return
    }

    const normalizedSerial = serial.trim()
    const normalizedAccSerial = accSerial.trim()

    if (!normalizedSerial) {
      showScanError("Serial nomer bo'sh")
      return
    }

    if (requireAccSerial && !reprint && !normalizedAccSerial) {
      showScanError("Aksessuar nomer bo'sh")
      return
    }

    if (!activePrinter) {
      showScanError("Printer tanlanmagan")
      return
    }
    if (!activePrinter.label_template_id) {
      showScanError("Etiketka shablon tanlanmagan")
      return
    }

    const payload: Record<string, unknown> = {
      serial: normalizedSerial,
      printer_v2_id: Number(activePrinter.id),
    }
    if (requireAccSerial && !reprint) {
      payload.acc_serial = normalizedAccSerial
    }

    submittingRef.current = true
    if (reprint) {
      setReprintingSerial(normalizedSerial)
    } else {
      setPrinting(true)
    }

    let result: Awaited<ReturnType<typeof Backend_Request>>
    try {
      result = await Backend_Request(
        payload,
        reprint && reprintURL ? reprintURL : printURL,
      )
    } finally {
      submittingRef.current = false
      setPrinting(false)
      setReprintingSerial(null)
    }

    if (result.result === "ok") {
      clearInputs()
      setScanError("")
      playScanOk()
      ShowOKToast(`${normalizedSerial}: ${reprint ? "Qayta chiqarildi" : "Chiqarildi"}`)
      const data = result.data as
        | {
            plan_warning?: string
            scan_photo?: { serial?: string; file_path?: string }
            scan_photo_error?: string
          }
        | undefined
      const planWarning = data?.plan_warning
      if (planWarning) {
        ShowWarningToast(planWarning)
      }
      if (showScanPhoto && !reprint) {
        const scanPhoto = data?.scan_photo
        const scanPhotoError = data?.scan_photo_error
        setLastScanDisplay({
          serial: scanPhoto?.serial || normalizedSerial,
          acc_serial: requireAccSerial ? normalizedAccSerial : undefined,
          file_path: scanPhoto?.file_path,
          photo_error: scanPhotoError,
        })
        if (scanPhotoError) {
          ShowErrorToast(scanPhotoError)
        }
      }
      if (!reprint) {
        bumpPlanRefresh()
      }
      await productsGetLast()
      focusSerialInput()
      return
    }

    showScanError(`${normalizedSerial}: ${result.error || "Xatolik"}`)
  }

  const isBusy = printing || reprintingSerial !== null

  const tableColumns = useMemo(
    () => [
      columnHelper.accessor("serial", {
        header: "Serial",
        cell: ({ getValue }) => (
          <span className="block truncate font-mono text-xs" title={String(getValue() ?? "")}>
            {String(getValue() ?? "")}
          </span>
        ),
      }),
      ...(requireAccSerial
        ? [
            columnHelper.accessor((row) => row.acc_serial ?? "", {
              id: "acc_serial",
              header: "Aksessuar",
              cell: ({ getValue }) => {
                const value = String(getValue() ?? "").trim()
                return (
                  <span className="block truncate font-mono text-xs" title={value}>
                    {value || "—"}
                  </span>
                )
              },
            }),
          ]
        : []),
      columnHelper.accessor("model", {
        header: "Modeli",
        cell: ({ getValue }) => (
          <span className="block truncate" title={String(getValue() ?? "")}>
            {String(getValue() ?? "")}
          </span>
        ),
      }),
      columnHelper.accessor("model_nomi", {
        header: "Model nomi",
        cell: ({ getValue }) => (
          <span className="block truncate" title={String(getValue() ?? "")}>
            {String(getValue() ?? "")}
          </span>
        ),
      }),
      columnHelper.display({
        id: "actions",
        header: "",
        cell: ({ row }) => {
          const isRowPrinting = reprintingSerial === row.original.serial
          const rowBusy = printing || reprintingSerial !== null
          return (
            <Button
              variant="outline"
              size="icon"
              className="size-11 rounded-xl"
              disabled={!canPrint || rowBusy}
              onClick={() => void printLabel(row.original.serial, true)}
            >
              {isRowPrinting ? <Loader2 className="size-4 animate-spin" /> : <Printer className="size-4" />}
            </Button>
          )
        },
      }),
    ],
    [canPrint, printing, reprintingSerial, requireAccSerial],
  )

  const table = useReactTable({
    data: lastProducts,
    columns: tableColumns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  })

  useEffect(() => {
    void printersGetAll()
    void productsGetLast()
    window.setTimeout(() => serialRef.current?.focus(), 100)
  }, [])

  const lastProductsGrid = useMemo(
    () =>
      requireAccSerial
        ? "minmax(0, 1fr) minmax(0, 0.9fr) minmax(0, 0.8fr) minmax(0, 1fr) 3.5rem"
        : "minmax(0, 1.2fr) minmax(0, 0.8fr) minmax(0, 1.2fr) 3.5rem",
    [requireAccSerial],
  )

  return (
    <PageContainer title={title} description={description} titleAddon={titleAddon} fullWidth contentClassName="space-y-2">
      <ProductionPlanWidget
        lineId={lineID}
        lineLabel={lineLabel}
        className="mb-1 w-full"
        refreshKey={planRefreshKey}
        hideHeader
        compact
      />
      <div className="grid w-full min-w-0 grid-cols-1 gap-3 lg:grid-cols-2 lg:items-start xl:grid-cols-[minmax(0,1fr)_minmax(0,1.15fr)]">
        <div className="flex min-w-0 flex-col gap-3">
          <Panel title="Printer V2" bodyClassName="p-3 sm:p-4">
            {printersLoading ? (
              <div className="flex items-center gap-2 py-6 text-sm text-muted-foreground">
                <Loader2 className="size-4 animate-spin" />
                Printerlar yuklanmoqda…
              </div>
            ) : printers.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                Bu liniya uchun V2 printer yo&apos;q.{" "}
                <Link to="/printers-v2" className="text-primary underline">
                  Printerlar V2
                </Link>{" "}
                sahifasidan qo&apos;shing.
              </p>
            ) : (
              <RadioGroup
                value={activePrinter?.id ? String(activePrinter.id) : undefined}
                className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-1"
                onValueChange={(value) =>
                  setSelectedPrinter(printers.find((printer) => printer.id === Number(value)) ?? null)
                }
              >
                {printers.map((printer) => (
                  <Label
                    key={printer.id}
                    htmlFor={`printer-v2-${lineID}-${printer.id}`}
                    className={cn(
                      "flex min-h-14 cursor-pointer flex-col justify-center gap-0.5 rounded-xl border p-3 text-base font-medium transition-colors",
                      activePrinter?.id === printer.id
                        ? "border-primary bg-primary/10 text-primary"
                        : "border-border bg-background/50 hover:bg-muted/60",
                    )}
                  >
                    <div className="flex items-center gap-3">
                      <RadioGroupItem
                        value={String(printer.id)}
                        id={`printer-v2-${lineID}-${printer.id}`}
                        className="size-5"
                      />
                      <span className="truncate">{printer.printer_name}</span>
                    </div>
                    {printer.label_template_name ? (
                      <span className="pl-8 text-sm font-normal text-muted-foreground">
                        {printer.label_template_name}
                      </span>
                    ) : (
                      <span className="pl-8 text-sm font-normal text-amber-600">Shablon tanlanmagan</span>
                    )}
                  </Label>
                ))}
              </RadioGroup>
            )}
          </Panel>

          <Panel title="Serial skaner" bodyClassName="p-3 sm:p-4">
            <form className="flex flex-col gap-3" onSubmit={handleScannerSubmit}>
              <div className="space-y-2">
                <Label htmlFor={`serial-v2-${lineID}`} className="text-base font-medium">
                  Serial
                </Label>
                <Input
                  ref={serialRef}
                  id={`serial-v2-${lineID}`}
                  type="text"
                  placeholder="Serial nomer"
                  className="h-14 w-full scroll-mb-4 text-lg"
                  disabled={isBusy}
                  onFocus={onInputFocusForKeyboard}
                  onInput={clearScanError}
                  onKeyDown={(e) => {
                    if (e.key !== "Enter") {
                      return
                    }
                    e.preventDefault()
                    e.stopPropagation()
                    submitScanner()
                  }}
                />
              </div>
              {requireAccSerial ? (
                <div className="space-y-2">
                  <Label htmlFor={`acc-serial-v2-${lineID}`} className="text-base font-medium">
                    Aksessuar
                  </Label>
                  <Input
                    ref={accSerialRef}
                    id={`acc-serial-v2-${lineID}`}
                    type="text"
                    placeholder="Aksessuar nomer"
                    className="h-14 w-full scroll-mb-4 text-lg"
                    value={accSerialDraft}
                    disabled={isBusy}
                    onFocus={onInputFocusForKeyboard}
                    onChange={(e) => {
                      clearScanError()
                      setAccSerialDraft(e.target.value)
                    }}
                    onKeyDown={(e) => {
                      if (e.key !== "Enter") {
                        return
                      }
                      e.preventDefault()
                      e.stopPropagation()
                      submitScanner()
                    }}
                  />
                </div>
              ) : null}
              <Button
                type="button"
                className="h-20 w-full rounded-2xl text-lg font-semibold shadow-lg lg:h-24 lg:text-xl"
                disabled={!canPrint || isBusy}
                onClick={() => submitScanner()}
              >
                {printing ? <Loader2 className="size-6 animate-spin" /> : <Printer className="size-6" />}
                {printing ? "Chop etilmoqda..." : "Print"}
              </Button>
            </form>
          </Panel>
        </div>

        <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-2">
          {scanError ? (
            <p className="shrink-0 rounded-xl border border-red-500/40 bg-red-500/10 px-3 py-2 text-base font-semibold text-red-600 dark:text-red-400">
              {scanError}
            </p>
          ) : null}
          <Panel
            title={LAST_RECORDS_TITLE}
            noPadding
            className={LAST_RECORDS_PANEL_CLASS}
            bodyClassName="min-h-0 flex-1 overflow-hidden flex flex-col"
          >
            {showScanPhoto && lastScanDisplay ? (
            <div className="flex shrink-0 items-center gap-4 border-b border-border bg-muted/20 p-3 sm:p-4">
              {lastScanDisplay.file_path ? (
                <img
                  src={`${Global_Data.server_ip}${lastScanDisplay.file_path}`}
                  alt=""
                  className="h-36 w-auto max-w-[55%] shrink-0 rounded-xl border border-border bg-black/5 object-contain sm:h-44"
                />
              ) : (
                <div
                  className={cn(
                    "flex h-36 w-[55%] max-w-[55%] shrink-0 flex-col items-center justify-center gap-2 rounded-xl border border-dashed px-3 text-center sm:h-44",
                    lastScanDisplay.photo_error
                      ? "border-destructive/40 bg-destructive/5 text-destructive"
                      : "border-border bg-muted/40 text-muted-foreground",
                  )}
                >
                  <CameraOff className="size-10 opacity-70 sm:size-12" strokeWidth={1.5} />
                  <p className="text-sm font-medium sm:text-base">Surat yo&apos;q</p>
                  {lastScanDisplay.photo_error ? (
                    <p className="text-xs leading-snug opacity-90 sm:text-sm">{lastScanDisplay.photo_error}</p>
                  ) : null}
                </div>
              )}
              <div className="flex min-w-0 flex-1 flex-col justify-center gap-2">
                <p className="break-all font-mono text-3xl font-bold leading-tight tracking-tight sm:text-4xl lg:text-5xl">
                  {lastScanDisplay.serial}
                </p>
                {lastScanDisplay.acc_serial ? (
                  <p className="break-all font-mono text-xl font-semibold leading-tight text-muted-foreground sm:text-2xl lg:text-3xl">
                    {lastScanDisplay.acc_serial}
                  </p>
                ) : null}
              </div>
            </div>
          ) : null}
          <PanelVirtualTable
            table={table}
            gridTemplateColumns={lastProductsGrid}
            emptyMessage="Hozircha mahsulot yo'q"
          />
          </Panel>
        </div>
      </div>
    </PageContainer>
  )
}
