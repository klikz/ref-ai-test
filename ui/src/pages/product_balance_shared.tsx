import { ShowErrorToast } from "@/components/showToast"
import { Backend_Request } from "@/services/backend"
import { useEffect, useState } from "react"

/** Yi'g'ish, T1–T3, Qadoqlash — mahsulot (serial) balansi. */
export const PRODUCT_LINE_IDS = [1, 4, 5, 6, 7, 12] as const

export type Line = {
  line_id: number
  name: string
}

export type ProductBalanceItem = {
  id: number
  serial: string
  model_id: number
  model_name: string
  modeli: string
  odoo_code: string
  time: string
}

export type ProductBalanceSummaryRow = {
  model_id: number
  model_name: string
  modeli: string
  odoo_code: string
  count: number
}

export type ProductTransferReportRow = {
  id: number
  serial: string
  line_id: number
  line_name: string
  model_id: number
  model_name: string
  modeli: string
  odoo_code: string
  transferred_at: string
  user_name: string
}

export type ProductBalanceResponse = {
  summary: ProductBalanceSummaryRow[]
  items: ProductBalanceItem[]
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

function pickDefaultProductLine(lines: Line[], preferredLineId?: number | null): Line | null {
  if (lines.length === 0) {
    return null
  }
  if (preferredLineId) {
    const preferred = lines.find((line) => line.line_id === preferredLineId)
    if (preferred) {
      return preferred
    }
  }
  return lines.find((line) => line.line_id === PRODUCT_LINE_IDS[0]) ?? lines[0]
}

export function normalizeProductBalance(data: unknown): ProductBalanceResponse {
  const empty: ProductBalanceResponse = { summary: [], items: [] }
  if (!data || typeof data !== "object") {
    return empty
  }

  const payload = data as Record<string, unknown>
  const summaryRaw = Array.isArray(payload.summary) ? payload.summary : []
  const itemsRaw = Array.isArray(payload.items) ? payload.items : []

  return {
    summary: summaryRaw.map((row) => {
      const item = row as Record<string, unknown>
      return {
        model_id: Number(item.model_id ?? 0),
        model_name: String(item.model_name ?? ""),
        modeli: String(item.modeli ?? ""),
        odoo_code: String(item.odoo_code ?? ""),
        count: Number(item.count ?? 0),
      }
    }),
    items: itemsRaw.map((row, index) => {
      const item = row as Record<string, unknown>
      return {
        id: Number(item.id ?? index + 1),
        serial: String(item.serial ?? ""),
        model_id: Number(item.model_id ?? 0),
        model_name: String(item.model_name ?? ""),
        modeli: String(item.modeli ?? ""),
        odoo_code: String(item.odoo_code ?? ""),
        time: String(item.time ?? ""),
      }
    }),
  }
}

export function normalizeProductTransferReport(data: unknown): ProductTransferReportRow[] {
  if (!Array.isArray(data)) {
    return []
  }
  return data.map((row, index) => {
    const item = row as Record<string, unknown>
    return {
      id: Number(item.id ?? index + 1),
      serial: String(item.serial ?? ""),
      line_id: Number(item.line_id ?? 0),
      line_name: String(item.line_name ?? ""),
      model_id: Number(item.model_id ?? 0),
      model_name: String(item.model_name ?? ""),
      modeli: String(item.modeli ?? ""),
      odoo_code: String(item.odoo_code ?? ""),
      transferred_at: String(item.transferred_at ?? ""),
      user_name: String(item.user_name ?? ""),
    }
  })
}

export function toDateInputValue(date: Date) {
  return date.toISOString().slice(0, 10)
}

const COLORS = {
  titleBg: "FF0F172A",
  titleText: "FFFFFFFF",
  metaBg: "FFE2E8F0",
  headerBg: "FF0891B2",
  headerText: "FFFFFFFF",
  zebra: "FFF8FAFC",
  border: "FFCBD5E1",
  summaryBg: "FFEFF6FF",
}

function applyThinBorder(cell: { border?: object }) {
  cell.border = {
    top: { style: "thin", color: { argb: COLORS.border } },
    left: { style: "thin", color: { argb: COLORS.border } },
    bottom: { style: "thin", color: { argb: COLORS.border } },
    right: { style: "thin", color: { argb: COLORS.border } },
  }
}

function styleTitleRow(
  worksheet: import("exceljs").Worksheet,
  title: string,
  subtitle: string,
  columnCount: number,
) {
  worksheet.mergeCells(1, 1, 1, columnCount)
  const titleCell = worksheet.getCell(1, 1)
  titleCell.value = title
  titleCell.font = { bold: true, size: 16, color: { argb: COLORS.titleText } }
  titleCell.alignment = { vertical: "middle", horizontal: "center" }
  titleCell.fill = { type: "pattern", pattern: "solid", fgColor: { argb: COLORS.titleBg } }
  worksheet.getRow(1).height = 30

  worksheet.mergeCells(2, 1, 2, columnCount)
  const subtitleCell = worksheet.getCell(2, 1)
  subtitleCell.value = subtitle
  subtitleCell.font = { size: 11, color: { argb: "FF334155" } }
  subtitleCell.alignment = { vertical: "middle", horizontal: "center" }
  subtitleCell.fill = { type: "pattern", pattern: "solid", fgColor: { argb: COLORS.metaBg } }
  worksheet.getRow(2).height = 22
}

function styleHeaderRow(worksheet: import("exceljs").Worksheet, rowNumber: number, headers: string[]) {
  const row = worksheet.getRow(rowNumber)
  headers.forEach((header, index) => {
    const cell = row.getCell(index + 1)
    cell.value = header
    cell.font = { bold: true, color: { argb: COLORS.headerText } }
    cell.alignment = { vertical: "middle", horizontal: "center", wrapText: true }
    cell.fill = { type: "pattern", pattern: "solid", fgColor: { argb: COLORS.headerBg } }
    applyThinBorder(cell)
  })
  row.height = 24
}

function autosizeColumns(worksheet: import("exceljs").Worksheet, widths: number[]) {
  widths.forEach((width, index) => {
    worksheet.getColumn(index + 1).width = width
  })
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

export async function buildProductBalanceWorkbook(params: {
  lineName: string
  summary: ProductBalanceSummaryRow[]
  items: ProductBalanceItem[]
}) {
  const ExcelJS = await import("exceljs")
  const workbook = new ExcelJS.Workbook()
  workbook.creator = "Premier REF"
  workbook.created = new Date()

  const totalCount = params.summary.reduce((sum, row) => sum + row.count, 0)
  const summaryHeaders = ["Modeli", "Model", "ODOO code", "Soni"]
  const summarySheet = workbook.addWorksheet("Model bo'yicha")
  styleTitleRow(
    summarySheet,
    "Mahsulot balansi",
    `Liniya: ${params.lineName} | Jami: ${totalCount} ta`,
    summaryHeaders.length,
  )
  styleHeaderRow(summarySheet, 4, summaryHeaders)

  params.summary.forEach((item, index) => {
    const rowNumber = 5 + index
    const row = summarySheet.getRow(rowNumber)
    row.getCell(1).value = item.modeli
    row.getCell(2).value = item.model_name
    row.getCell(3).value = item.odoo_code
    row.getCell(4).value = item.count
    row.eachCell((cell) => {
      cell.alignment = { vertical: "middle", horizontal: "center" }
      applyThinBorder(cell)
      if (index % 2 === 1) {
        cell.fill = { type: "pattern", pattern: "solid", fgColor: { argb: COLORS.zebra } }
      }
    })
    row.getCell(4).numFmt = "#,##0"
  })

  const summaryTotalRow = summarySheet.getRow(5 + params.summary.length + 1)
  summarySheet.mergeCells(5 + params.summary.length + 1, 1, 5 + params.summary.length + 1, 3)
  summaryTotalRow.getCell(1).value = "Jami"
  summaryTotalRow.getCell(4).value = totalCount
  summaryTotalRow.eachCell((cell) => {
    cell.font = { bold: true }
    cell.fill = { type: "pattern", pattern: "solid", fgColor: { argb: COLORS.summaryBg } }
    applyThinBorder(cell)
  })

  summarySheet.views = [{ state: "frozen", ySplit: 4 }]
  autosizeColumns(summarySheet, [18, 36, 18, 12])

  const serialHeaders = ["Serial", "Model", "Modeli", "ODOO code", "Vaqt"]
  const serialSheet = workbook.addWorksheet("Seriallar")
  styleTitleRow(
    serialSheet,
    "Faol seriallar",
    `Liniya: ${params.lineName} | ${params.items.length} ta`,
    serialHeaders.length,
  )
  styleHeaderRow(serialSheet, 4, serialHeaders)

  params.items.forEach((item, index) => {
    const rowNumber = 5 + index
    const row = serialSheet.getRow(rowNumber)
    row.getCell(1).value = item.serial
    row.getCell(2).value = item.model_name
    row.getCell(3).value = item.modeli
    row.getCell(4).value = item.odoo_code
    row.getCell(5).value = item.time
    row.eachCell((cell) => {
      cell.alignment = { vertical: "middle", horizontal: index === 0 ? "left" : "center" }
      applyThinBorder(cell)
      if (index % 2 === 1) {
        cell.fill = { type: "pattern", pattern: "solid", fgColor: { argb: COLORS.zebra } }
      }
    })
  })

  serialSheet.views = [{ state: "frozen", ySplit: 4 }]
  autosizeColumns(serialSheet, [28, 36, 18, 18, 20])

  const excelBuffer = await workbook.xlsx.writeBuffer()
  return new Blob([excelBuffer], {
    type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  })
}

export function matchesProductFilter(values: unknown[], filterValue: string) {
  const query = filterValue.trim().toLocaleLowerCase()
  if (!query) {
    return true
  }
  return values
    .map((value) => String(value ?? "").toLocaleLowerCase())
    .join(" ")
    .includes(query)
}

export function useProductBalanceLines(initialLineId?: number | null) {
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
      const allLines = normalizeLineRows(result.data)
      const order = new Map(PRODUCT_LINE_IDS.map((id, index) => [id, index]))
      const productLines = allLines
        .filter((line) => order.has(line.line_id as (typeof PRODUCT_LINE_IDS)[number]))
        .sort((a, b) => (order.get(a.line_id as (typeof PRODUCT_LINE_IDS)[number]) ?? 0) - (order.get(b.line_id as (typeof PRODUCT_LINE_IDS)[number]) ?? 0))
      setLines(productLines)
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
      return pickDefaultProductLine(lines, initialLineId)
    })
  }, [initialLineId, lines, linesLoading])

  return { lines, selectedLine, setSelectedLine, linesLoading }
}
