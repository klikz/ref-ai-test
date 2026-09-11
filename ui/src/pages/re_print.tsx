import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { Link } from "react-router-dom"
import { Loader2, Printer } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import { onInputFocusForKeyboard, useKeyboardSafeInput } from "@/lib/keyboard-safe-input"
import { playScanError, playScanOk } from "@/lib/scan-feedback"
import {
  LAST_RECORDS_PANEL_CLASS,
  LAST_RECORDS_VISIBLE_ROWS,
} from "@/lib/last-records"
import {
  ESHIK_LINE_ID,
  QADOQLASH_LINE_ID,
  YIGISH_LINE_ID,
} from "@/pages/production_plan_shared"
import { cn } from "@/lib/utils"

type ReprintLineId = typeof YIGISH_LINE_ID | typeof ESHIK_LINE_ID | typeof QADOQLASH_LINE_ID

type ReprintLine = {
  id: ReprintLineId
  name: string
  reprintURL: string
  hint: string
}

type PrinterV2 = {
  id: number
  line_id: number
  printer_name: string
  address?: string
  label_template_id?: number
  label_template_name?: string
}

type LastRow = {
  id: string
  serial: string
  detail: string
  time: string
}

const REPRINT_LINES: ReprintLine[] = [
  {
    id: ESHIK_LINE_ID,
    name: "Eshik yig'uv va eshikka PPU quyish uchastkasi",
    reprintURL: "/api/lines/eshik/v2/reprint",
    hint: "Eshik serialini skanerlang — tanlangan printer/shablonga 1 nusxa",
  },
  {
    id: YIGISH_LINE_ID,
    name: "Boshlang'ich yig'uv uchastkasi",
    reprintURL: "/api/lines/yigish/v2/reprint",
    hint: "Mahsulot serialini skanerlang — tanlangan printer/shablonga 1 nusxa",
  },
  {
    id: QADOQLASH_LINE_ID,
    name: "Yakuniy yig'uv uchastkasi",
    reprintURL: "/api/lines/qadoqlash/reprint",
    hint: "Product serialini skanerlang — faqat tanlangan bitta shablon chop etiladi",
  },
]

function reprintPayload(lineId: ReprintLineId, serial: string, printerId: number) {
  const payload: Record<string, unknown> = {
    serial,
    printer_v2_id: printerId,
  }
  if (lineId === ESHIK_LINE_ID) {
    payload.copy = 1
  }
  return payload
}

export default function RePrintPage() {
  const serialRef = useRef<HTMLInputElement>(null)
  useKeyboardSafeInput(serialRef)

  const [lineId, setLineId] = useState<ReprintLineId>(YIGISH_LINE_ID)
  const [printers, setPrinters] = useState<PrinterV2[]>([])
  const [printersLoading, setPrintersLoading] = useState(true)
  const [printerId, setPrinterId] = useState(0)
  const [serial, setSerial] = useState("")
  const [printing, setPrinting] = useState(false)
  const [reprintingSerial, setReprintingSerial] = useState("")
  const [lastRows, setLastRows] = useState<LastRow[]>([])
  const [scanError, setScanError] = useState("")

  const line = useMemo(
    () => REPRINT_LINES.find((item) => item.id === lineId) ?? REPRINT_LINES[0],
    [lineId],
  )
  const selectedPrinter = printers.find((item) => item.id === printerId)
  const canPrint = Boolean(printerId && selectedPrinter?.label_template_id) && !printing && !reprintingSerial
  const isBusy = printing || reprintingSerial !== ""

  const loadPrinters = useCallback(async (selectedLineId: ReprintLineId) => {
    setPrintersLoading(true)
    const result = await Backend_Request<PrinterV2[]>(
      { line_id: selectedLineId },
      "/api/tech/printers-v2/by-line",
    )
    setPrintersLoading(false)
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Printerlar yuklanmadi")
      setPrinters([])
      setPrinterId(0)
      return
    }
    const list = result.data ?? []
    setPrinters(list)
    setPrinterId((current) => {
      if (current && list.some((item) => item.id === current)) {
        return current
      }
      return list[0]?.id ?? 0
    })
  }, [])

  const loadLast = useCallback(async (selectedLineId: ReprintLineId) => {
    if (selectedLineId === YIGISH_LINE_ID) {
      const result = await Backend_Request<Array<{ id?: number; serial?: string; model?: string; model_nomi?: string; time?: string }>>(
        { line_id: YIGISH_LINE_ID },
        "/api/lines/last",
      )
      if (result.result !== "ok") {
        ShowErrorToast(result.error || "Oxirgi yozuvlar yuklanmadi")
        return
      }
      setLastRows(
        (result.data ?? []).slice(0, LAST_RECORDS_VISIBLE_ROWS).map((row, index) => ({
          id: String(row.id ?? `${row.serial}-${index}`),
          serial: String(row.serial ?? ""),
          detail: String(row.model || row.model_nomi || ""),
          time: String(row.time ?? ""),
        })),
      )
      return
    }

    if (selectedLineId === ESHIK_LINE_ID) {
      const result = await Backend_Request<Array<{ session_id?: number; serial?: string; factory_code?: string; full_name_uz?: string; c_time?: string }>>(
        { limit: LAST_RECORDS_VISIBLE_ROWS },
        "/api/lines/eshik/v2/sessions/last",
      )
      if (result.result !== "ok") {
        ShowErrorToast(result.error || "Oxirgi yozuvlar yuklanmadi")
        return
      }
      setLastRows(
        (result.data ?? []).slice(0, LAST_RECORDS_VISIBLE_ROWS).map((row, index) => ({
          id: String(row.session_id ?? `${row.serial}-${index}`),
          serial: String(row.serial ?? ""),
          detail: [row.factory_code, row.full_name_uz].filter(Boolean).join(" · "),
          time: String(row.c_time ?? ""),
        })),
      )
      return
    }

    const result = await Backend_Request<{
      sessions?: Array<{ id?: number; serial?: string; modeli?: string; time?: string }>
    }>({ limit: LAST_RECORDS_VISIBLE_ROWS }, "/api/lines/qadoqlash/sessions/last")
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Oxirgi yozuvlar yuklanmadi")
      return
    }
    setLastRows(
      (result.data?.sessions ?? []).slice(0, LAST_RECORDS_VISIBLE_ROWS).map((row, index) => ({
        id: String(row.id ?? `${row.serial}-${index}`),
        serial: String(row.serial ?? ""),
        detail: String(row.modeli ?? ""),
        time: String(row.time ?? ""),
      })),
    )
  }, [])

  useEffect(() => {
    setScanError("")
    setSerial("")
    setLastRows([])
    void Promise.all([loadPrinters(lineId), loadLast(lineId)])
    requestAnimationFrame(() => serialRef.current?.focus())
  }, [lineId, loadLast, loadPrinters])

  async function reprintSerial(value: string, fromList: boolean) {
    const normalized = value.trim()
    if (!normalized || isBusy) {
      return
    }
    if (!printerId) {
      ShowErrorToast("Printer tanlang")
      return
    }
    if (!selectedPrinter?.label_template_id) {
      ShowErrorToast("Etiketka shablon tanlanmagan")
      return
    }

    if (fromList) {
      setReprintingSerial(normalized)
    } else {
      setPrinting(true)
    }

    const result = await Backend_Request(
      reprintPayload(lineId, normalized, printerId),
      line.reprintURL,
    )

    setPrinting(false)
    setReprintingSerial("")

    if (result.result !== "ok") {
      const message = result.error || "Qayta chop etilmadi"
      setScanError(`${normalized}: ${message}`)
      playScanError()
      ShowErrorToast(message)
      return
    }

    setScanError("")
    setSerial("")
    playScanOk()
    ShowOKToast(`${normalized}: qayta chiqarildi`)
    await loadLast(lineId)
    requestAnimationFrame(() => serialRef.current?.focus())
  }

  function onSerialSubmit(event: React.FormEvent) {
    event.preventDefault()
    void reprintSerial(serial, false)
  }

  return (
    <PageContainer
      title="Qayta chop"
      description="Boshlang'ich yig'uv, Eshik va Yakuniy yig'uv — tanlangan shablonga 1 nusxa"
    >
      <Panel title="Liniya" description="Qaysi liniya etiketkasini qayta chop qilish">
        <div className="grid gap-2 sm:grid-cols-3">
          {REPRINT_LINES.map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => setLineId(item.id)}
              className={cn(
                "rounded-2xl border px-4 py-3 text-left transition",
                lineId === item.id
                  ? "border-primary bg-primary/10 text-primary"
                  : "border-border bg-background/50 hover:bg-muted/60",
              )}
            >
              <p className="font-semibold">{item.name}</p>
              {item.id === QADOQLASH_LINE_ID ? (
                <p className="mt-1 text-xs font-normal text-muted-foreground">Faqat bitta shablon</p>
              ) : null}
            </button>
          ))}
        </div>
      </Panel>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
        <Panel title="Printer / shablon" description={line.hint}>
          {printersLoading ? (
            <div className="flex items-center gap-2 py-6 text-sm text-muted-foreground">
              <Loader2 className="size-4 animate-spin" />
              Printerlar yuklanmoqda…
            </div>
          ) : printers.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              Bu liniya uchun printer yo‘q.{" "}
              <Link to="/printers-v2" className="text-primary underline">
                Printerlar
              </Link>{" "}
              sahifasidan qo‘shing.
            </p>
          ) : (
            <RadioGroup
              value={printerId ? String(printerId) : undefined}
              className="grid grid-cols-1 gap-2"
              onValueChange={(value) => setPrinterId(Number(value))}
            >
              {printers.map((printer) => (
                <Label
                  key={printer.id}
                  htmlFor={`reprint-printer-${printer.id}`}
                  className={cn(
                    "flex min-h-14 cursor-pointer flex-col justify-center gap-0.5 rounded-xl border p-3 text-base font-medium transition-colors",
                    printerId === printer.id
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-border bg-background/50 hover:bg-muted/60",
                  )}
                >
                  <div className="flex items-center gap-3">
                    <RadioGroupItem
                      value={String(printer.id)}
                      id={`reprint-printer-${printer.id}`}
                      className="size-5"
                    />
                    <span className="truncate">{printer.printer_name}</span>
                  </div>
                  {printer.label_template_name ? (
                    <span className="pl-8 text-sm font-normal text-muted-foreground">
                      Shablon: {printer.label_template_name}
                    </span>
                  ) : (
                    <span className="pl-8 text-sm font-normal text-amber-600">Shablon tanlanmagan</span>
                  )}
                </Label>
              ))}
            </RadioGroup>
          )}
        </Panel>

        <Panel title="Serial skaner">
          <form className="space-y-3" onSubmit={onSerialSubmit}>
            <div className="space-y-2">
              <Label htmlFor="reprint-serial">Serial</Label>
              <Input
                id="reprint-serial"
                ref={serialRef}
                value={serial}
                autoComplete="off"
                spellCheck={false}
                disabled={isBusy}
                placeholder="Serialni skanerlang va Enter"
                className="h-12 rounded-xl font-mono text-base"
                onFocus={onInputFocusForKeyboard}
                onChange={(event) => {
                  if (scanError) setScanError("")
                  setSerial(event.target.value)
                }}
              />
            </div>
            {scanError ? (
              <p className="text-sm font-medium text-destructive">{scanError}</p>
            ) : null}
            <Button type="submit" className="h-11 rounded-xl" disabled={!canPrint || !serial.trim()}>
              {printing ? <Loader2 className="size-4 animate-spin" /> : <Printer className="size-4" />}
              Qayta chop etish
            </Button>
          </form>
        </Panel>
      </div>

      <Panel
        title={`Oxirgi ${LAST_RECORDS_VISIBLE_ROWS} ta`}
        description="Ro‘yxatdan ham tanlangan shablonga qayta chop qilish mumkin"
        className={LAST_RECORDS_PANEL_CLASS}
      >
        {lastRows.length === 0 ? (
          <div className="py-10 text-center text-sm text-muted-foreground">Hali yozuv yo‘q</div>
        ) : (
          <div className="overflow-hidden rounded-xl border">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-left">
                <tr>
                  <th className="px-4 py-3 font-medium">Serial</th>
                  <th className="px-4 py-3 font-medium">Ma'lumot</th>
                  <th className="px-4 py-3 font-medium">Vaqt</th>
                  <th className="w-14 px-2 py-3" />
                </tr>
              </thead>
              <tbody>
                {lastRows.map((row) => {
                  const busy = reprintingSerial === row.serial
                  return (
                    <tr key={row.id} className="border-t">
                      <td className="px-4 py-3 font-mono text-xs sm:text-sm">{row.serial}</td>
                      <td className="px-4 py-3">{row.detail || "—"}</td>
                      <td className="px-4 py-3 text-muted-foreground">{row.time}</td>
                      <td className="px-2 py-2 text-right">
                        <Button
                          type="button"
                          size="icon"
                          variant="ghost"
                          className="rounded-xl"
                          disabled={!canPrint || busy}
                          title="Tanlangan shablonga qayta chop"
                          onClick={() => void reprintSerial(row.serial, true)}
                        >
                          {busy ? <Loader2 className="size-4 animate-spin" /> : <Printer className="size-4" />}
                        </Button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </Panel>
    </PageContainer>
  )
}
