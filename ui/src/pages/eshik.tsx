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
  ESHIK_LINE_ID,
  attachPlanProgress,
  fetchPlanDay,
  filterItemsByPlannedIds,
  parseLockedPlanDay,
  type PlanProgressFields,
} from "@/pages/production_plan_shared"

type PrinterV2 = {
  id: number
  line_id: number
  printer_name: string
  address?: string
  label_template_id?: number
  label_template_name?: string
}

type EshikPart = {
  id: number
  door_code: string
  component_id: number
  factory_code?: string
  seriya_raqami: string
  index1: string
  index2: string
}

type EshikModel = {
  id: number
  model_name: string
  freeze: EshikPart
  ref: EshikPart
} & PlanProgressFields

type EshikSession = {
  session_id: number
  eshik_model_id: number
  door_code: string
  model_name: string
  factory_code: string
  full_name_uz: string
  serial: string
  counter: number
  user_name: string
  c_time: string
}

type PrintResponse = {
  serial?: string
  parts?: { door_code?: string; serial?: string }[]
}

function EshikPlanModelButton({
  item,
  busy,
  done,
  disabled,
  onClick,
}: {
  item: EshikModel
  busy: boolean
  done: boolean
  disabled: boolean
  onClick: () => void
}) {
  const planProgress =
    item.planned_qty > 0 ? Math.min(100, Math.round((item.actual_qty / item.planned_qty) * 100)) : 0

  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={cn(
        "min-h-40 rounded-2xl border px-4 py-3.5 text-left transition shadow-sm",
        "hover:brightness-110 active:scale-[0.99]",
        "disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:brightness-100",
        done && "bg-muted text-muted-foreground",
        !done && "border-transparent bg-primary text-primary-foreground",
      )}
    >
      <div className="space-y-2.5">
        <div className="font-semibold leading-snug">{item.model_name}</div>
        <dl className="grid gap-1 text-xs leading-snug sm:text-sm">
          <div className="grid grid-cols-[7.5rem_1fr] gap-2">
            <dt className="opacity-75">Freeze</dt>
            <dd className="min-w-0 break-words font-mono font-medium">
              {item.freeze?.seriya_raqami || "—"} · {item.freeze?.index1 || "—"}/{item.freeze?.index2 || "—"}
            </dd>
          </div>
          <div className="grid grid-cols-[7.5rem_1fr] gap-2">
            <dt className="opacity-75">Ref</dt>
            <dd className="min-w-0 break-words font-mono font-medium">
              {item.ref?.seriya_raqami || "—"} · {item.ref?.index1 || "—"}/{item.ref?.index2 || "—"}
            </dd>
          </div>
        </dl>
        <div className={cn("space-y-2 border-t pt-2.5", done ? "border-current/20" : "border-primary-foreground/20")}>
          <div className="grid grid-cols-[7.5rem_1fr_auto] items-center gap-2 text-xs sm:text-sm">
            <span className="opacity-75">Reja (eshik)</span>
            <span className="font-semibold tabular-nums">
              {item.actual_qty}/{item.planned_qty}
            </span>
            {busy ? <Loader2 className="size-5 animate-spin" /> : <Printer className="size-5 shrink-0 opacity-90" />}
          </div>
          <div className={cn("h-1.5 w-full overflow-hidden rounded-full", done ? "bg-black/10 dark:bg-white/10" : "bg-primary-foreground/20")}>
            <div
              className={cn("h-full rounded-full transition-all", done ? "bg-muted-foreground/70" : "bg-primary-foreground")}
              style={{ width: `${planProgress}%` }}
            />
          </div>
        </div>
      </div>
    </button>
  )
}

function normalizeSessions(data: unknown): EshikSession[] {
  if (!Array.isArray(data)) {
    return []
  }
  return data
    .map((row) => {
      const record = row as Record<string, unknown>
      return {
        session_id: Number(record.session_id ?? record.id ?? 0),
        eshik_model_id: Number(record.eshik_model_id ?? 0),
        door_code: String(record.door_code ?? ""),
        model_name: String(record.model_name ?? ""),
        factory_code: String(record.factory_code ?? ""),
        full_name_uz: String(record.full_name_uz ?? ""),
        serial: String(record.serial ?? ""),
        counter: Number(record.counter ?? 0),
        user_name: String(record.user_name ?? ""),
        c_time: String(record.c_time ?? ""),
      }
    })
    .filter((session) => session.session_id > 0)
}

export default function EshikPage() {
  const [items, setItems] = useState<EshikModel[]>([])
  const [planStatus, setPlanStatus] = useState("")
  const [planDate, setPlanDate] = useState("")
  const [shiftLabel, setShiftLabel] = useState("")
  const [printers, setPrinters] = useState<PrinterV2[]>([])
  const [printerId, setPrinterId] = useState(0)
  const [lastRows, setLastRows] = useState<EshikSession[]>([])
  const [loading, setLoading] = useState(true)
  const [printingId, setPrintingId] = useState(0)
  const [reprintingSessionId, setReprintingSessionId] = useState(0)

  const loadPlanItems = useCallback(async () => {
    const [modelsResult, planResult] = await Promise.all([
      Backend_Request<EshikModel[]>({}, "/api/production/eshik/all"),
      fetchPlanDay(ESHIK_LINE_ID, undefined, { currentShift: true }),
    ])

    if (modelsResult.result !== "ok") {
      ShowErrorToast(modelsResult.error || "Modellar yuklanmadi")
      setItems([])
      return
    }

    const all = modelsResult.data ?? []
    const { status, planByKey, plannedIds } = parseLockedPlanDay(planResult)
    setPlanStatus(status)
    setPlanDate(planResult.result === "ok" && planResult.data ? planResult.data.plan_date || "" : "")
    setShiftLabel(planResult.result === "ok" && planResult.data ? planResult.data.current_shift?.label || "" : "")

    setItems(
      filterItemsByPlannedIds(all, plannedIds, (item) => item.id).map((item) =>
        attachPlanProgress(item, planByKey, (row) => row.id),
      ),
    )
  }, [])

  const loadPrinters = useCallback(async () => {
    const result = await Backend_Request<PrinterV2[]>({ line_id: ESHIK_LINE_ID }, "/api/tech/printers-v2/by-line")
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
    const result = await Backend_Request<EshikSession[]>({ limit: LAST_RECORDS_VISIBLE_ROWS }, "/api/lines/eshik/v2/sessions/last")
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Oxirgi printlar yuklanmadi")
      return
    }
    setLastRows(normalizeSessions(result.data).slice(0, LAST_RECORDS_VISIBLE_ROWS))
  }, [])

  const reloadAll = useCallback(async () => {
    setLoading(true)
    await Promise.all([loadPlanItems(), loadPrinters(), loadLast()])
    setLoading(false)
  }, [loadLast, loadPlanItems, loadPrinters])

  useEffect(() => {
    reloadAll()
  }, [reloadAll])

  const selectedPrinter = useMemo(
    () => printers.find((item) => item.id === printerId),
    [printerId, printers],
  )

  async function printModel(item: EshikModel) {
    if (!printerId) {
      ShowErrorToast("Printer tanlang")
      return
    }
    if (planStatus !== "locked") {
      ShowErrorToast("Reja lock qilinmagan")
      return
    }
    setPrintingId(item.id)
    const result = await Backend_Request<PrintResponse>(
      { eshik_model_id: item.id, printer_v2_id: printerId, copy: 1 },
      "/api/lines/eshik/v2/print",
    )
    setPrintingId(0)
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Chop etilmadi")
      // Print fail after commit still leaves sessions/balance — refresh for reprint.
      await Promise.all([loadPlanItems(), loadLast()])
      return
    }
    const parts = result.data?.parts ?? []
    const serials = parts.map((part) => part.serial).filter(Boolean)
    ShowOKToast(serials.length ? `Chop etildi: ${serials.join(" · ")}` : "Chop etildi (freeze + ref)")
    await Promise.all([loadPlanItems(), loadLast()])
  }

  async function reprintSession(row: EshikSession) {
    if (!printerId) {
      ShowErrorToast("Printer tanlang")
      return
    }
    setReprintingSessionId(row.session_id)
    const result = await Backend_Request<PrintResponse>(
      { session_id: row.session_id, printer_v2_id: printerId, copy: 1 },
      "/api/lines/eshik/v2/reprint",
    )
    setReprintingSessionId(0)
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Qayta chop etilmadi")
      return
    }
    ShowOKToast(`Qayta chop: ${row.serial}`)
  }

  return (
    <PageContainer
      title="Eshik yig'uv va eshikka PPU quyish uchastkasi"
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
            ? "Bir bosishda freeze va ref chop etiladi"
            : "Reja lock qilingandan keyin chop etish mumkin"
        }
      >
        {loading ? (
          <div className="flex items-center justify-center gap-2 py-16 text-muted-foreground">
            <Loader2 className="size-5 animate-spin" />
            Yuklanmoqda...
          </div>
        ) : items.length === 0 ? (
          <div className="py-16 text-center text-muted-foreground">
            Joriy smenada rejadagi model yo‘q
          </div>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {items.map((item) => {
              const busy = printingId === item.id
              const done = item.planned_qty > 0 && item.actual_qty >= item.planned_qty && !item.allow_overplan
              return (
                <EshikPlanModelButton
                  key={item.id}
                  item={item}
                  busy={busy}
                  done={done}
                  disabled={busy || !printerId || planStatus !== "locked" || done}
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
          <div className="py-10 text-center text-muted-foreground">Hali chop etilgan serial yo‘q</div>
        ) : (
          <div className="overflow-hidden rounded-xl border">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-left">
                <tr>
                  <th className="px-4 py-3 font-medium">Serial</th>
                  <th className="px-4 py-3 font-medium">Model / eshik</th>
                  <th className="px-4 py-3 font-medium">Vaqt</th>
                  <th className="w-14 px-2 py-3" />
                </tr>
              </thead>
              <tbody>
                {lastRows.map((row) => {
                  const busy = reprintingSessionId === row.session_id
                  return (
                    <tr key={row.session_id} className="border-t">
                      <td className="px-4 py-3 font-mono text-xs sm:text-sm">{row.serial}</td>
                      <td className="px-4 py-3">
                        <div className="font-medium">{row.model_name || row.full_name_uz || row.factory_code}</div>
                        <div className="text-xs text-muted-foreground">
                          {row.door_code || "—"}
                          {row.factory_code ? ` · ${row.factory_code}` : ""}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{row.c_time}</td>
                      <td className="px-2 py-2 text-right">
                        <Button
                          size="icon"
                          variant="ghost"
                          className="rounded-xl"
                          disabled={!printerId || busy}
                          onClick={() => reprintSession(row)}
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
