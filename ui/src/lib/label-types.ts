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

/** Palette drag payload: oddiy tip yoki `vline` (vertikal chiziq). */
export function isLabelPaletteDropType(value: string): boolean {
  return isLabelElementType(value) || value === "vline"
}

export type LabelLineOrientation = "horizontal" | "vertical"

export function normalizeLineOrientation(value: unknown): LabelLineOrientation {
  return value === "vertical" ? "vertical" : "horizontal"
}

export type BarcodeFormat = "ean13" | "code128"

/** static — matn o'zgarmaydi; backend — chop etishda serverdan keladi */
export type LabelDataSource = "static" | "backend"

export type LabelElementRotationDeg = 0 | 90 | 180 | 270

export const LABEL_ELEMENT_ROTATION_DEGS: LabelElementRotationDeg[] = [0, 90, 180, 270]

export function normalizeElementRotationDeg(value: unknown): LabelElementRotationDeg {
  const n = typeof value === "number" ? value : Number(value)
  if (n === 90 || n === 180 || n === 270) {
    return n
  }
  return 0
}

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
  /** Faqat line: gorizontal (default) yoki vertikal. */
  orientation?: "horizontal" | "vertical"
  rows?: number
  cols?: number
  cells?: LabelTableCell[]
  /** Har bir ustun kengligi (mm). Yig'indisi width ga teng. */
  colWidths?: number[]
  zIndex?: number
  /** Element aylanishi (markaz atrofida). Print layoutga ta'sir qiladi. */
  rotationDeg?: LabelElementRotationDeg
  /** Bir xil groupId — guruhlangan elementlar; print layoutga ta'sir qilmaydi. */
  groupId?: string
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
  density?: number
  speed?: number
  gap_mm?: number
  use_printer_defaults?: boolean
  size_only?: boolean
  definition: LabelDefinition
}

export type LabelResizeHandle = "nw" | "n" | "ne" | "e" | "se" | "s" | "sw" | "w"

export const LABEL_CORNER_RESIZE_HANDLES: LabelResizeHandle[] = ["nw", "ne", "se", "sw"]

export function isLabelCornerResizeHandle(handle: LabelResizeHandle): boolean {
  return LABEL_CORNER_RESIZE_HANDLES.includes(handle)
}

function roundMm10(value: number): number {
  return Math.round(value * 10) / 10
}

const LABEL_ELEMENT_MIN_SIZE_MM = 1

/**
 * Resize box from a handle. Corner handles keep aspect ratio; edge handles resize freely.
 * Opposite corner/edge stays anchored.
 */
export function resizeLabelElementBox(
  origin: { x: number; y: number; width: number; height: number },
  handle: LabelResizeHandle,
  dx: number,
  dy: number,
  labelWidth: number,
  labelHeight: number,
  proportional = isLabelCornerResizeHandle(handle),
): { x: number; y: number; width: number; height: number } {
  const min = LABEL_ELEMENT_MIN_SIZE_MM
  const ow = Math.max(min, origin.width)
  const oh = Math.max(min, origin.height)
  const aspect = ow / oh

  let x = origin.x
  let y = origin.y
  let width = ow
  let height = oh

  const applyFree = () => {
    switch (handle) {
      case "e":
        width = ow + dx
        break
      case "w":
        width = ow - dx
        x = origin.x + dx
        break
      case "s":
        height = oh + dy
        break
      case "n":
        height = oh - dy
        y = origin.y + dy
        break
      case "se":
        width = ow + dx
        height = oh + dy
        break
      case "sw":
        width = ow - dx
        height = oh + dy
        x = origin.x + dx
        break
      case "ne":
        width = ow + dx
        height = oh - dy
        y = origin.y + dy
        break
      case "nw":
        width = ow - dx
        height = oh - dy
        x = origin.x + dx
        y = origin.y + dy
        break
    }
  }

  const applyProportional = () => {
    // Pick the dominant delta relative to aspect so drag feels natural.
    let newW: number
    let newH: number
    switch (handle) {
      case "se": {
        const candW = ow + dx
        const candH = oh + dy
        if (Math.abs(dx) * oh >= Math.abs(dy) * ow) {
          newW = candW
          newH = newW / aspect
        } else {
          newH = candH
          newW = newH * aspect
        }
        width = newW
        height = newH
        break
      }
      case "nw": {
        const candW = ow - dx
        const candH = oh - dy
        if (Math.abs(dx) * oh >= Math.abs(dy) * ow) {
          newW = candW
          newH = newW / aspect
        } else {
          newH = candH
          newW = newH * aspect
        }
        width = newW
        height = newH
        x = origin.x + ow - width
        y = origin.y + oh - height
        break
      }
      case "ne": {
        const candW = ow + dx
        const candH = oh - dy
        if (Math.abs(dx) * oh >= Math.abs(dy) * ow) {
          newW = candW
          newH = newW / aspect
        } else {
          newH = candH
          newW = newH * aspect
        }
        width = newW
        height = newH
        y = origin.y + oh - height
        break
      }
      case "sw": {
        const candW = ow - dx
        const candH = oh + dy
        if (Math.abs(dx) * oh >= Math.abs(dy) * ow) {
          newW = candW
          newH = newW / aspect
        } else {
          newH = candH
          newW = newH * aspect
        }
        width = newW
        height = newH
        x = origin.x + ow - width
        break
      }
      default:
        applyFree()
        return
    }
  }

  if (proportional && isLabelCornerResizeHandle(handle)) {
    applyProportional()
  } else {
    applyFree()
  }

  // Enforce minimum size while keeping the anchored edge fixed.
  if (width < min) {
    if (handle === "w" || handle === "nw" || handle === "sw") {
      x = origin.x + ow - min
    }
    width = min
    if (proportional && isLabelCornerResizeHandle(handle)) {
      height = Math.max(min, width / aspect)
      if (handle === "nw" || handle === "ne") {
        y = origin.y + oh - height
      }
    }
  }
  if (height < min) {
    if (handle === "n" || handle === "nw" || handle === "ne") {
      y = origin.y + oh - min
    }
    height = min
    if (proportional && isLabelCornerResizeHandle(handle)) {
      width = Math.max(min, height * aspect)
      if (handle === "nw" || handle === "sw") {
        x = origin.x + ow - width
      }
    }
  }

  // Clamp inside label; shrink from the moving edges if needed.
  if (x < 0) {
    width = Math.max(min, width + x)
    x = 0
  }
  if (y < 0) {
    height = Math.max(min, height + y)
    y = 0
  }
  if (x + width > labelWidth) {
    width = Math.max(min, labelWidth - x)
  }
  if (y + height > labelHeight) {
    height = Math.max(min, labelHeight - y)
  }

  if (proportional && isLabelCornerResizeHandle(handle) && height > 0) {
    const clampedAspect = width / height
    if (Math.abs(clampedAspect - aspect) > 0.001) {
      // Re-fit to aspect within remaining room from the anchored corner.
      const maxW = handle === "nw" || handle === "sw" ? origin.x + ow - x : labelWidth - x
      const maxH = handle === "nw" || handle === "ne" ? origin.y + oh - y : labelHeight - y
      width = Math.min(width, maxW)
      height = Math.min(height, maxH)
      if (width / aspect <= height) {
        height = width / aspect
      } else {
        width = height * aspect
      }
      if (handle === "nw" || handle === "sw") {
        x = origin.x + ow - width
      }
      if (handle === "nw" || handle === "ne") {
        y = origin.y + oh - height
      }
      x = Math.max(0, x)
      y = Math.max(0, y)
    }
  }

  return {
    x: roundMm10(Math.max(0, x)),
    y: roundMm10(Math.max(0, y)),
    width: roundMm10(Math.max(min, width)),
    height: roundMm10(Math.max(min, height)),
  }
}

/** CSS rotate(θ) (clockwise, Y-down) inverse: screen delta → local (unrotated) delta. */
export function screenDeltaToLocalElementDelta(
  dx: number,
  dy: number,
  rotationDeg: number,
): { dx: number; dy: number } {
  const rot = normalizeElementRotationDeg(rotationDeg)
  if (rot === 0) {
    return { dx, dy }
  }
  const rad = (rot * Math.PI) / 180
  const c = Math.cos(rad)
  const s = Math.sin(rad)
  return {
    dx: dx * c - dy * s,
    dy: dx * s + dy * c,
  }
}

/**
 * Keep geometric center fixed when width/height change (needed for CSS rotate about center).
 */
export function placeBoxKeepingCenter(
  origin: { x: number; y: number; width: number; height: number },
  nextWidth: number,
  nextHeight: number,
  labelWidth: number,
  labelHeight: number,
): { x: number; y: number; width: number; height: number } {
  const min = LABEL_ELEMENT_MIN_SIZE_MM
  let width = Math.max(min, nextWidth)
  let height = Math.max(min, nextHeight)
  width = Math.min(width, Math.max(min, labelWidth))
  height = Math.min(height, Math.max(min, labelHeight))

  const cx = origin.x + origin.width / 2
  const cy = origin.y + origin.height / 2
  let x = cx - width / 2
  let y = cy - height / 2
  x = Math.max(0, Math.min(x, labelWidth - width))
  y = Math.max(0, Math.min(y, labelHeight - height))

  return {
    x: roundMm10(x),
    y: roundMm10(y),
    width: roundMm10(width),
    height: roundMm10(height),
  }
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

export const LABEL_ELEMENT_SCALE_STEP = 1.1

function scaleLabelFontSize(fontSize: number | undefined, factor: number): number | undefined {
  if (fontSize == null) {
    return undefined
  }
  return Math.round(Math.max(4, fontSize * factor) * 2) / 2
}

/**
 * Scale all elements uniformly about their combined bounding-box center.
 * Also scales fontSize, strokeWidth, and table colWidths / cell fonts.
 */
export function scaleAllLabelElements(
  elements: LabelElement[],
  factor: number,
  labelWidth: number,
  labelHeight: number,
): LabelElement[] {
  if (elements.length === 0 || !Number.isFinite(factor) || factor <= 0 || factor === 1) {
    return elements
  }

  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity
  for (const el of elements) {
    minX = Math.min(minX, el.x)
    minY = Math.min(minY, el.y)
    maxX = Math.max(maxX, el.x + el.width)
    maxY = Math.max(maxY, el.y + el.height)
  }
  const cx = (minX + maxX) / 2
  const cy = (minY + maxY) / 2
  const minSize = LABEL_ELEMENT_MIN_SIZE_MM

  return elements.map((el) => {
    let width = Math.max(minSize, el.width * factor)
    let height = Math.max(minSize, el.height * factor)
    let x = cx + (el.x - cx) * factor
    let y = cy + (el.y - cy) * factor

    width = Math.min(width, Math.max(minSize, labelWidth))
    height = Math.min(height, Math.max(minSize, labelHeight))
    x = Math.max(0, Math.min(x, Math.max(0, labelWidth - width)))
    y = Math.max(0, Math.min(y, Math.max(0, labelHeight - height)))

    const next: LabelElement = {
      ...el,
      x: roundMm10(x),
      y: roundMm10(y),
      width: roundMm10(width),
      height: roundMm10(height),
      fontSize: scaleLabelFontSize(el.fontSize, factor),
      strokeWidth:
        el.strokeWidth != null
          ? Math.round(Math.max(0.1, el.strokeWidth * factor) * 100) / 100
          : el.strokeWidth,
    }

    if (el.type === "table") {
      const cols = el.cols ?? 1
      const scaledCols =
        el.colWidths?.map((w) => roundMm10(Math.max(0.1, w * factor))) ??
        equalTableColWidths(cols, next.width)
      next.colWidths = normalizeTableColWidths(cols, next.width, scaledCols)
      if (el.cells) {
        next.cells = el.cells.map((cell) => ({
          ...cell,
          fontSize: scaleLabelFontSize(cell.fontSize, factor),
        }))
      }
    }

    return next
  })
}

/** Tanlangan id lar + ularning groupId a'zolari. */
export function expandSelectionWithGroups(elements: LabelElement[], ids: string[]): string[] {
  const idSet = new Set(ids)
  const groupIds = new Set<string>()
  for (const el of elements) {
    if (idSet.has(el.id) && el.groupId) {
      groupIds.add(el.groupId)
    }
  }
  if (groupIds.size === 0) {
    return [...idSet]
  }
  for (const el of elements) {
    if (el.groupId && groupIds.has(el.groupId)) {
      idSet.add(el.id)
    }
  }
  return [...idSet]
}

/** Guruh a'zolari (groupId bo'yicha). */
export function elementIdsInGroup(elements: LabelElement[], groupId: string): string[] {
  return elements.filter((el) => el.groupId === groupId).map((el) => el.id)
}

/**
 * Tanlangan elementlarni birga siljitadi; nisbiy joylashuv saqlanadi
 * (to'plam bounding box label ichida qoladi).
 */
export function offsetLabelElementsByIds(
  elements: LabelElement[],
  ids: string[],
  dx: number,
  dy: number,
  labelWidth: number,
  labelHeight: number,
): LabelElement[] {
  if (dx === 0 && dy === 0) {
    return elements
  }
  const idSet = new Set(ids)
  const selected = elements.filter((el) => idSet.has(el.id))
  if (selected.length === 0) {
    return elements
  }

  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity
  for (const el of selected) {
    minX = Math.min(minX, el.x)
    minY = Math.min(minY, el.y)
    maxX = Math.max(maxX, el.x + el.width)
    maxY = Math.max(maxY, el.y + el.height)
  }

  const clampedDx = Math.max(-minX, Math.min(dx, Math.max(0, labelWidth - maxX)))
  const clampedDy = Math.max(-minY, Math.min(dy, Math.max(0, labelHeight - maxY)))
  if (clampedDx === 0 && clampedDy === 0) {
    return elements
  }

  return elements.map((el) => {
    if (!idSet.has(el.id)) {
      return el
    }
    return {
      ...el,
      x: Math.round((el.x + clampedDx) * 10) / 10,
      y: Math.round((el.y + clampedDy) * 10) / 10,
    }
  })
}

/** Origins dan hisoblangan absolute offset (drag paytida). */
export function applyLabelElementDragFromOrigins(
  elements: LabelElement[],
  origins: Record<string, { x: number; y: number }>,
  dx: number,
  dy: number,
  labelWidth: number,
  labelHeight: number,
): LabelElement[] {
  const ids = Object.keys(origins)
  if (ids.length === 0) {
    return elements
  }

  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity
  for (const el of elements) {
    const orig = origins[el.id]
    if (!orig) {
      continue
    }
    minX = Math.min(minX, orig.x)
    minY = Math.min(minY, orig.y)
    maxX = Math.max(maxX, orig.x + el.width)
    maxY = Math.max(maxY, orig.y + el.height)
  }

  const clampedDx = Math.max(-minX, Math.min(dx, Math.max(0, labelWidth - maxX)))
  const clampedDy = Math.max(-minY, Math.min(dy, Math.max(0, labelHeight - maxY)))

  return elements.map((el) => {
    const orig = origins[el.id]
    if (!orig) {
      return el
    }
    return {
      ...el,
      x: Math.round((orig.x + clampedDx) * 10) / 10,
      y: Math.round((orig.y + clampedDy) * 10) / 10,
    }
  })
}

export function groupLabelElements(elements: LabelElement[], ids: string[]): LabelElement[] {
  if (ids.length < 2) {
    return elements
  }
  const idSet = new Set(ids)
  const groupId = createElementId().replace(/^el-/, "grp-")
  return elements.map((el) => (idSet.has(el.id) ? { ...el, groupId } : el))
}

export function ungroupLabelElements(elements: LabelElement[], ids: string[]): LabelElement[] {
  const idSet = new Set(ids)
  return elements.map((el) => {
    if (!idSet.has(el.id) || !el.groupId) {
      return el
    }
    const { groupId: _removed, ...rest } = el
    return rest
  })
}

export function selectionHasGroup(elements: LabelElement[], ids: string[]): boolean {
  const idSet = new Set(ids)
  return elements.some((el) => idSet.has(el.id) && Boolean(el.groupId))
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
  const withRotation: LabelElement = {
    ...withZIndex,
    rotationDeg: normalizeElementRotationDeg(withZIndex.rotationDeg),
  }
  const dataSource = inferDataSource(withRotation)
  if (withRotation.type === "line") {
    return {
      ...withRotation,
      orientation: normalizeLineOrientation(withRotation.orientation),
    }
  }
  if (withRotation.type === "text") {
    if (dataSource === "static") {
      return { ...withRotation, dataSource, binding: undefined }
    }
    return { ...withRotation, dataSource, staticText: undefined }
  }
  if (withRotation.type === "barcode" || withRotation.type === "datamatrix" || withRotation.type === "qrcode") {
    return { ...withRotation, dataSource: "backend" }
  }
  if (withRotation.type === "table") {
    const rows = clampTableRows(withRotation.rows ?? 2)
    const cols = clampTableCols(withRotation.cols ?? 3)
    let cells = withRotation.cells
    if (!cells || cells.length !== rows * cols) {
      cells = resizeTableCells(cells ?? [], withRotation.rows ?? rows, withRotation.cols ?? cols, rows, cols)
    }
    return {
      ...withRotation,
      rows,
      cols,
      cells: cells.map(normalizeTableCell),
      colWidths: normalizeTableColWidths(cols, withRotation.width, withRotation.colWidths),
    }
  }
  return withRotation
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
      return normalizeElement({
        ...base,
        height: 0.5,
        width: 40,
        strokeWidth: 0.3,
        orientation: "horizontal",
      })
    case "rect":
      return { ...base, width: 40, height: 25, strokeWidth: 0.3 }
    case "table":
      return createTableElement(2, 3, { ...base })
    default:
      return base
  }
}

export function createVerticalLineElement(): LabelElement {
  return normalizeElement({
    id: createElementId(),
    type: "line",
    x: 5,
    y: 5,
    width: 0.5,
    height: 40,
    zIndex: 1,
    strokeWidth: 0.3,
    orientation: "vertical",
  })
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
