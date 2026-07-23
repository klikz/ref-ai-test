import { useCallback, useEffect, useMemo, useRef, useState, Fragment } from "react"
import { Lock, Plus, RefreshCw, Save, Trash2, Unlock } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import { cn } from "@/lib/utils"
import {
  PLAN_AUX_LINE_IDS,
  PLAN_LINES,
  PLAN_PRODUCT_LINE_IDS,
  YIGISH_LINE_ID,
  effectivePlanDayStatus,
  emptyDayCell,
  isPlanDayEditable,
  isPlanPastDate,
  type PlanCatalogItem,
  type PlanMonthResponse,
  type PlanMonthRow,
  buildPlanSaveCells,
  clonePlanRows,
  fetchPlanMonth,
  savePlanCells,
} from "@/pages/production_plan_shared"

type Props = {
  yearMonth: string
}

function formatDayHeader(date: string) {
  const [, , day] = date.split("-")
  return day
}

function getShiftPlannedQty(row: PlanMonthRow, date: string, shiftNo: 1 | 2) {
  const cell = row.days[date]
  if (!cell) {
    return 0
  }
  return shiftNo === 2 ? (cell.shift2?.planned_qty ?? 0) : (cell.shift1?.planned_qty ?? 0)
}

function getShiftActualQty(row: PlanMonthRow, date: string, shiftNo: 1 | 2) {
  const cell = row.days[date]
  if (!cell) {
    return 0
  }
  return shiftNo === 2 ? (cell.shift2?.actual_qty ?? 0) : (cell.shift1?.actual_qty ?? 0)
}

const planQtyColClass = "w-[3.25rem] min-w-[3.25rem] max-w-[3.25rem]"
const planQtyInputClass =
  "h-8 w-full min-w-0 rounded-none border-0 bg-transparent px-0.5 text-center text-sm tabular-nums shadow-none focus-visible:ring-1 [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"

function formatCatalogOption(item: PlanCatalogItem, lineId: number) {
  const parts = [item.label]
  const rangi = item.rangi?.trim()
  if ((PLAN_PRODUCT_LINE_IDS as readonly number[]).includes(lineId) && rangi) {
    parts.push(rangi)
  }
  const fullName = item.full_name_uz?.trim()
  if (PLAN_AUX_LINE_IDS.includes(lineId as (typeof PLAN_AUX_LINE_IDS)[number]) && fullName) {
    parts.push(fullName)
  }
  const seriya = item.seriya_raqami?.trim()
  const odoo = item.odoo_code?.trim()
  if (seriya) {
    parts.push(seriya)
  }
  if (odoo) {
    parts.push(odoo)
  }
  return parts.join(" — ")
}

export function ProductionPlanMonthGrid({ yearMonth }: Props) {
  const [lineId, setLineId] = useState(YIGISH_LINE_ID)
  const [data, setData] = useState<PlanMonthResponse | null>(null)
  const [rows, setRows] = useState<PlanMonthRow[]>([])
  const snapshotRef = useRef<PlanMonthRow[]>([])
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [locking, setLocking] = useState(false)
  const [addItemKey, setAddItemKey] = useState("")

  const loadMonth = useCallback(async () => {
    setLoading(true)
    const result = await fetchPlanMonth(lineId, yearMonth, true)
    setLoading(false)
    if (result.result === "ok" && result.data) {
      setData(result.data)
      const cloned = clonePlanRows(result.data.rows)
      setRows(cloned)
      snapshotRef.current = clonePlanRows(result.data.rows)
    } else {
      setData(null)
      setRows([])
      snapshotRef.current = []
      ShowErrorToast(result.error || "Reja yuklanmadi")
    }
  }, [lineId, yearMonth])

  useEffect(() => {
    void loadMonth()
  }, [loadMonth])

  const catalogAvailable = useMemo(() => {
    if (!data) {
      return []
    }
    const used = new Set(rows.map((r) => r.item_key))
    return data.catalog.filter((c) => !used.has(c.item_key))
  }, [data, rows])

  const isDirty = useMemo(() => {
    if (!data) {
      return false
    }
    const cells = buildPlanSaveCells(data.line_id, data.dates, data.day_statuses, rows, snapshotRef.current)
    return cells.length > 0
  }, [data, rows])

  function updatePlannedQty(itemKey: number, date: string, shiftNo: 1 | 2, raw: string) {
    const qty = Math.max(0, Number.parseInt(raw, 10) || 0)
    setRows((prev) =>
      prev.map((row) => {
        if (row.item_key !== itemKey) {
          return row
        }
        const days = { ...row.days }
        const prevCell = days[date] ?? emptyDayCell()
        const nextCell = {
          shift1: { ...(prevCell.shift1 ?? { planned_qty: 0, actual_qty: 0 }) },
          shift2: { ...(prevCell.shift2 ?? { planned_qty: 0, actual_qty: 0 }) },
        }
        if (shiftNo === 2) {
          nextCell.shift2 = { ...nextCell.shift2, planned_qty: qty }
        } else {
          nextCell.shift1 = { ...nextCell.shift1, planned_qty: qty }
        }
        days[date] = nextCell
        return { ...row, days }
      }),
    )
  }

  function updateAllowOverplan(itemKey: number, allow: boolean) {
    setRows((prev) => prev.map((row) => (row.item_key === itemKey ? { ...row, allow_overplan: allow } : row)))
  }

  function addRow(catalogItem: PlanCatalogItem) {
    const isProduct = (PLAN_PRODUCT_LINE_IDS as readonly number[]).includes(lineId)
    setRows((prev) => [
      ...prev,
      {
        item_key: catalogItem.item_key,
        model_id: isProduct ? catalogItem.item_key : 0,
        component_id: isProduct ? 0 : catalogItem.item_key,
        label: catalogItem.label,
        rangi: catalogItem.rangi ?? "",
        allow_overplan: true,
        days: {},
      },
    ])
    setAddItemKey("")
  }

  function removeRow(itemKey: number) {
    setRows((prev) => prev.filter((r) => r.item_key !== itemKey))
  }

  async function ensureSaved(): Promise<boolean> {
    if (!data) {
      return false
    }
    if (!isDirty) {
      return true
    }
    const cells = buildPlanSaveCells(data.line_id, data.dates, data.day_statuses, rows, snapshotRef.current)
    if (cells.length === 0) {
      ShowErrorToast("Saqlash uchun o'zgarish topilmadi")
      return false
    }
    setSaving(true)
    const result = await savePlanCells(data.line_id, cells)
    setSaving(false)
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Saqlash xatolik")
      return false
    }
    snapshotRef.current = clonePlanRows(rows)
    await loadMonth()
    return true
  }

  async function handleSave() {
    if (!data) {
      return
    }
    const cells = buildPlanSaveCells(data.line_id, data.dates, data.day_statuses, rows, snapshotRef.current)
    if (cells.length === 0) {
      ShowOKToast("O'zgarish yo'q")
      return
    }
    setSaving(true)
    const result = await savePlanCells(data.line_id, cells)
    setSaving(false)
    if (result.result === "ok") {
      ShowOKToast(`Saqlandi: ${result.data?.saved_cells ?? cells.length} ta katak`)
      snapshotRef.current = clonePlanRows(rows)
      void loadMonth()
    } else {
      ShowErrorToast(result.error || "Saqlash xatolik")
    }
  }

  async function lockDay(planDate: string) {
    if (!(await ensureSaved())) {
      return
    }
    setLocking(true)
    const result = await Backend_Request<unknown>(
      { line_id: lineId, plan_date: planDate },
      "/api/production/plan/lock-day",
    )
    setLocking(false)
    if (result.result === "ok") {
      ShowOKToast(`${planDate} tasdiqlandi`)
      void loadMonth()
    } else {
      ShowErrorToast(result.error || "Tasdiqlash xatolik")
    }
  }

  async function lockMonth() {
    if (!(await ensureSaved())) {
      return
    }
    setLocking(true)
    const result = await Backend_Request<{ locked_days: number }>(
      { line_id: lineId, year_month: yearMonth },
      "/api/production/plan/lock-month",
    )
    setLocking(false)
    if (result.result === "ok") {
      ShowOKToast(`Oy tasdiqlandi: ${result.data?.locked_days ?? 0} kun`)
      void loadMonth()
    } else {
      ShowErrorToast(result.error || "Oy tasdiqlash xatolik")
    }
  }

  async function unlockDay(planDate: string) {
    setLocking(true)
    const result = await Backend_Request<unknown>(
      { line_id: lineId, plan_date: planDate },
      "/api/production/plan/unlock-day",
    )
    setLocking(false)
    if (result.result === "ok") {
      ShowOKToast(`${planDate} qayta ochildi — tahrirlash mumkin`)
      void loadMonth()
    } else {
      ShowErrorToast(result.error || "Ochish xatolik")
    }
  }

  async function unlockMonth() {
    setLocking(true)
    const result = await Backend_Request<{ unlocked_days: number }>(
      { line_id: lineId, year_month: yearMonth },
      "/api/production/plan/unlock-month",
    )
    setLocking(false)
    if (result.result === "ok") {
      ShowOKToast(`Oy ochildi: ${result.data?.unlocked_days ?? 0} kun`)
      void loadMonth()
    } else {
      ShowErrorToast(result.error || "Oyni ochish xatolik")
    }
  }

  const lineName = data?.line_name ?? PLAN_LINES.find((l) => l.line_id === lineId)?.name ?? ""

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div className="space-y-2">
          <Label>Liniya</Label>
          <div className="flex flex-wrap gap-2">
            {PLAN_LINES.map((line) => (
              <Button
                key={line.line_id}
                type="button"
                size="sm"
                variant={lineId === line.line_id ? "default" : "outline"}
                onClick={() => setLineId(line.line_id)}
              >
                {line.name}
              </Button>
            ))}
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" onClick={() => void loadMonth()} disabled={loading}>
            <RefreshCw className={cn("size-4", loading && "animate-spin")} />
          </Button>
          <Button size="sm" onClick={() => void handleSave()} disabled={saving || !isDirty}>
            <Save className="size-4" />
            {saving ? "Saqlanmoqda..." : "Saqlash"}
          </Button>
          <Button size="sm" variant="outline" disabled={locking} onClick={() => void lockMonth()}>
            <Lock className="size-4" />
            Oyni tasdiqlash
          </Button>
          <Button size="sm" variant="outline" disabled={locking} onClick={() => void unlockMonth()}>
            <Unlock className="size-4" />
            Oyni ochish
          </Button>
        </div>
      </div>

      <div className="flex flex-wrap items-end gap-3 rounded-xl border border-border/60 bg-muted/30 p-3">
        <div className="min-w-[12rem] flex-1 space-y-1">
          <Label htmlFor="plan-add-item">Qator qo&apos;shish</Label>
          <select
            id="plan-add-item"
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={addItemKey}
            onChange={(e) => setAddItemKey(e.target.value)}
          >
            <option value="">
        {(PLAN_PRODUCT_LINE_IDS as readonly number[]).includes(lineId)
                ? "Model tanlang..."
                : "Komponent tanlang..."}
            </option>
            {catalogAvailable.map((item) => (
              <option key={item.item_key} value={String(item.item_key)}>
                {formatCatalogOption(item, lineId)}
              </option>
            ))}
          </select>
        </div>
        <Button
          type="button"
          variant="secondary"
          disabled={!addItemKey}
          onClick={() => {
            const item = catalogAvailable.find((c) => String(c.item_key) === addItemKey)
            if (item) {
              addRow(item)
            }
          }}
        >
          <Plus className="size-4" />
          Qo&apos;shish
        </Button>
      </div>

      <div className="overflow-x-auto rounded-xl border border-border/60">
        <table className="w-max min-w-full border-collapse text-sm">
          <thead>
            <tr>
              <th
                colSpan={3}
                className="sticky left-0 z-20 border-b border-r bg-slate-200 px-2 py-2 text-left font-medium text-slate-700"
              >
                {lineName} — {yearMonth}
              </th>
              {data?.dates.map((date) => {
                const status = effectivePlanDayStatus(date, data.day_statuses)
                const locked = status === "locked"
                const past = isPlanPastDate(date)
                return (
                  <th
                    key={date}
                    colSpan={4}
                    className={cn(
                      "border-b border-r px-1 py-1 text-center text-xs font-semibold text-white",
                      past ? "bg-slate-600" : locked ? "bg-emerald-700" : "bg-cyan-600",
                    )}
                  >
                    <div className="flex flex-col items-center gap-0.5">
                      <span>{formatDayHeader(date)}</span>
                      {past ? (
                        <span title="O'tgan kun">
                          <Lock className="mx-auto size-3 opacity-70" />
                        </span>
                      ) : !locked ? (
                        <button
                          type="button"
                          className="rounded px-1 text-[10px] text-cyan-100 hover:bg-cyan-700"
                          disabled={locking}
                          onClick={() => void lockDay(date)}
                          title={date}
                        >
                          <Lock className="mx-auto size-3" />
                        </button>
                      ) : (
                        <button
                          type="button"
                          className="rounded px-1 text-[10px] text-emerald-100 hover:bg-emerald-800"
                          disabled={locking}
                          onClick={() => void unlockDay(date)}
                          title={`${date} — qayta ochish`}
                        >
                          <Unlock className="mx-auto size-3" />
                        </button>
                      )}
                    </div>
                  </th>
                )
              })}
            </tr>
            <tr>
              <th className="sticky left-0 z-20 min-w-[10rem] border-b border-r bg-slate-100 px-2 py-1 text-left text-xs font-medium">
                Model / Komponent
              </th>
              <th className="sticky left-[10rem] z-20 w-20 min-w-[5rem] border-b border-r bg-slate-100 px-2 py-1 text-left text-xs font-medium">
                Rangi
              </th>
              <th className="sticky left-[15rem] z-20 w-16 border-b border-r bg-slate-100 px-1 py-1 text-center text-[10px] font-medium">
                Qayta
              </th>
              {data?.dates.map((date) => (
                <Fragment key={date}>
                  <th className={cn("border-b border-r bg-emerald-100 px-0.5 py-1 text-center text-[10px] font-semibold text-emerald-800", planQtyColClass)}>
                    1S
                  </th>
                  <th className={cn("border-b border-r bg-slate-100 px-0.5 py-1 text-center text-[10px] font-semibold text-slate-600", planQtyColClass)}>
                    1F
                  </th>
                  <th className={cn("border-b border-r bg-sky-100 px-0.5 py-1 text-center text-[10px] font-semibold text-sky-800", planQtyColClass)}>
                    2S
                  </th>
                  <th className={cn("border-b border-r bg-slate-100 px-0.5 py-1 text-center text-[10px] font-semibold text-slate-600", planQtyColClass)}>
                    2F
                  </th>
                </Fragment>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, rowIndex) => {
              const rowBg = rowIndex % 2 === 0 ? "bg-background" : "bg-muted/20"
              return (
              <tr key={row.item_key} className={cn("group", rowBg)}>
                <td className={cn("sticky left-0 z-10 border-b border-r px-2 py-1 font-medium", rowBg)}>
                  <div className="flex items-center justify-between gap-2">
                    <span className="truncate">{row.label}</span>
                    <button
                      type="button"
                      className="shrink-0 rounded p-1 text-red-600 opacity-60 hover:bg-red-50 hover:opacity-100"
                      title="Qatorni o'chirish"
                      onClick={() => removeRow(row.item_key)}
                    >
                      <Trash2 className="size-3.5" />
                    </button>
                  </div>
                </td>
                <td className={cn("sticky left-[10rem] z-10 border-b border-r px-2 py-1 text-muted-foreground", rowBg)}>
                  <span className="truncate">{row.rangi || ""}</span>
                </td>
                <td className={cn("sticky left-[15rem] z-10 border-b border-r px-1 py-1 text-center", rowBg)}>
                  <input
                    type="checkbox"
                    checked={row.allow_overplan}
                    onChange={(e) => updateAllowOverplan(row.item_key, e.target.checked)}
                    title="Qo'shimcha reja ruxsat"
                  />
                </td>
                {data?.dates.map((date) => {
                  const locked = !isPlanDayEditable(date, data.day_statuses)
                  const planned1 = getShiftPlannedQty(row, date, 1)
                  const actual1 = getShiftActualQty(row, date, 1)
                  const planned2 = getShiftPlannedQty(row, date, 2)
                  const actual2 = getShiftActualQty(row, date, 2)
                  return (
                    <Fragment key={`${row.item_key}-${date}`}>
                      <td className={cn("border-b border-r p-0", planQtyColClass, rowBg)}>
                        <Input
                          type="number"
                          min={0}
                          value={planned1 || ""}
                          disabled={locked}
                          onChange={(e) => updatePlannedQty(row.item_key, date, 1, e.target.value)}
                          className={cn(
                            planQtyInputClass,
                            locked && "cursor-not-allowed opacity-60",
                          )}
                          title="1-sm reja"
                        />
                      </td>
                      <td
                        className={cn(
                          "border-b border-r px-0.5 py-1 text-center text-sm tabular-nums text-slate-600",
                          planQtyColClass,
                          rowBg,
                        )}
                        title="1-sm fakt"
                      >
                        {actual1 > 0 ? actual1 : ""}
                      </td>
                      <td className={cn("border-b border-r p-0", planQtyColClass, rowBg)}>
                        <Input
                          type="number"
                          min={0}
                          value={planned2 || ""}
                          disabled={locked}
                          onChange={(e) => updatePlannedQty(row.item_key, date, 2, e.target.value)}
                          className={cn(
                            planQtyInputClass,
                            locked && "cursor-not-allowed opacity-60",
                          )}
                          title="2-sm reja"
                        />
                      </td>
                      <td
                        className={cn(
                          "border-b border-r px-0.5 py-1 text-center text-sm tabular-nums text-slate-600",
                          planQtyColClass,
                          rowBg,
                        )}
                        title="2-sm fakt"
                      >
                        {actual2 > 0 ? actual2 : ""}
                      </td>
                    </Fragment>
                  )
                })}
              </tr>
            )})}
            {!loading && rows.length === 0 ? (
              <tr>
                <td
                  colSpan={(data?.dates.length ?? 0) * 2 + 3}
                  className="h-24 text-center text-muted-foreground"
                >
                  Reja bo&apos;sh — qator qo&apos;shing yoki Excel import qiling
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>

      <p className="text-xs text-muted-foreground">
        Excel ko&apos;rinishi: kunlar gorizontal, modellar vertikal. Yashil — Reja (tahrirlash mumkin), kulrang —
        Fakt. O&apos;tgan kunlar avtomatik bloklangan — faqat bugun va kelajak tahrirlanadi. Tasdiqlangan kunlarni{" "}
        <Unlock className="inline size-3" /> bilan qayta ochish mumkin (faqat bugun va kelajak).
        {PLAN_AUX_LINE_IDS.includes(lineId as (typeof PLAN_AUX_LINE_IDS)[number])
          ? " Komponentlar ro'yxati liniya sozlamalaridan."
          : " T1/T2/T3 — seriya raqami T bilan tugaydi; Ichki — I bilan tugaydi."}
      </p>
    </div>
  )
}
