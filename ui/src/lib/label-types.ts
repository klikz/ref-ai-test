import type { LabelDateFormat } from "@/lib/label-date-format"
import { DEFAULT_LABEL_DATE_FORMAT, TODAY_BINDING } from "@/lib/label-date-format"
import { GSCODE_DATA38_BINDING } from "@/lib/label-derived-bindings"

export type { LabelDateFormat } from "@/lib/label-date-format"
export { TODAY_BINDING } from "@/lib/label-date-format"

export type LabelElementType = "text" | "barcode" | "datamatrix" | "qrcode" | "image" | "line" | "rect" | "table"

export const LABEL_ELEMENT_DRAG_MIME = "application/x-label-element-type"

const LABEL_ELEMENT_TYPES: LabelElementType[] = [
  "text",
  "barcode",
  "datamatrix",
  "qrcode",
  "image",
  "line",
  "rect",
  "table",
]

export const LABEL_FONT_FAMILIES = [
  { id: "arial", label: "Arial", css: "Arial, Helvetica, sans-serif" },
  { id: "segoe", label: "Segoe UI", css: '"Segoe UI", sans-serif' },
  { id: "tahoma", label: "Tahoma", css: "Tahoma, sans-serif" },
  { id: "calibri", label: "Calibri", css: "Calibri, sans-serif" },
  { id: "consola", label: "Consolas", css: '"Label Consolas", Consolas, monospace' },
] as const

export type LabelFontFamily = (typeof LABEL_FONT_FAMILIES)[number]["id"]

export const DEFAULT_LABEL_FONT_FAMILY: LabelFontFamily = "arial"

export function labelFontFamilyCss(fontFamily?: string): string {
  const match = LABEL_FONT_FAMILIES.find((f) => f.id === fontFamily)
  return match?.css ?? LABEL_FONT_FAMILIES[0].css
}

export type LabelTableCell = {
  dataSource?: LabelDataSource
  staticText?: string
  binding?: string
  prefix?: string
  fontSize?: number
  fontFamily?: LabelFontFamily
  dateFormat?: LabelDateFormat
  fontWeight?: "normal" | "bold"
  align?: "left" | "center" | "right"
  fillColor?: string
}

export function isLabelElementType(value: string): value is LabelElementType {
  return LABEL_ELEMENT_TYPES.includes(value as LabelElementType)
}

export type BarcodeFormat = "ean13" | "code128"

/** static — matn o'zgarmaydi; backend — chop etishda serverdan keladi */
export type LabelDataSource = "static" | "backend"

export type LabelElement = {
  id: string
  type: LabelElementType
  x: number
  y: number
  width: number
  height: number
  dataSource?: LabelDataSource
  binding?: string
  staticText?: string
  prefix?: string
  fontSize?: number
  fontFamily?: LabelFontFamily
  dateFormat?: LabelDateFormat
  fontWeight?: "normal" | "bold"
  align?: "left" | "center" | "right"
  format?: BarcodeFormat
  src?: string
  strokeWidth?: number
  fillColor?: string
  rows?: number
  cols?: number
  cells?: LabelTableCell[]
  /** Har bir ustun kengligi (mm). Yig'indisi width ga teng. */
  colWidths?: number[]
  zIndex?: number
}

export type LabelDefinition = {
  version: number
  elements: LabelElement[]
}

export type LabelTemplate = {
  id: number
  name: string
  line_id: number
  line_name?: string
  width_mm: number
  height_mm: number
  dpi: number
  print_rotation_deg?: LabelPrintRotationDeg
  definition: LabelDefinition
}

export type LabelPrintRotationDeg = 0 | 90

export function offsetAllLabelElements(
  elements: LabelElement[],
  dx: number,
  dy: number,
  labelWidth: number,
  labelHeight: number,
): LabelElement[] {
  return elements.map((el) => ({
    ...el,
    x: Math.round(Math.max(0, Math.min(el.x + dx, Math.max(0, labelWidth - el.width))) * 10) / 10,
    y: Math.round(Math.max(0, Math.min(el.y + dy, Math.max(0, labelHeight - el.height))) * 10) / 10,
  }))
}

export const DEFAULT_LABEL_DEFINITION: LabelDefinition = {
  version: 1,
  elements: [],
}

export const MM_TO_PX = 4

export const LABEL_ZOOM_MIN = 0.25
export const LABEL_ZOOM_MAX = 3
export const LABEL_ZOOM_STEP = 0.25

export function mmToPx(mm: number, zoom = 1): number {
  return mm * MM_TO_PX * zoom
}

/** pt → px on canvas (matches Go print: pt → mm → dots at template DPI) */
export function ptToCanvasPx(pt: number, zoom = 1): number {
  return pt * (25.4 / 72) * MM_TO_PX * zoom
}

export function pxToMm(px: number, zoom = 1): number {
  return Math.round((px / (MM_TO_PX * zoom)) * 10) / 10
}

export function createElementId(): string {
  return `el-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
}

export function inferDataSource(element: LabelElement): LabelDataSource {
  if (element.dataSource) {
    return element.dataSource
  }
  if (element.type === "barcode" || element.type === "datamatrix" || element.type === "qrcode") {
    return "backend"
  }
  if (element.binding) {
    return "backend"
  }
  return "static"
}

export function inferTableCellDataSource(cell: LabelTableCell): LabelDataSource {
  if (cell.dataSource) {
    return cell.dataSource
  }
  if (cell.binding) {
    return "backend"
  }
  return "static"
}

export function normalizeTableCell(cell: LabelTableCell): LabelTableCell {
  const dataSource = inferTableCellDataSource(cell)
  if (dataSource === "static") {
    return { ...cell, dataSource, binding: undefined }
  }
  return { ...cell, dataSource, staticText: undefined }
}

export const TABLE_ROWS_MAX = 20
export const TABLE_COLS_MAX = 10

export function clampTableRows(rows: number): number {
  return Math.max(1, Math.min(TABLE_ROWS_MAX, Math.round(rows)))
}

export function clampTableCols(cols: number): number {
  return Math.max(1, Math.min(TABLE_COLS_MAX, Math.round(cols)))
}

export function createTableCells(
  rows: number,
  cols: number,
  fill?: (row: number, col: number) => Partial<LabelTableCell>,
): LabelTableCell[] {
  const cells: LabelTableCell[] = []
  for (let row = 0; row < rows; row++) {
    for (let col = 0; col < cols; col++) {
      cells.push(
        normalizeTableCell({
          dataSource: "static",
          staticText: "",
          align: "center",
          ...(fill?.(row, col) ?? {}),
        }),
      )
    }
  }
  return cells
}

export function resizeTableCells(
  cells: LabelTableCell[],
  oldRows: number,
  oldCols: number,
  newRows: number,
  newCols: number,
): LabelTableCell[] {
  const next = createTableCells(newRows, newCols)
  for (let row = 0; row < Math.min(oldRows, newRows); row++) {
    for (let col = 0; col < Math.min(oldCols, newCols); col++) {
      const prev = cells[row * oldCols + col]
      if (prev) {
        next[row * newCols + col] = prev
      }
    }
  }
  return next
}

export function insertTableRow(
  cells: LabelTableCell[],
  rows: number,
  cols: number,
  atRow: number,
): { cells: LabelTableCell[]; rows: number } | null {
  if (rows >= TABLE_ROWS_MAX) {
    return null
  }
  const insertAt = Math.max(0, Math.min(atRow, rows))
  const newRows = rows + 1
  const next: LabelTableCell[] = []
  for (let row = 0; row < newRows; row++) {
    for (let col = 0; col < cols; col++) {
      if (row < insertAt) {
        next.push(cells[row * cols + col])
      } else if (row === insertAt) {
        next.push(normalizeTableCell({ staticText: "", align: "center" }))
      } else {
        next.push(cells[(row - 1) * cols + col])
      }
    }
  }
  return { cells: next, rows: newRows }
}

export function insertTableCol(
  cells: LabelTableCell[],
  rows: number,
  cols: number,
  atCol: number,
): { cells: LabelTableCell[]; cols: number } | null {
  if (cols >= TABLE_COLS_MAX) {
    return null
  }
  const insertAt = Math.max(0, Math.min(atCol, cols))
  const newCols = cols + 1
  const next: LabelTableCell[] = []
  for (let row = 0; row < rows; row++) {
    const rowCells: LabelTableCell[] = []
    for (let col = 0; col < cols; col++) {
      rowCells.push(cells[row * cols + col])
    }
    rowCells.splice(insertAt, 0, normalizeTableCell({ staticText: "", align: "center" }))
    next.push(...rowCells)
  }
  return { cells: next, cols: newCols }
}

export function removeTableRow(
  cells: LabelTableCell[],
  rows: number,
  cols: number,
  atRow: number,
): { cells: LabelTableCell[]; rows: number } | null {
  if (rows <= 1) {
    return null
  }
  const removeAt = Math.max(0, Math.min(atRow, rows - 1))
  const newRows = rows - 1
  const next: LabelTableCell[] = []
  for (let row = 0; row < rows; row++) {
    if (row === removeAt) {
      continue
    }
    for (let col = 0; col < cols; col++) {
      next.push(cells[row * cols + col])
    }
  }
  return { cells: next, rows: newRows }
}

export function removeTableCol(
  cells: LabelTableCell[],
  rows: number,
  cols: number,
  atCol: number,
): { cells: LabelTableCell[]; cols: number } | null {
  if (cols <= 1) {
    return null
  }
  const removeAt = Math.max(0, Math.min(atCol, cols - 1))
  const newCols = cols - 1
  const next: LabelTableCell[] = []
  for (let row = 0; row < rows; row++) {
    for (let col = 0; col < cols; col++) {
      if (col === removeAt) {
        continue
      }
      next.push(cells[row * cols + col])
    }
  }
  return { cells: next, cols: newCols }
}

function roundMm(value: number): number {
  return Math.round(value * 10) / 10
}

function fixColWidthsSum(widths: number[], total: number): number[] {
  if (widths.length === 0) {
    return widths
  }
  const next = [...widths]
  const sum = next.reduce((acc, w) => acc + w, 0)
  next[next.length - 1] = roundMm(next[next.length - 1] + (total - sum))
  return next
}

export function equalTableColWidths(cols: number, totalWidth: number): number[] {
  if (cols < 1) {
    return []
  }
  const base = roundMm(totalWidth / cols)
  return fixColWidthsSum(Array.from({ length: cols }, () => base), totalWidth)
}

export function normalizeTableColWidths(cols: number, totalWidth: number, colWidths?: number[]): number[] {
  if (cols < 1 || totalWidth <= 0) {
    return []
  }
  if (!colWidths || colWidths.length !== cols) {
    return equalTableColWidths(cols, totalWidth)
  }
  const sum = colWidths.reduce((acc, w) => acc + w, 0)
  if (sum <= 0) {
    return equalTableColWidths(cols, totalWidth)
  }
  const scaled = colWidths.map((w) => roundMm((w / sum) * totalWidth))
  return fixColWidthsSum(scaled, totalWidth)
}

export function scaleTableColWidthsToWidth(
  cols: number,
  oldWidth: number,
  colWidths: number[] | undefined,
  newWidth: number,
): number[] {
  const current = normalizeTableColWidths(cols, oldWidth, colWidths)
  if (oldWidth <= 0) {
    return equalTableColWidths(cols, newWidth)
  }
  const ratio = newWidth / oldWidth
  const scaled = current.map((w) => roundMm(w * ratio))
  return fixColWidthsSum(scaled, newWidth)
}

export function resizeTableColWidths(
  colWidths: number[] | undefined,
  oldCols: number,
  newCols: number,
  totalWidth: number,
): { colWidths: number[]; width: number } {
  const current = normalizeTableColWidths(oldCols, totalWidth, colWidths)
  if (newCols === oldCols) {
    return { colWidths: current, width: totalWidth }
  }
  if (newCols < oldCols) {
    const kept = current.slice(0, newCols)
    const removed = current.slice(newCols)
    const removedSum = removed.reduce((acc, w) => acc + w, 0)
    if (kept.length > 0) {
      kept[kept.length - 1] = roundMm(kept[kept.length - 1] + removedSum)
    }
    const width = roundMm(kept.reduce((acc, w) => acc + w, 0))
    return { colWidths: kept, width }
  }
  const avg = current.reduce((acc, w) => acc + w, 0) / oldCols
  const next = [...current]
  for (let i = oldCols; i < newCols; i++) {
    next.push(roundMm(avg))
  }
  const width = roundMm(next.reduce((acc, w) => acc + w, 0))
  return { colWidths: fixColWidthsSum(next, width), width }
}

export function insertTableColWidths(
  colWidths: number[] | undefined,
  cols: number,
  atCol: number,
  oldTotalWidth: number,
  insertedWidth: number,
): { colWidths: number[]; width: number } {
  const current = normalizeTableColWidths(cols, oldTotalWidth, colWidths)
  const insertAt = Math.max(0, Math.min(atCol, cols))
  const next = [...current]
  next.splice(insertAt, 0, roundMm(insertedWidth))
  const width = roundMm(oldTotalWidth + insertedWidth)
  return { colWidths: fixColWidthsSum(next, width), width }
}

export function removeTableColWidths(
  colWidths: number[] | undefined,
  cols: number,
  atCol: number,
  oldTotalWidth: number,
): { colWidths: number[]; width: number } {
  const current = normalizeTableColWidths(cols, oldTotalWidth, colWidths)
  const removeAt = Math.max(0, Math.min(atCol, cols - 1))
  const removed = current[removeAt]
  const next = current.filter((_, index) => index !== removeAt)
  const width = Math.max(0.1, roundMm(oldTotalWidth - removed))
  return { colWidths: next, width }
}

export function getTableCell(element: LabelElement, row: number, col: number): LabelTableCell {
  const cols = element.cols ?? 1
  const cells = element.cells ?? []
  return cells[row * cols + col] ?? { dataSource: "static", staticText: "" }
}

export function normalizeElement(element: LabelElement): LabelElement {
  const withZIndex =
    element.zIndex == null ? element : { ...element, zIndex: Math.round(element.zIndex) }
  const dataSource = inferDataSource(withZIndex)
  if (withZIndex.type === "text") {
    if (dataSource === "static") {
      return { ...withZIndex, dataSource, binding: undefined }
    }
    return { ...withZIndex, dataSource, staticText: undefined }
  }
  if (withZIndex.type === "barcode" || withZIndex.type === "datamatrix" || withZIndex.type === "qrcode") {
    return { ...withZIndex, dataSource: "backend" }
  }
  if (withZIndex.type === "table") {
    const rows = clampTableRows(withZIndex.rows ?? 2)
    const cols = clampTableCols(withZIndex.cols ?? 3)
    let cells = withZIndex.cells
    if (!cells || cells.length !== rows * cols) {
      cells = resizeTableCells(cells ?? [], withZIndex.rows ?? rows, withZIndex.cols ?? cols, rows, cols)
    }
    return {
      ...withZIndex,
      rows,
      cols,
      cells: cells.map(normalizeTableCell),
      colWidths: normalizeTableColWidths(cols, withZIndex.width, withZIndex.colWidths),
    }
  }
  return withZIndex
}

export function createStaticTextElement(text = "Matn"): LabelElement {
  return normalizeElement({
    ...createDefaultElement("text"),
    dataSource: "static",
    staticText: text,
  })
}

export function createBackendTextElement(binding: string, prefix = ""): LabelElement {
  return normalizeElement({
    ...createDefaultElement("text"),
    dataSource: "backend",
    binding,
    prefix,
    staticText: undefined,
  })
}

export function createTodayDateElement(dateFormat: LabelDateFormat = DEFAULT_LABEL_DATE_FORMAT): LabelElement {
  return normalizeElement({
    ...createDefaultElement("text"),
    dataSource: "backend",
    binding: TODAY_BINDING,
    dateFormat,
    staticText: undefined,
  })
}

export function createGS1Data38Element(): LabelElement {
  return normalizeElement({
    ...createDefaultElement("text"),
    dataSource: "backend",
    binding: GSCODE_DATA38_BINDING,
    staticText: undefined,
  })
}

export function createBackendBarcodeElement(binding: string, format: BarcodeFormat = "ean13"): LabelElement {
  return normalizeElement({
    ...createDefaultElement("barcode"),
    binding,
    format,
  })
}

export function createBackendDataMatrixElement(binding = "gscode.data"): LabelElement {
  return normalizeElement({
    ...createDefaultElement("datamatrix"),
    binding,
  })
}

export function createTableElement(rows: number, cols: number, partial?: Partial<LabelElement>): LabelElement {
  const tableRows = clampTableRows(rows)
  const tableCols = clampTableCols(cols)
  const rowHeightMm = 6
  const width = Math.min(90, Math.max(20, tableCols * 30))
  return normalizeElement({
    id: createElementId(),
    type: "table",
    x: 5,
    y: 5,
    width,
    height: Math.max(rowHeightMm, tableRows * rowHeightMm),
    strokeWidth: 0.3,
    fontSize: 8,
    fontFamily: DEFAULT_LABEL_FONT_FAMILY,
    zIndex: 1,
    rows: tableRows,
    cols: tableCols,
    cells: createTableCells(tableRows, tableCols),
    colWidths: equalTableColWidths(tableCols, width),
    ...partial,
  })
}

export function createDefaultElement(type: LabelElementType): LabelElement {
  const base = { id: createElementId(), type, x: 5, y: 5, width: 30, height: 8, zIndex: 1 }

  switch (type) {
    case "text":
      return normalizeElement({
        ...base,
        dataSource: "static",
        staticText: "Matn",
        fontSize: 10,
        fontFamily: DEFAULT_LABEL_FONT_FAMILY,
        fontWeight: "normal",
        align: "left",
      })
    case "barcode":
      return normalizeElement({
        ...base,
        height: 12,
        dataSource: "backend",
        binding: "model.gs1_ean13",
        format: "ean13",
      })
    case "datamatrix":
      return normalizeElement({
        ...base,
        width: 18,
        height: 18,
        dataSource: "backend",
        binding: "gscode.data",
      })
    case "qrcode":
      return normalizeElement({
        ...base,
        width: 18,
        height: 18,
        dataSource: "backend",
        binding: "serial",
      })
    case "image":
      return { ...base, width: 20, height: 10, src: "" }
    case "line":
      return { ...base, height: 0.5, width: 40, strokeWidth: 0.3 }
    case "rect":
      return { ...base, width: 40, height: 25, strokeWidth: 0.3 }
    case "table":
      return createTableElement(2, 3, { ...base })
    default:
      return base
  }
}

export function parseDefinition(raw: unknown): LabelDefinition {
  if (!raw || typeof raw !== "object") {
    return DEFAULT_LABEL_DEFINITION
  }
  const obj = raw as LabelDefinition
  return {
    version: obj.version ?? 1,
    elements: Array.isArray(obj.elements) ? obj.elements.map(normalizeElement) : [],
  }
}
