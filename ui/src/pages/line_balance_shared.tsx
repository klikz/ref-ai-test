import { ChevronLeft, ChevronRight } from "lucide-react"
import {
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  useReactTable,
  type Table as ReactTable,
} from "@tanstack/react-table"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { ShowErrorToast } from "@/components/showToast"
import { Backend_Request } from "@/services/backend"
import { cn } from "@/lib/utils"
import { useEffect, useState } from "react"

export type Line = {
  line_id: number
  name: string
}

export type BalanceRow = {
  id: number
  main_code: string
  odoo_code: string
  component_id: number
  full_name_uz: string
  quantity: number
}

export type BalanceTransaction = {
  id: number
  line_id: number
  line_name: string
  component_id: number
  manufacturer_code: string
  standard_name_uz: string
  odoo_code: string
  quantity_change: number
  quantity_after: number
  user_id: number
  user_name: string
  user_login: string
  source: string
  comment: string
  created_at: string
}

export const DEFAULT_PAGE_SIZE = 50
export const PAGE_SIZE_OPTIONS = [25, 50, 100, 200] as const

/** Product assembly lines — no component balance in this screen. */
export const LINE_BALANCE_EXCLUDED_LINE_IDS = [4, 5, 6, 7] as const

export const balanceColumnHelper = createColumnHelper<BalanceRow>()
export const txColumnHelper = createColumnHelper<BalanceTransaction>()

function pickDefaultLine(lines: Line[], preferredLineId?: number | null): Line {
  if (preferredLineId) {
    const preferred = lines.find((line) => line.line_id === preferredLineId)
    if (preferred) {
      return preferred
    }
  }
  const finPress = lines.find((line) => /fin\s*press/i.test(line.name))
  return finPress ?? lines[0]
}

function normalizeLineRows(data: unknown): Line[] {
  if (!Array.isArray(data)) {
    return []
  }
  return data
    .map((row) => ({
      line_id: Number((row as Line)?.line_id ?? (row as { id?: number })?.id ?? 0),
      name: String((row as Line)?.name ?? ""),
    }))
    .filter((line) => line.line_id > 0 && line.name)
}

export function normalizeBalanceRows(data: unknown): BalanceRow[] {
  if (Array.isArray(data)) {
    return data.map((row, index) => mapBalanceRow(row, index))
  }
  if (data && typeof data === "object") {
    const wrapped = data as { data?: unknown }
    if (Array.isArray(wrapped.data)) {
      return wrapped.data.map((row, index) => mapBalanceRow(row, index))
    }
  }
  return []
}

function mapBalanceRow(row: unknown, index: number): BalanceRow {
  const item = row as Record<string, unknown>
  return {
    id: Number(item.id ?? index + 1),
    main_code: String(item.main_code ?? item.factory_code ?? item.manufacturer_code ?? ""),
    odoo_code: String(item.odoo_code ?? ""),
    component_id: Number(item.component_id ?? item.componentId ?? 0),
    full_name_uz: String(item.full_name_uz ?? ""),
    quantity: Number(item.quantity ?? 0),
  }
}

export function normalizeBalanceTransactions(data: unknown): BalanceTransaction[] {
  if (!Array.isArray(data)) {
    return []
  }
  return data.map((row, index) => {
    const item = row as Partial<BalanceTransaction>
    return {
      id: Number(item.id ?? index + 1),
      line_id: Number(item.line_id ?? 0),
      line_name: String(item.line_name ?? ""),
      component_id: Number(item.component_id ?? 0),
      manufacturer_code: String(item.manufacturer_code ?? ""),
      standard_name_uz: String(item.standard_name_uz ?? ""),
      odoo_code: String(item.odoo_code ?? ""),
      quantity_change: Number(item.quantity_change ?? 0),
      quantity_after: Number(item.quantity_after ?? 0),
      user_id: Number(item.user_id ?? 0),
      user_name: String(item.user_name ?? ""),
      user_login: String(item.user_login ?? ""),
      source: String(item.source ?? ""),
      comment: String(item.comment ?? ""),
      created_at: String(item.created_at ?? ""),
    }
  })
}

export function toDateInputValue(date: Date) {
  return date.toISOString().slice(0, 10)
}

export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement("a")
  link.href = url
  link.download = filename
  link.rel = "noopener"
  document.body.appendChild(link)
  link.click()
  link.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

export function matchesComponentFilter(values: unknown[], filterValue: string) {
  const query = filterValue.trim().toLocaleLowerCase()
  if (!query) {
    return true
  }
  return values
    .map((value) => String(value ?? "").toLocaleLowerCase())
    .join(" ")
    .includes(query)
}

function sourceLabel(source: string) {
  if (source === "manual") {
    return "Qo'lda"
  }
  if (source === "fin_press_print") {
    return "Fin press"
  }
  if (source === "radiator_print") {
    return "Radiator chiqish"
  }
  return source
}

const COLORS = {
  titleBg: "FF0F172A",
  titleText: "FFFFFFFF",
  metaBg: "FFE2E8F0",
  headerBg: "FF0891B2",
  headerText: "FFFFFFFF",
  zebra: "FFF8FAFC",
  border: "FFCBD5E1",
  positive: "FF166534",
  positiveBg: "FFDCFCE7",
  negative: "FFB91C1C",
  negativeBg: "FFFEE2E2",
  summaryBg: "FFEFF6FF",
}

const balanceHeaders = ["Korxona kodi", "ODOO code", "Komponent nomi", "Balans"]
const transactionHeaders = [
  "Vaqt",
  "Liniya",
  "Manufacturer code",
  "ODOO code",
  "Standard name uz",
  "O'zgarish",
  "Balans keyin",
  "Foydalanuvchi",
  "Login",
  "Manba",
  "Izoh",
]

function applyThinBorder(cell: { border?: object }) {
  cell.border = {
    top: { style: "thin", color: { argb: COLORS.border } },
    left: { style: "thin", color: { argb: COLORS.border } },
    bottom: { style: "thin", color: { argb: COLORS.border } },
    right: { style: "thin", color: { argb: COLORS.border } },
  }
}

function styleTitleRow(worksheet: import("exceljs").Worksheet, title: string, subtitle: string, columnCount: number) {
  worksheet.mergeCells(1, 1, 1, columnCount)
  const titleCell = worksheet.getCell(1, 1)
  titleCell.value = title
  titleCell.font = { bold: true, size: 16, color: { argb: COLORS.titleText } }
  titleCell.alignment = { vertical: "middle", horizontal: "center" }
  titleCell.fill = {
    type: "pattern",
    pattern: "solid",
    fgColor: { argb: COLORS.titleBg },
  }
  worksheet.getRow(1).height = 30

  worksheet.mergeCells(2, 1, 2, columnCount)
  const subtitleCell = worksheet.getCell(2, 1)
  subtitleCell.value = subtitle
  subtitleCell.font = { size: 11, color: { argb: "FF334155" } }
  subtitleCell.alignment = { vertical: "middle", horizontal: "center" }
  subtitleCell.fill = {
    type: "pattern",
    pattern: "solid",
    fgColor: { argb: COLORS.metaBg },
  }
  worksheet.getRow(2).height = 22
}

function styleHeaderRow(worksheet: import("exceljs").Worksheet, rowNumber: number, headers: string[]) {
  const row = worksheet.getRow(rowNumber)
  headers.forEach((header, index) => {
    const cell = row.getCell(index + 1)
    cell.value = header
    cell.font = { bold: true, color: { argb: COLORS.headerText } }
    cell.alignment = { vertical: "middle", horizontal: "center", wrapText: true }
    cell.fill = {
      type: "pattern",
      pattern: "solid",
      fgColor: { argb: COLORS.headerBg },
    }
    applyThinBorder(cell)
  })
  row.height = 24
}

function autosizeColumns(worksheet: import("exceljs").Worksheet, widths: number[]) {
  widths.forEach((width, index) => {
    const column = worksheet.getColumn(index + 1)
    column.width = width
  })
}

export async function buildBalanceWorkbook(params: {
  lineName: string
  dateFrom: string
  dateTo: string
  balances: BalanceRow[]
  transactions: BalanceTransaction[]
}) {
  const ExcelJS = await import("exceljs")
  const workbook = new ExcelJS.Workbook()
  workbook.creator = "Premier AC"
  workbook.created = new Date()

  const balanceSheet = workbook.addWorksheet("Joriy balans")
  styleTitleRow(
    balanceSheet,
    "Liniya komponent balansi",
    `Liniya: ${params.lineName}`,
    balanceHeaders.length,
  )
  styleHeaderRow(balanceSheet, 4, balanceHeaders)

  params.balances.forEach((item, index) => {
    const rowNumber = 5 + index
    const row = balanceSheet.getRow(rowNumber)
    row.getCell(1).value = item.main_code
    row.getCell(2).value = item.odoo_code
    row.getCell(3).value = item.full_name_uz
    row.getCell(4).value = Number(item.quantity)

    row.eachCell((cell) => {
      cell.alignment = { vertical: "middle", horizontal: "center" }
      applyThinBorder(cell)
      if (index % 2 === 1) {
        cell.fill = {
          type: "pattern",
          pattern: "solid",
          fgColor: { argb: COLORS.zebra },
        }
      }
    })
    row.getCell(4).numFmt = "#,##0.####"
  })

  const balanceTotalRow = balanceSheet.getRow(5 + params.balances.length + 1)
  balanceSheet.mergeCells(5 + params.balances.length + 1, 1, 5 + params.balances.length + 1, 3)
  balanceTotalRow.getCell(1).value = "Jami pozitsiyalar"
  balanceTotalRow.getCell(4).value = params.balances.length
  balanceTotalRow.eachCell((cell) => {
    cell.font = { bold: true }
    cell.fill = {
      type: "pattern",
      pattern: "solid",
      fgColor: { argb: COLORS.summaryBg },
    }
    applyThinBorder(cell)
  })

  balanceSheet.views = [{ state: "frozen", ySplit: 4 }]
  autosizeColumns(balanceSheet, [24, 18, 36, 14])

  const txSheet = workbook.addWorksheet("Tranzaksiyalar")
  styleTitleRow(
    txSheet,
    "Balans tranzaksiyalari",
    `Liniya: ${params.lineName} | Davr: ${params.dateFrom} — ${params.dateTo}`,
    transactionHeaders.length,
  )
  styleHeaderRow(txSheet, 4, transactionHeaders)

  let totalAdded = 0
  let totalRemoved = 0

  params.transactions.forEach((item, index) => {
    const rowNumber = 5 + index
    const row = txSheet.getRow(rowNumber)
    const change = Number(item.quantity_change)
    if (change > 0) {
      totalAdded += change
    } else {
      totalRemoved += Math.abs(change)
    }

    row.getCell(1).value = item.created_at
    row.getCell(2).value = item.line_name
    row.getCell(3).value = item.manufacturer_code
    row.getCell(4).value = item.odoo_code
    row.getCell(5).value = item.standard_name_uz
    row.getCell(6).value = change
    row.getCell(7).value = Number(item.quantity_after)
    row.getCell(8).value = item.user_name || "-"
    row.getCell(9).value = item.user_login || "-"
    row.getCell(10).value = sourceLabel(item.source)
    row.getCell(11).value = item.comment || ""

    row.eachCell((cell, colNumber) => {
      cell.alignment = {
        vertical: "middle",
        horizontal: colNumber >= 6 && colNumber <= 7 ? "center" : "left",
        wrapText: colNumber === 11,
      }
      applyThinBorder(cell)
      if (index % 2 === 1) {
        cell.fill = {
          type: "pattern",
          pattern: "solid",
          fgColor: { argb: COLORS.zebra },
        }
      }
    })

    const changeCell = row.getCell(6)
    changeCell.numFmt = change > 0 ? "+#,##0.####;#,##0.####" : "#,##0.####"
    if (change > 0) {
      changeCell.font = { bold: true, color: { argb: COLORS.positive } }
      changeCell.fill = {
        type: "pattern",
        pattern: "solid",
        fgColor: { argb: COLORS.positiveBg },
      }
    } else if (change < 0) {
      changeCell.font = { bold: true, color: { argb: COLORS.negative } }
      changeCell.fill = {
        type: "pattern",
        pattern: "solid",
        fgColor: { argb: COLORS.negativeBg },
      }
    }

    row.getCell(7).numFmt = "#,##0.####"
  })

  const summaryRowNumber = 5 + params.transactions.length + 1
  const summaryRow = txSheet.getRow(summaryRowNumber)
  txSheet.mergeCells(summaryRowNumber, 1, summaryRowNumber, 5)
  summaryRow.getCell(1).value = "Jami qo'shilgan / ayirilgan"
  summaryRow.getCell(6).value = totalAdded
  summaryRow.getCell(7).value = -totalRemoved
  summaryRow.eachCell((cell, colNumber) => {
    cell.font = { bold: true }
    cell.fill = {
      type: "pattern",
      pattern: "solid",
      fgColor: { argb: COLORS.summaryBg },
    }
    applyThinBorder(cell)
    if (colNumber === 6) {
      cell.font = { bold: true, color: { argb: COLORS.positive } }
      cell.numFmt = "+#,##0.####"
    }
    if (colNumber === 7) {
      cell.font = { bold: true, color: { argb: COLORS.negative } }
      cell.numFmt = "#,##0.####"
    }
  })

  txSheet.views = [{ state: "frozen", ySplit: 4 }]
  autosizeColumns(txSheet, [20, 14, 22, 18, 28, 14, 14, 18, 16, 14, 30])

  const excelBuffer = await workbook.xlsx.writeBuffer()
  return new Blob([excelBuffer], {
    type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  })
}

export function TablePaginationControls<T>({
  table,
  controlId,
  pageSize,
  onPageSizeChange,
}: {
  table: ReactTable<T>
  controlId: string
  pageSize: number
  onPageSizeChange: (size: number) => void
}) {
  const filteredCount = table.getFilteredRowModel().rows.length
  const pageIndex = table.getState().pagination.pageIndex
  const pageCount = table.getPageCount()

  if (filteredCount === 0) {
    return null
  }

  const from = pageIndex * pageSize + 1
  const to = Math.min((pageIndex + 1) * pageSize, filteredCount)

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border/50 pt-3">
      <span className="text-sm text-muted-foreground">
        {from}–{to} / {filteredCount} ta
      </span>
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-2">
          <Label htmlFor={`page-size-${controlId}`} className="text-sm text-muted-foreground">
            Sahifada
          </Label>
          <select
            id={`page-size-${controlId}`}
            value={pageSize}
            onChange={(event) => onPageSizeChange(Number(event.target.value))}
            className={cn(
              "h-9 rounded-lg border border-input bg-background px-2.5 text-sm font-medium shadow-sm",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            )}
          >
            {PAGE_SIZE_OPTIONS.map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            className="h-9 gap-1"
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage()}
          >
            <ChevronLeft className="size-4" />
            Oldingi
          </Button>
          <span className="min-w-16 text-center text-sm font-medium">
            {pageIndex + 1} / {Math.max(pageCount, 1)}
          </span>
          <Button
            variant="outline"
            size="sm"
            className="h-9 gap-1"
            onClick={() => table.nextPage()}
            disabled={!table.getCanNextPage()}
          >
            Keyingi
            <ChevronRight className="size-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}

export function useLineBalanceLines(initialLineId?: number | null) {
  const [lines, setLines] = useState<Line[]>([])
  const [selectedLine, setSelectedLine] = useState<Line | null>(null)
  const [linesLoading, setLinesLoading] = useState(true)

  useEffect(() => {
    void (async () => {
      setLinesLoading(true)
      const result = await Backend_Request<Line[]>({}, "/api/lines/all")
      setLinesLoading(false)
      if (result.result !== "ok") {
        ShowErrorToast(result.error || "Xatolik")
        return
      }
      const excluded = new Set<number>(LINE_BALANCE_EXCLUDED_LINE_IDS)
      const data = normalizeLineRows(result.data).filter((line) => !excluded.has(line.line_id))
      setLines(data)
    })()
  }, [])

  useEffect(() => {
    if (linesLoading) {
      return
    }
    if (lines.length === 0) {
      setSelectedLine(null)
      return
    }
    setSelectedLine((current) => {
      if (current && lines.some((line) => line.line_id === current.line_id)) {
        return current
      }
      return pickDefaultLine(lines, initialLineId)
    })
  }, [initialLineId, lines, linesLoading])

  return { lines, selectedLine, setSelectedLine, linesLoading }
}

export {
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  useReactTable,
}
