import { Global_Data } from "@/config/config"
import { Backend_Request } from "@/services/backend"

export const PLAN_PRODUCT_LINE_IDS: readonly number[] = [1, 12]
export const PLAN_AUX_LINE_IDS: readonly number[] = [20]

export const PLAN_LINES: readonly { line_id: number; name: string }[] = [
  { line_id: 20, name: "Eshik yig'uv va eshikka PPU quyish uchastkasi" },
  { line_id: 1, name: "Boshlang'ich yig'uv uchastkasi" },
  { line_id: 12, name: "Yakuniy yig'uv uchastkasi" },
]

export const YIGISH_LINE_ID = 1
export const ESHIK_LINE_ID = 20
export const QADOQLASH_LINE_ID = 12

/** Reja hisoboti / export uchun faol liniyalar (Boshlang'ich, Eshik, Yakuniy). */
export function allPlanLineIds(): number[] {
  return PLAN_LINES.map((line) => line.line_id)
}

export function isPlanProductLine(lineId: number): boolean {
  return (PLAN_PRODUCT_LINE_IDS as readonly number[]).includes(lineId)
}

export type PlanItemRow = {
  id: number
  plan_date: string
  line_id: number
  line_name?: string
  model_id: number
  component_id: number
  item_key: number
  label: string
  artikul_raqami: string
  odoo_code: string
  brend?: string
  seriya_raqami?: string
  modeli?: string
  rangi?: string
  gscode_count?: number
  shift_no: number
  planned_qty: number
  allow_overplan: boolean
  actual_qty: number
  remaining_qty: number
  completion_pct: number
}

export type CurrentShiftInfo = {
  shift_no: number
  plan_date: string
  start_time: string
  end_time: string
  label: string
}

export type PlanDayResponse = {
  plan_date: string
  line_id: number
  status: string
  shift_no?: number
  current_shift?: CurrentShiftInfo
  items: PlanItemRow[]
}

export type PlanDashboardResponse = {
  plan_date: string
  current_shift_no?: number
  current_plan_date?: string
  lines: {
    line_id: number
    line_name: string
    planned_total: number
    actual_total: number
    completion_pct: number
    models_in_plan: number
    models_complete: number
    shift1_planned?: number
    shift1_actual?: number
    shift2_planned?: number
    shift2_actual?: number
    current_shift?: number
  }[]
  items: PlanItemRow[]
}

export type PlanImportError = {
  sheet: string
  row: number
  column: string
  message: string
}

export type PlanCatalogItem = {
  item_key: number
  label: string
  odoo_code: string
  seriya_raqami: string
  full_name_uz: string
  rangi: string
  allow_overplan: boolean
}

export type PlanMonthShiftCell = {
  planned_qty: number
  actual_qty: number
}

export type PlanMonthDayCell = {
  shift1: PlanMonthShiftCell
  shift2: PlanMonthShiftCell
}

export type PlanMonthRow = {
  item_key: number
  model_id: number
  component_id: number
  label: string
  rangi: string
  allow_overplan: boolean
  days: Record<string, PlanMonthDayCell>
}

export type PlanMonthResponse = {
  year_month: string
  line_id: number
  line_name: string
  dates: string[]
  day_statuses: Record<string, string>
  rows: PlanMonthRow[]
  catalog: PlanCatalogItem[]
}

export type PlanSaveCell = {
  plan_date: string
  line_id: number
  model_id: number
  component_id: number
  shift_no: number
  planned_qty: number
  allow_overplan: boolean
}

function emptyShiftCell(): PlanMonthShiftCell {
  return { planned_qty: 0, actual_qty: 0 }
}

export function emptyDayCell(): PlanMonthDayCell {
  return { shift1: emptyShiftCell(), shift2: emptyShiftCell() }
}

export function clonePlanRows(rows: PlanMonthRow[]): PlanMonthRow[] {
  return rows.map((row) => ({
    ...row,
    days: Object.fromEntries(
      Object.entries(row.days).map(([date, cell]) => [
        date,
        {
          shift1: { ...(cell.shift1 ?? emptyShiftCell()) },
          shift2: { ...(cell.shift2 ?? emptyShiftCell()) },
        },
      ]),
    ),
  }))
}

export function buildPlanSaveCells(
  lineId: number,
  dates: string[],
  dayStatuses: Record<string, string>,
  current: PlanMonthRow[],
  initial: PlanMonthRow[],
): PlanSaveCell[] {
  const isProductLine = (PLAN_PRODUCT_LINE_IDS as readonly number[]).includes(lineId)
  const cells: PlanSaveCell[] = []
  const initialByKey = new Map(initial.map((r) => [r.item_key, r]))
  const currentKeys = new Set(current.map((r) => r.item_key))

  const rowModelID = (row: PlanMonthRow) =>
    row.model_id > 0 ? row.model_id : isProductLine ? row.item_key : 0
  const rowComponentID = (row: PlanMonthRow) =>
    row.component_id > 0 ? row.component_id : !isProductLine ? row.item_key : 0

  const pushCell = (row: PlanMonthRow, date: string, shiftNo: number, qty: number, allow: boolean) => {
    if (!isPlanDayEditable(date, dayStatuses)) {
      return
    }
    cells.push({
      plan_date: date,
      line_id: lineId,
      model_id: rowModelID(row),
      component_id: rowComponentID(row),
      shift_no: shiftNo,
      planned_qty: qty,
      allow_overplan: allow,
    })
  }

  const shiftQty = (cell: PlanMonthDayCell | undefined, shiftNo: number) => {
    if (!cell) {
      return 0
    }
    return shiftNo === 2 ? (cell.shift2?.planned_qty ?? 0) : (cell.shift1?.planned_qty ?? 0)
  }

  for (const row of current) {
    const prev = initialByKey.get(row.item_key)
    const allowChanged = prev ? prev.allow_overplan !== row.allow_overplan : false
    for (const date of dates) {
      for (const shiftNo of [1, 2]) {
        const qty = shiftQty(row.days[date], shiftNo)
        const prevQty = shiftQty(prev?.days[date], shiftNo)
        if (qty !== prevQty || (allowChanged && (qty > 0 || prevQty > 0))) {
          pushCell(row, date, shiftNo, qty, row.allow_overplan)
        }
      }
    }
  }

  for (const prev of initial) {
    if (currentKeys.has(prev.item_key)) {
      continue
    }
    for (const date of dates) {
      for (const shiftNo of [1, 2]) {
        const prevQty = shiftQty(prev.days[date], shiftNo)
        if (prevQty > 0) {
          pushCell(prev, date, shiftNo, 0, prev.allow_overplan)
        }
      }
    }
  }

  return cells
}

export async function fetchPlanMonth(lineId: number, yearMonth: string, includeActual = true) {
  return Backend_Request<PlanMonthResponse>(
    { line_id: lineId, year_month: yearMonth, include_actual: includeActual },
    "/api/production/plan/month",
  )
}

export async function savePlanCells(lineId: number, cells: PlanSaveCell[]) {
  return Backend_Request<{ saved_cells: number }>(
    { line_id: lineId, cells },
    "/api/production/plan/save",
  )
}

export function currentYearMonth() {
  const now = new Date()
  const month = String(now.getMonth() + 1).padStart(2, "0")
  return `${now.getFullYear()}-${month}`
}

export function todayDateInput() {
  const now = new Date()
  const month = String(now.getMonth() + 1).padStart(2, "0")
  const day = String(now.getDate()).padStart(2, "0")
  return `${now.getFullYear()}-${month}-${day}`
}

export function isPlanPastDate(date: string): boolean {
  return date < todayDateInput()
}

export function effectivePlanDayStatus(date: string, dayStatuses: Record<string, string>): string {
  if (isPlanPastDate(date)) {
    return "locked"
  }
  return dayStatuses[date] || "draft"
}

export function isPlanDayEditable(date: string, dayStatuses: Record<string, string>): boolean {
  return effectivePlanDayStatus(date, dayStatuses) !== "locked"
}

export async function fetchPlanDay(lineId: number, planDate?: string, opts?: { currentShift?: boolean; shiftNo?: number }) {
  const payload: Record<string, unknown> = {
    line_id: lineId,
    plan_date: planDate ?? "",
  }
  if (opts?.currentShift) {
    payload.current_shift = true
  }
  if (opts?.shiftNo && opts.shiftNo > 0) {
    payload.shift_no = opts.shiftNo
  }
  if (!opts?.currentShift && !planDate) {
    payload.plan_date = todayDateInput()
  }
  return Backend_Request<PlanDayResponse>(payload, "/api/production/plan/day")
}

export async function downloadPlanExport(yearMonth: string, includeActual: boolean) {
  const token = Global_Data.getAccessToken()
  const response = await fetch(`${Global_Data.server_ip}/api/production/plan/export`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: token } : {}),
    },
    body: JSON.stringify({ year_month: yearMonth, include_actual: includeActual }),
  })
  if (!response.ok) {
    const payload = await response.json().catch(() => null)
    throw new Error(String(payload?.error ?? "Export xatolik"))
  }
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const link = document.createElement("a")
  link.href = url
  link.download = `ishlab_chiqarish_rejasi_${yearMonth}.xlsx`
  link.click()
  URL.revokeObjectURL(url)
}

export async function getTodayPlannedItemIds(lineId: number, planDate?: string): Promise<Set<number>> {
  const result = await fetchPlanDay(lineId, planDate, planDate ? undefined : { currentShift: true })
  if (result.result !== "ok" || !result.data || result.data.status !== "locked") {
    return new Set()
  }
  return plannedIdsFromItems(result.data.items)
}

export type PlanProgressFields = {
  planned_qty: number
  actual_qty: number
  completion_pct: number
  allow_overplan: boolean
}

export const EMPTY_PLAN_PROGRESS: PlanProgressFields = {
  planned_qty: 0,
  actual_qty: 0,
  completion_pct: 0,
  allow_overplan: false,
}

export function plannedIdsFromItems(items: PlanItemRow[]): Set<number> {
  const ids = new Set<number>()
  for (const item of items) {
    const id = planItemKey(item)
    if (id > 0 && item.planned_qty > 0) {
      ids.add(id)
    }
  }
  return ids
}

export function parseLockedPlanDay(planResult: Awaited<ReturnType<typeof fetchPlanDay>>) {
  const status = planResult.result === "ok" && planResult.data ? planResult.data.status : ""
  const items =
    planResult.result === "ok" && planResult.data?.status === "locked" ? planResult.data.items : []
  return {
    status,
    items,
    planByKey: planItemsByKey(items),
    plannedIds: plannedIdsFromItems(items),
  }
}

export function planProgressFromItem(item?: PlanItemRow): PlanProgressFields {
  if (!item) {
    return EMPTY_PLAN_PROGRESS
  }
  return {
    planned_qty: item.planned_qty,
    actual_qty: item.actual_qty,
    completion_pct: item.completion_pct,
    allow_overplan: item.allow_overplan,
  }
}

export function attachPlanProgress<T>(
  item: T,
  planByKey: Map<number, PlanItemRow>,
  getId: (item: T) => number,
): T & PlanProgressFields {
  return { ...item, ...planProgressFromItem(planByKey.get(getId(item))) }
}

export function sumPlanProgress(rows: Pick<PlanProgressFields, "planned_qty" | "actual_qty">[]) {
  return rows.reduce(
    (acc, row) => ({
      planned: acc.planned + row.planned_qty,
      actual: acc.actual + row.actual_qty,
    }),
    { planned: 0, actual: 0 },
  )
}

export function planListPanelDescription(planStatus: string) {
  return planStatus === "locked"
    ? "Bugungi tasdiqlangan reja bo'yicha"
    : "Reja tasdiqlanmagan — /production/plan sahifasida kunni qulflang"
}

export function planItemKey(item: Pick<PlanItemRow, "model_id" | "component_id" | "item_key">): number {
  if (item.model_id > 0) {
    return item.model_id
  }
  if (item.component_id > 0) {
    return item.component_id
  }
  return item.item_key
}

export function planItemsByKey(items: PlanItemRow[]): Map<number, PlanItemRow> {
  const map = new Map<number, PlanItemRow>()
  for (const item of items) {
    const id = planItemKey(item)
    if (id > 0) {
      map.set(id, item)
    }
  }
  return map
}

export function filterItemsByPlannedIds<T>(items: T[], plannedIds: Set<number>, getId: (item: T) => number): T[] {
  if (plannedIds.size === 0) {
    return []
  }
  return items.filter((item) => plannedIds.has(getId(item)))
}

export async function uploadPlanImport(yearMonth: string, file: File) {
  const token = Global_Data.getAccessToken()
  const form = new FormData()
  form.append("year_month", yearMonth)
  form.append("file", file)
  const response = await fetch(`${Global_Data.server_ip}/api/production/plan/import`, {
    method: "POST",
    headers: token ? { Authorization: token } : {},
    body: form,
  })
  const payload = await response.json()
  if (!response.ok || payload.result === "error") {
    return {
      result: "error" as const,
      error: String(payload.error ?? "Import xatolik"),
      data: payload.data as { errors?: PlanImportError[] } | undefined,
    }
  }
  return { result: "ok" as const, data: payload.data }
}
