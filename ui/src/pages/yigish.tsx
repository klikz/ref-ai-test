import { useCallback, useEffect, useMemo, useState } from "react"
import { Loader2, Printer, RefreshCw } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import { cn } from "@/lib/utils"
import {
  LAST_RECORDS_VISIBLE_ROWS,
  LAST_RECORDS_PANEL_CLASS,
} from "@/lib/last-records"
import {
  YIGISH_LINE_ID,
  fetchPlanDay,
  type PlanItemRow,
} from "@/pages/production_plan_shared"

type PrinterV2 = {
  id: number
  line_id: number
  printer_name: string
  address?: string
  label_template_id?: number
  label_template_name?: string
}

type LastPrintRow = {
  id: number
  serial: string
  acc_serial?: string
  model: string
  model_nomi: string
  time: string
}

type PrintResponse = {
  serial?: string
  gscode?: string
  model?: { modeli?: string; qisqa_nomi?: string }
}

function planModelDetail(value?: string | number | null): string {
  if (value == null) {
    return "—"
  }
  const text = String(value).trim()
  return text || "—"
}

function yigishGsCodeRemaining(count?: number | null): string {
  if (count == null) {
    return "—"
  }
  return `${count} ta qoldiq`
}

function YigishPlanModelButton({ item, busy, done, disabled, onClick }: {
  item: PlanItemRow
  busy: boolean
  done: boolean
  disabled: boolean
  onClick: () => void
}) {
  const gsCodeRemaining = item.gscode_count ?? 0
  const noGsCode = gsCodeRemaining <= 0
  const details = [
    { label: "Brend", value: planModelDetail(item.brend) },
    { label: "Seriya raqami", value: planModelDetail(item.seriya_raqami) },
    { label: "Modeli", value: planModelDetail(item.modeli || item.label) },
    { label: "Rangi", value: planModelDetail(item.rangi) },
    {
      label: "GS Code",
      value: yigishGsCodeRemaining(item.gscode_count),
      warn: noGsCode,
    },
  ]

  const planProgress =
    item.planned_qty > 0 ? Math.min(100, Math.round((item.actual_qty / item.planned_qty) * 100)) : 0

  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={cn(
        "min-h-44 rounded-2xl border px-4 py-3.5 text-left transition shadow-sm",
        "hover:brightness-110 active:scale-[0.99]",
        "disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:brightness-100",
        done && "bg-muted text-muted-foreground",
        !done && noGsCode && "border-red-700 bg-red-600 text-white hover:bg-red-500",
        !done && !noGsCode && "border-transparent bg-primary text-primary-foreground",
      )}
    >
      <div className="space-y-2.5">
        <dl className="grid gap-1 text-xs leading-snug sm:text-sm">
          {details.map((row) => (
            <div key={row.label} className="grid grid-cols-[7.5rem_1fr] gap-2">
              <dt className="opacity-75">{row.label}</dt>
              <dd
                className={cn(
                  "font-medium",
                  row.warn && (done ? "text-amber-700 dark:text-amber-300" : noGsCode ? "text-red-100" : "text-amber-200"),
                )}
              >
                {row.value}
              </dd>
            </div>
          ))}
        </dl>
        <div
          className={cn(
            "space-y-2 border-t pt-2.5",
            done || noGsCode ? "border-current/20" : "border-primary-foreground/20",
          )}
        >
          <div className="grid grid-cols-[7.5rem_1fr_auto] items-center gap-2 text-xs sm:text-sm">
            <span className="opacity-75">Reja</span>
            <span className="font-semibold tabular-nums">
              {item.actual_qty}/{item.planned_qty}
              {item.odoo_code ? (
                <span className="ml-1 font-normal opacity-80">· {item.odoo_code}</span>
              ) : null}
            </span>
            {busy ? (
              <Loader2 className="size-5 animate-spin" />
            ) : (
              <Printer className="size-5 shrink-0 opacity-90" />
            )}
          </div>
          <div
            className={cn(
              "h-1.5 w-full overflow-hidden rounded-full",
              done || noGsCode ? "bg-black/10 dark:bg-white/10" : "bg-primary-foreground/20",
            )}
          >
            <div
              className={cn(
                "h-full rounded-full transition-all",
                done ? "bg-muted-foreground/70" : noGsCode ? "bg-white/90" : "bg-primary-foreground",
              )}
              style={{ width: `${planProgress}%` }}
            />
          </div>
        </div>
      </div>
    </button>
  )
}

export default function YigishPage() {
  const [planItems, setPlanItems] = useState<PlanItemRow[]>([])
  const [planStatus, setPlanStatus] = useState("")
  const [planDate, setPlanDate] = useState("")
  const [shiftLabel, setShiftLabel] = useState("")
  const [printers, setPrinters] = useState<PrinterV2[]>([])
  const [printerId, setPrinterId] = useState(0)
  const [lastRows, setLastRows] = useState<LastPrintRow[]>([])
  const [loading, setLoading] = useState(true)
  const [printingModelId, setPrintingModelId] = useState(0)
  const [reprintingSerial, setReprintingSerial] = useState("")

  const loadPlan = useCallback(async () => {
    const result = await fetchPlanDay(YIGISH_LINE_ID, undefined, { currentShift: true })
    if (result.result !== "ok" || !result.data) {
      ShowErrorToast(result.error || "Reja yuklanmadi")
      setPlanItems([])
      setPlanStatus("")
      return
    }
    setPlanStatus(result.data.status || "")
    setPlanDate(result.data.plan_date || "")
    setShiftLabel(result.data.current_shift?.label || "")
    const items = (result.data.items || []).filter((item) => item.model_id > 0 && item.planned_qty > 0)
    setPlanItems(items)
  }, [])

  const loadPrinters = useCallback(async () => {
    const result = await Backend_Request<PrinterV2[]>({ line_id: YIGISH_LINE_ID }, "/api/tech/printers-v2/by-line")
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Printerlar yuklanmadi")
      return
    }
    const list = result.data || []
    setPrinters(list)
    setPrinterId((current) => {
      if (current && list.some((p) => p.id === current)) {
        return current
      }
      return list[0]?.id ?? 0
    })
  }, [])

  const loadLast = useCallback(async () => {
    const result = await Backend_Request<LastPrintRow[]>({ line_id: YIGISH_LINE_ID }, "/api/lines/last")
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Oxirgi printlar yuklanmadi")
      return
    }
    setLastRows((result.data || []).slice(0, LAST_RECORDS_VISIBLE_ROWS))
  }, [])

  const reloadAll = useCallback(async () => {
    setLoading(true)
    await Promise.all([loadPlan(), loadPrinters(), loadLast()])
    setLoading(false)
  }, [loadLast, loadPlan, loadPrinters])

  useEffect(() => {
    reloadAll()
  }, [reloadAll])

  const selectedPrinter = useMemo(
    () => printers.find((item) => item.id === printerId),
    [printerId, printers],
  )

  async function printModel(item: PlanItemRow) {
    if (!printerId) {
      ShowErrorToast("Printer tanlang")
      return
    }
    if (planStatus !== "locked") {
      ShowErrorToast("Reja lock qilinmagan")
      return
    }
    setPrintingModelId(item.model_id)
    const result = await Backend_Request<PrintResponse>(
      { model_id: item.model_id, printer_v2_id: printerId, quantity: 1 },
      "/api/lines/yigish/v2/print",
    )
    setPrintingModelId(0)
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Chop etilmadi")
      return
    }
    ShowOKToast(result.data?.serial ? `Chop etildi: ${result.data.serial}` : "Chop etildi")
    await Promise.all([loadPlan(), loadLast()])
  }

  async function reprintSerial(serial: string) {
    if (!printerId) {
      ShowErrorToast("Printer tanlang")
      return
    }
    setReprintingSerial(serial)
    const result = await Backend_Request<PrintResponse>(
      { serial, printer_v2_id: printerId },
      "/api/lines/yigish/v2/reprint",
    )
    setReprintingSerial("")
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Qayta chop etilmadi")
      return
    }
    ShowOKToast(`Qayta chop: ${serial}`)
  }

  return (
    <PageContainer
      title="Boshlang'ich yig'uv uchastkasi"
      description={[planDate, shiftLabel, planStatus ? `Reja: ${planStatus}` : ""]
        .filter(Boolean)
        .join(" · ")}
      actions={
        <div className="flex flex-wrap items-center justify-end gap-2">
          <select
            value={printerId || ""}
            onChange={(event) => setPrinterId(Number(event.target.value))}
            className="h-8 min-w-44 max-w-xs rounded-lg border border-input bg-background px-2 text-sm"
            title={selectedPrinter?.address || undefined}
          >
            <option value="">Printer tanlang</option>
            {printers.map((printer) => (
              <option key={printer.id} value={printer.id}>
                {printer.printer_name}
                {printer.label_template_name ? ` · ${printer.label_template_name}` : ""}
              </option>
            ))}
          </select>
          <Button variant="outline" className="rounded-xl" onClick={reloadAll} disabled={loading}>
            <RefreshCw className={cn("size-4", loading && "animate-spin")} />
            Yangilash
          </Button>
        </div>
      }
    >
      <Panel
        title="Rejadagi modellar"
        description={
          planStatus === "locked"
            ? "Modelni bosing — serial generate qilinadi va printerdan chop etiladi"
            : "Reja lock qilingandan keyin chop etish mumkin"
        }
      >
        {loading ? (
          <div className="flex items-center justify-center gap-2 py-16 text-muted-foreground">
            <Loader2 className="size-5 animate-spin" />
            Yuklanmoqda...
          </div>
        ) : planItems.length === 0 ? (
          <div className="py-16 text-center text-muted-foreground">
            Joriy smenada rejadagi model yo‘q
          </div>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {planItems.map((item) => {
              const busy = printingModelId === item.model_id
              const done = item.remaining_qty <= 0 && !item.allow_overplan
              return (
                <YigishPlanModelButton
                  key={`${item.model_id}-${item.shift_no}`}
                  item={item}
                  busy={busy}
                  done={done}
                  disabled={busy || !printerId || planStatus !== "locked" || done || (item.gscode_count ?? 0) <= 0}
                  onClick={() => printModel(item)}
                />
              )
            })}
          </div>
        )}
      </Panel>

      <Panel
        title={`Oxirgi ${LAST_RECORDS_VISIBLE_ROWS} ta`}
        description="Qayta chop etish uchun printer ikonkasini bosing"
        className={LAST_RECORDS_PANEL_CLASS}
      >
        {lastRows.length === 0 ? (
          <div className="py-10 text-center text-muted-foreground">Hali chop etilgan mahsulot yo‘q</div>
        ) : (
          <div className="overflow-hidden rounded-xl border">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-left">
                <tr>
                  <th className="px-4 py-3 font-medium">Serial</th>
                  <th className="px-4 py-3 font-medium">Model</th>
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
                      <td className="px-4 py-3">
                        <div className="font-medium">{row.model || row.model_nomi}</div>
                        {row.model && row.model_nomi && row.model !== row.model_nomi ? (
                          <div className="text-xs text-muted-foreground">{row.model_nomi}</div>
                        ) : null}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{row.time}</td>
                      <td className="px-2 py-2 text-right">
                        <Button
                          size="icon"
                          variant="ghost"
                          className="rounded-xl"
                          disabled={!printerId || busy}
                          onClick={() => reprintSerial(row.serial)}
                          title="Qayta chop etish"
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
