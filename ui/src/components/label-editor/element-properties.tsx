import { Trash2, Upload } from "lucide-react"
import { useState } from "react"
import { useDropzone } from "react-dropzone"

import { BindingSelect } from "@/components/label-editor/label-data-panel"
import {
  PropertiesGroupToggle,
  PropertiesSection,
  readPropertiesGroupedPreference,
  writePropertiesGroupedPreference,
} from "@/components/label-editor/properties-section"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { BARCODE_BINDINGS, DATAMATRIX_BINDINGS, TEXT_BINDINGS } from "@/lib/label-binding-groups"
import { getBindingLabel } from "@/lib/label-bindings"
import {
  DEFAULT_LABEL_DATE_FORMAT,
  isTodayBinding,
  LABEL_DATE_FORMATS,
  type LabelDateFormat,
} from "@/lib/label-date-format"
import type { LabelDataSource, LabelElement, LabelFontFamily, LabelTableCell } from "@/lib/label-types"
import {
  TABLE_COLS_MAX,
  TABLE_ROWS_MAX,
  LABEL_ELEMENT_ROTATION_DEGS,
  LABEL_FONT_FAMILIES,
  clampTableCols,
  clampTableRows,
  createTableCells,
  equalTableColWidths,
  getTableCell,
  inferDataSource,
  inferTableCellDataSource,
  insertTableCol,
  insertTableColWidths,
  insertTableRow,
  normalizeElement,
  normalizeElementRotationDeg,
  normalizeTableCell,
  normalizeTableColWidths,
  placeBoxKeepingCenter,
  removeTableCol,
  removeTableColWidths,
  removeTableRow,
  resizeTableCells,
  resizeTableColWidths,
  scaleTableColWidthsToWidth,
} from "@/lib/label-types"
import { cn } from "@/lib/utils"
import { labelAssetUrl } from "@/lib/label-image"

type ElementPropertiesProps = {
  element: LabelElement | null
  selectedCount?: number
  canGroup?: boolean
  canUngroup?: boolean
  onGroup?: () => void
  onUngroup?: () => void
  onChange: (element: LabelElement) => void
  onDelete: () => void
  onImageFile?: (file: File) => void
  labelWidthMm?: number
  labelHeightMm?: number
}

function NumberField({
  label,
  value,
  onChange,
  integer = false,
  min,
}: {
  label: string
  value: number
  onChange: (v: number) => void
  integer?: boolean
  min?: number
}) {
  return (
    <div className="space-y-1">
      <Label className="text-xs">{label}</Label>
      <Input
        type="number"
        step={integer ? 1 : 0.5}
        min={min}
        value={value}
        onChange={(e) => {
          const raw = Number(e.target.value)
          if (!Number.isFinite(raw)) {
            return
          }
          onChange(integer ? Math.round(raw) : raw)
        }}
        className="h-8"
      />
    </div>
  )
}

function ZIndexField({
  value,
  onChange,
}: {
  value: number
  onChange: (zIndex: number) => void
}) {
  return (
    <NumberField
      label="Qavat (z-index)"
      value={value}
      onChange={onChange}
      integer
      min={0}
    />
  )
}

function DataSourceToggle({
  value,
  onChange,
}: {
  value: LabelDataSource
  onChange: (v: LabelDataSource) => void
}) {
  return (
    <div className="space-y-1">
      <Label className="text-xs">Ma&apos;lumot turi</Label>
      <div className="grid grid-cols-2 gap-1 rounded-lg border p-1">
        <button
          type="button"
          className={cn(
            "rounded-md px-2 py-1.5 text-xs font-medium transition-colors",
            value === "static" ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:bg-muted",
          )}
          onClick={() => onChange("static")}
        >
          Statik
        </button>
        <button
          type="button"
          className={cn(
            "rounded-md px-2 py-1.5 text-xs font-medium transition-colors",
            value === "backend" ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:bg-muted",
          )}
          onClick={() => onChange("backend")}
        >
          Backend
        </button>
      </div>
    </div>
  )
}

function DateFormatField({
  value,
  onChange,
}: {
  value: LabelDateFormat
  onChange: (dateFormat: LabelDateFormat) => void
}) {
  return (
    <div className="space-y-1">
      <Label className="text-xs">Sana formati</Label>
      <select
        className="h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
        value={value}
        onChange={(e) => onChange(e.target.value as LabelDateFormat)}
      >
        {LABEL_DATE_FORMATS.map((format) => (
          <option key={format.id} value={format.id}>
            {format.label}
          </option>
        ))}
      </select>
    </div>
  )
}

function FontFamilyField({
  label = "Shrift turi",
  value,
  onChange,
}: {
  label?: string
  value: LabelFontFamily
  onChange: (fontFamily: LabelFontFamily) => void
}) {
  return (
    <div className="space-y-1">
      <Label className="text-xs">{label}</Label>
      <select
        className="h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
        value={value}
        onChange={(e) => onChange(e.target.value as LabelFontFamily)}
      >
        {LABEL_FONT_FAMILIES.map((font) => (
          <option key={font.id} value={font.id}>
            {font.label}
          </option>
        ))}
      </select>
    </div>
  )
}

export function ElementProperties({
  element,
  selectedCount = 0,
  canGroup = false,
  canUngroup = false,
  onGroup,
  onUngroup,
  onChange,
  onDelete,
  onImageFile,
  labelWidthMm = 1000,
  labelHeightMm = 1000,
}: ElementPropertiesProps) {
  if (selectedCount > 1) {
    return (
      <div className="space-y-3">
        <p className="text-sm font-medium">{selectedCount} ta element tanlangan</p>
        <p className="text-xs text-muted-foreground">
          Guruhlash — birga surish uchun. Ctrl+G / Ctrl+Shift+G.
        </p>
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="outline" size="sm" disabled={!canGroup} onClick={onGroup}>
            Guruhlash
          </Button>
          <Button type="button" variant="outline" size="sm" disabled={!canUngroup} onClick={onUngroup}>
            Ajratish
          </Button>
          <Button type="button" variant="destructive" size="sm" className="gap-1" onClick={onDelete}>
            <Trash2 className="size-3.5" />
            O&apos;chirish
          </Button>
        </div>
      </div>
    )
  }

  if (!element) {
    return (
      <div className="text-sm text-muted-foreground">
        Element tanlang yoki chapdan statik matn / backend maydon qo&apos;shing
      </div>
    )
  }

  function patch(partial: Partial<LabelElement>) {
    const sizeChanging =
      (partial.width !== undefined && partial.width !== element.width) ||
      (partial.height !== undefined && partial.height !== element.height)
    const rotationDeg = normalizeElementRotationDeg(element.rotationDeg)

    let next: LabelElement = { ...element, ...partial }
    if (element.type === "table" && partial.width !== undefined && partial.width !== element.width) {
      next = {
        ...next,
        colWidths: scaleTableColWidthsToWidth(
          element.cols ?? 3,
          element.width,
          element.colWidths,
          partial.width,
        ),
      }
    }

    if (sizeChanging && rotationDeg !== 0) {
      const placed = placeBoxKeepingCenter(
        element,
        next.width,
        next.height,
        labelWidthMm,
        labelHeightMm,
      )
      next = { ...next, ...placed }
    }

    onChange(normalizeElement(next))
  }

  function setDataSource(dataSource: LabelDataSource) {
    if (element.type !== "text") {
      return
    }
    if (dataSource === "static") {
      patch({ dataSource: "static", binding: undefined, staticText: element.staticText ?? "" })
      return
    }
    patch({
      dataSource: "backend",
      staticText: undefined,
      binding: element.binding ?? TEXT_BINDINGS[0]?.key ?? "serial",
    })
  }

  const dataSource = inferDataSource(element)
  const [grouped, setGrouped] = useState(readPropertiesGroupedPreference)

  function toggleGrouped(next: boolean) {
    setGrouped(next)
    writePropertiesGroupedPreference(next)
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-2">
        <div className="min-w-0">
          <div className="text-sm font-medium capitalize">
            {element.type === "table" ? "Jadval" : element.type === "qrcode" ? "QR kod" : element.type}
          </div>
          {(element.type === "text" || element.type === "barcode" || element.type === "datamatrix" || element.type === "qrcode") && (
            <div className="text-[10px] text-muted-foreground">
              {dataSource === "static" ? "Statik ma'lumot" : "Backend maydon"}
            </div>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-1">
          {canUngroup ? (
            <Button type="button" variant="outline" size="sm" className="h-8" onClick={onUngroup}>
              Ajratish
            </Button>
          ) : null}
          <PropertiesGroupToggle grouped={grouped} onChange={toggleGrouped} />
          <Button variant="outline" size="icon" className="size-8" onClick={onDelete}>
            <Trash2 className="size-4 text-destructive" />
          </Button>
        </div>
      </div>

      <PropertiesSection title="Joylashuv" grouped={grouped} defaultOpen>
        <div className="grid grid-cols-2 gap-2">
          <NumberField label="X (mm)" value={element.x} onChange={(x) => patch({ x })} />
          <NumberField label="Y (mm)" value={element.y} onChange={(y) => patch({ y })} />
          <NumberField label="W (mm)" value={element.width} onChange={(width) => patch({ width })} />
          <NumberField label="H (mm)" value={element.height} onChange={(height) => patch({ height })} />
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">Aylanish</Label>
          <select
            className="flex h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
            value={element.rotationDeg ?? 0}
            onChange={(e) => patch({ rotationDeg: normalizeElementRotationDeg(Number(e.target.value)) })}
          >
            {LABEL_ELEMENT_ROTATION_DEGS.map((deg) => (
              <option key={deg} value={deg}>
                {deg}°
              </option>
            ))}
          </select>
        </div>
      </PropertiesSection>

      {element.type === "text" && (
        <>
          <PropertiesSection title="Ma'lumot" grouped={grouped} defaultOpen>
            <DataSourceToggle value={dataSource} onChange={setDataSource} />

            {dataSource === "static" ? (
              <div className="space-y-1">
                <Label className="text-xs">Statik matn</Label>
                <Input
                  value={element.staticText ?? ""}
                  onChange={(e) => patch({ staticText: e.target.value })}
                  className="h-8"
                  placeholder="Masalan: Ishlab chiqaruvchi:"
                />
                <p className="text-[10px] text-muted-foreground">Chop etishda o&apos;zgarmaydi</p>
              </div>
            ) : (
              <>
                <div className="space-y-1">
                  <Label className="text-xs">Backend maydon</Label>
                  <BindingSelect
                    value={element.binding ?? ""}
                    bindings={TEXT_BINDINGS}
                    onChange={(binding) => patch({ binding })}
                    allowEmpty={false}
                  />
                  {element.binding ? (
                    <code className="block text-[10px] text-muted-foreground">kalit: {element.binding}</code>
                  ) : null}
                </div>
                {isTodayBinding(element.binding) ? (
                  <>
                    <DateFormatField
                      value={element.dateFormat ?? DEFAULT_LABEL_DATE_FORMAT}
                      onChange={(dateFormat) => patch({ dateFormat })}
                    />
                    <p className="text-[10px] text-muted-foreground">
                      Har safar chop etishda joriy sana qo&apos;yiladi
                    </p>
                  </>
                ) : null}
                {element.binding === "gscode.data38" ? (
                  <p className="text-[10px] text-muted-foreground">
                    GS1 Data ning birinchi 38 belgisi chop etiladi
                  </p>
                ) : null}
                <div className="space-y-1">
                  <Label className="text-xs">Oldingi qism (ixtiyoriy)</Label>
                  <Input
                    value={element.prefix ?? ""}
                    onChange={(e) => patch({ prefix: e.target.value })}
                    className="h-8"
                    placeholder="S/N: "
                  />
                  <p className="text-[10px] text-muted-foreground">
                    Ko&apos;rinishi: {element.prefix ?? ""}
                    {element.binding ? getBindingLabel(element.binding) : "..."}
                  </p>
                </div>
              </>
            )}
          </PropertiesSection>

          <PropertiesSection title="Shrift va ko'rinish" grouped={grouped} defaultOpen={false}>
            <FontFamilyField
              value={element.fontFamily ?? "arial"}
              onChange={(fontFamily) => patch({ fontFamily })}
            />
            <NumberField
              label="Shrift (pt)"
              value={element.fontSize ?? 10}
              onChange={(fontSize) => patch({ fontSize })}
            />
            <div className="space-y-1">
              <Label className="text-xs">Qalinlik</Label>
              <select
                className="h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
                value={element.fontWeight ?? "normal"}
                onChange={(e) => patch({ fontWeight: e.target.value as "normal" | "bold" })}
              >
                <option value="normal">Oddiy</option>
                <option value="bold">Qalin</option>
              </select>
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Tekislash</Label>
              <select
                className="h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
                value={element.align ?? "left"}
                onChange={(e) => patch({ align: e.target.value as "left" | "center" | "right" })}
              >
                <option value="left">Chap</option>
                <option value="center">Markaz</option>
                <option value="right">O&apos;ng</option>
              </select>
            </div>
            <ZIndexField value={element.zIndex ?? 1} onChange={(zIndex) => patch({ zIndex })} />
          </PropertiesSection>
        </>
      )}

      {element.type === "barcode" && (
        <PropertiesSection title="Shtrix kod" grouped={grouped} defaultOpen>
          <div className="rounded-md border border-dashed bg-muted/30 px-2 py-1.5 text-[10px] text-muted-foreground">
            Backend dan keladigan ma&apos;lumot (shtrix kod)
          </div>
          <div className="space-y-1">
            <Label className="text-xs">Backend maydon</Label>
            <BindingSelect
              value={element.binding ?? ""}
              bindings={BARCODE_BINDINGS}
              onChange={(binding) => patch({ binding })}
              allowEmpty={false}
            />
            {element.binding ? (
              <code className="block text-[10px] text-muted-foreground">kalit: {element.binding}</code>
            ) : null}
          </div>
          <div className="space-y-1">
            <Label className="text-xs">Format</Label>
            <select
              className="h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
              value={element.format ?? "ean13"}
              onChange={(e) => patch({ format: e.target.value as "ean13" | "code128" })}
            >
              <option value="ean13">EAN13</option>
              <option value="code128">Code128</option>
            </select>
          </div>
          <ZIndexField value={element.zIndex ?? 1} onChange={(zIndex) => patch({ zIndex })} />
        </PropertiesSection>
      )}

      {element.type === "datamatrix" && (
        <PropertiesSection title="DataMatrix" grouped={grouped} defaultOpen>
          <div className="rounded-md border border-dashed bg-muted/30 px-2 py-1.5 text-[10px] text-muted-foreground">
            Backend dan keladigan ma&apos;lumot (GS1 DataMatrix)
          </div>
          <div className="space-y-1">
            <Label className="text-xs">Backend maydon</Label>
            <BindingSelect
              value={element.binding ?? ""}
              bindings={DATAMATRIX_BINDINGS}
              onChange={(binding) => patch({ binding })}
              allowEmpty={false}
            />
            {element.binding ? (
              <code className="block text-[10px] text-muted-foreground">kalit: {element.binding}</code>
            ) : null}
          </div>
          <ZIndexField value={element.zIndex ?? 1} onChange={(zIndex) => patch({ zIndex })} />
        </PropertiesSection>
      )}

      {element.type === "qrcode" && (
        <PropertiesSection title="QR kod" grouped={grouped} defaultOpen>
          <div className="rounded-md border border-dashed bg-muted/30 px-2 py-1.5 text-[10px] text-muted-foreground">
            Backend dan keladigan ma&apos;lumot (QR kod)
          </div>
          <div className="space-y-1">
            <Label className="text-xs">Backend maydon</Label>
            <BindingSelect
              value={element.binding ?? ""}
              bindings={BARCODE_BINDINGS}
              onChange={(binding) => patch({ binding })}
              allowEmpty={false}
            />
            {element.binding ? (
              <code className="block text-[10px] text-muted-foreground">kalit: {element.binding}</code>
            ) : null}
          </div>
          <ZIndexField value={element.zIndex ?? 1} onChange={(zIndex) => patch({ zIndex })} />
        </PropertiesSection>
      )}

      {element.type === "image" && (
        <PropertiesSection title="Rasm" grouped={grouped} defaultOpen>
          <ImageProperties
            element={element}
            onImageFile={onImageFile}
            onChange={patch}
            onZIndex={(zIndex) => patch({ zIndex })}
            zIndex={element.zIndex ?? 1}
          />
        </PropertiesSection>
      )}

      {element.type === "table" && (
        <TableProperties element={element} onChange={onChange} grouped={grouped} />
      )}

      {(element.type === "line" || element.type === "rect") && (
        <PropertiesSection title="Ko'rinish" grouped={grouped} defaultOpen>
          {element.type === "line" ? (
            <div className="space-y-1">
              <Label className="text-xs">Yo&apos;nalish</Label>
              <select
                className="flex h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
                value={element.orientation === "vertical" ? "vertical" : "horizontal"}
                onChange={(e) => {
                  const orientation = e.target.value === "vertical" ? "vertical" : "horizontal"
                  const swapAxes =
                    (orientation === "vertical" && element.orientation !== "vertical") ||
                    (orientation === "horizontal" && element.orientation === "vertical")
                  if (swapAxes) {
                    patch({
                      orientation,
                      width: element.height,
                      height: element.width,
                    })
                  } else {
                    patch({ orientation })
                  }
                }}
              >
                <option value="horizontal">Gorizontal</option>
                <option value="vertical">Vertikal</option>
              </select>
            </div>
          ) : null}
          <NumberField
            label="Chiziq qalinligi (mm)"
            value={element.strokeWidth ?? 0.3}
            onChange={(strokeWidth) => patch({ strokeWidth })}
          />
          <ZIndexField value={element.zIndex ?? 1} onChange={(zIndex) => patch({ zIndex })} />
          {element.type === "rect" && (
            <div className="space-y-1">
              <Label className="text-xs">Fon rangi</Label>
              <div className="flex items-center gap-2">
                <input
                  type="color"
                  className="h-8 w-12 cursor-pointer rounded border border-input"
                  value={element.fillColor?.match(/^#[0-9a-f]{6}$/i) ? element.fillColor : "#ffff00"}
                  onChange={(e) => patch({ fillColor: e.target.value })}
                />
                <Input
                  value={element.fillColor ?? ""}
                  onChange={(e) => patch({ fillColor: e.target.value || undefined })}
                  className="h-8"
                  placeholder="#FFFF00 yoki bo'sh"
                />
              </div>
            </div>
          )}
        </PropertiesSection>
      )}
    </div>
  )
}

function TableProperties({
  element,
  onChange,
  grouped,
}: {
  element: LabelElement
  onChange: (element: LabelElement) => void
  grouped: boolean
}) {
  const rows = element.rows ?? 2
  const cols = element.cols ?? 3
  const cells = element.cells ?? createTableCells(rows, cols)
  const [selectedRow, setSelectedRow] = useState(0)
  const [selectedCol, setSelectedCol] = useState(0)

  const safeRow = Math.min(selectedRow, rows - 1)
  const safeCol = Math.min(selectedCol, cols - 1)
  const cell = getTableCell(element, safeRow, safeCol)
  const cellSource = inferTableCellDataSource(cell)
  const colWidths = normalizeTableColWidths(cols, element.width, element.colWidths)

  function patchTable(partial: Partial<LabelElement>) {
    onChange(normalizeElement({ ...element, ...partial }))
  }

  function patchCell(partial: Partial<LabelTableCell>) {
    const nextCells = [...cells]
    nextCells[safeRow * cols + safeCol] = normalizeTableCell({ ...cell, ...partial })
    patchTable({ cells: nextCells })
  }

  function setRows(newRows: number) {
    const r = clampTableRows(newRows)
    if (r === rows) {
      return
    }
    const rowHeight = element.height / rows
    patchTable({
      rows: r,
      cells: resizeTableCells(cells, rows, cols, r, cols),
      height: Math.max(rowHeight, Math.round(rowHeight * r * 10) / 10),
    })
    setSelectedRow((prev) => Math.min(prev, r - 1))
  }

  function setCols(newCols: number) {
    const c = clampTableCols(newCols)
    if (c === cols) {
      return
    }
    const resized = resizeTableColWidths(element.colWidths, cols, c, element.width)
    patchTable({
      cols: c,
      cells: resizeTableCells(cells, rows, cols, rows, c),
      colWidths: resized.colWidths,
      width: resized.width,
    })
    setSelectedCol((prev) => Math.min(prev, c - 1))
  }

  function insertRow(atRow: number) {
    const result = insertTableRow(cells, rows, cols, atRow)
    if (!result) {
      return
    }
    const rowHeight = element.height / rows
    patchTable({
      rows: result.rows,
      cells: result.cells,
      height: Math.round((element.height + rowHeight) * 10) / 10,
    })
    setSelectedRow(atRow <= safeRow ? safeRow + 1 : safeRow)
  }

  function insertCol(atCol: number) {
    const result = insertTableCol(cells, rows, cols, atCol)
    if (!result) {
      return
    }
    const insertedWidth = colWidths[Math.min(safeCol, colWidths.length - 1)] ?? element.width / cols
    const widths = insertTableColWidths(element.colWidths, cols, atCol, element.width, insertedWidth)
    patchTable({
      cols: result.cols,
      cells: result.cells,
      colWidths: widths.colWidths,
      width: widths.width,
    })
    setSelectedCol(atCol <= safeCol ? safeCol + 1 : safeCol)
  }

  function deleteRow(atRow: number) {
    const result = removeTableRow(cells, rows, cols, atRow)
    if (!result) {
      return
    }
    const rowHeight = element.height / rows
    patchTable({
      rows: result.rows,
      cells: result.cells,
      height: Math.max(rowHeight, Math.round((element.height - rowHeight) * 10) / 10),
    })
    setSelectedRow((prev) => Math.min(prev, result.rows - 1))
  }

  function deleteCol(atCol: number) {
    const result = removeTableCol(cells, rows, cols, atCol)
    if (!result) {
      return
    }
    const widths = removeTableColWidths(element.colWidths, cols, atCol, element.width)
    patchTable({
      cols: result.cols,
      cells: result.cells,
      colWidths: widths.colWidths,
      width: widths.width,
    })
    setSelectedCol((prev) => Math.min(prev, result.cols - 1))
  }

  function setColWidth(col: number, widthMm: number) {
    const w = Math.max(1, Math.round(widthMm * 10) / 10)
    const next = [...colWidths]
    next[col] = w
    const totalWidth = Math.round(next.reduce((acc, value) => acc + value, 0) * 10) / 10
    patchTable({ colWidths: next, width: totalWidth })
  }

  function equalizeColWidths() {
    patchTable({ colWidths: equalTableColWidths(cols, element.width) })
  }

  function setCellDataSource(dataSource: LabelDataSource) {
    if (dataSource === "static") {
      patchCell({ dataSource: "static", binding: undefined, staticText: cell.staticText ?? "" })
      return
    }
    patchCell({
      dataSource: "backend",
      staticText: undefined,
      binding: cell.binding ?? TEXT_BINDINGS[0]?.key ?? "serial",
    })
  }

  return (
    <>
      <PropertiesSection title="Jadval tuzilmasi" grouped={grouped} defaultOpen>
        <div className="grid grid-cols-2 gap-2">
          <NumberField label="Qatorlar" value={rows} onChange={setRows} />
          <NumberField label="Ustunlar" value={cols} onChange={setCols} />
        </div>

        <div className="space-y-2">
          <div className="text-xs font-semibold text-muted-foreground">Qator / ustun qo&apos;shish</div>
          <div className="grid grid-cols-2 gap-1.5">
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-8 text-xs"
              disabled={rows >= TABLE_ROWS_MAX}
              onClick={() => insertRow(safeRow)}
            >
              Qator yuqoriga
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-8 text-xs"
              disabled={rows >= TABLE_ROWS_MAX}
              onClick={() => insertRow(safeRow + 1)}
            >
              Qator pastga
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-8 text-xs"
              disabled={cols >= TABLE_COLS_MAX}
              onClick={() => insertCol(safeCol)}
            >
              Ustun chapga
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-8 text-xs"
              disabled={cols >= TABLE_COLS_MAX}
              onClick={() => insertCol(safeCol + 1)}
            >
              Ustun o&apos;ngga
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-8 text-xs text-destructive hover:text-destructive"
              disabled={rows <= 1}
              onClick={() => deleteRow(safeRow)}
            >
              Qatorni o&apos;chirish
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-8 text-xs text-destructive hover:text-destructive"
              disabled={cols <= 1}
              onClick={() => deleteCol(safeCol)}
            >
              Ustunni o&apos;chirish
            </Button>
          </div>
          <p className="text-[10px] text-muted-foreground">
            Tanlangan katak: {safeRow + 1}-qator, {safeCol + 1}-ustun
          </p>
        </div>
      </PropertiesSection>

      <PropertiesSection title="Ustun kengliklari" grouped={grouped} defaultOpen={false}>
        <div className="flex items-center justify-end">
          <Button type="button" variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={equalizeColWidths}>
            Tenglashtirish
          </Button>
        </div>
        <NumberField
          label={`${safeCol + 1}-ustun kengligi (mm)`}
          value={colWidths[safeCol] ?? element.width / cols}
          onChange={(width) => setColWidth(safeCol, width)}
        />
        <p className="text-[10px] text-muted-foreground">
          Jami: {colWidths.reduce((acc, w) => acc + w, 0).toFixed(1)} mm (W = {element.width} mm)
        </p>
      </PropertiesSection>

      <PropertiesSection title="Ko'rinish" grouped={grouped} defaultOpen={false}>
        <NumberField
          label="Chiziq qalinligi (mm)"
          value={element.strokeWidth ?? 0.3}
          onChange={(strokeWidth) => patchTable({ strokeWidth })}
        />
        <NumberField
          label="Standart shrift (pt)"
          value={element.fontSize ?? 8}
          onChange={(fontSize) => patchTable({ fontSize })}
        />
        <FontFamilyField
          label="Standart shrift turi"
          value={element.fontFamily ?? "arial"}
          onChange={(fontFamily) => patchTable({ fontFamily })}
        />
        <ZIndexField value={element.zIndex ?? 1} onChange={(zIndex) => patchTable({ zIndex })} />
      </PropertiesSection>

      <PropertiesSection title="Katak ma'lumoti" grouped={grouped} defaultOpen>
        <div className="grid grid-cols-2 gap-2">
          <div className="space-y-1">
            <Label className="text-xs">Qator</Label>
            <select
              className="h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
              value={safeRow}
              onChange={(e) => setSelectedRow(Number(e.target.value))}
            >
              {Array.from({ length: rows }).map((_, i) => (
                <option key={i} value={i}>
                  {i + 1}
                </option>
              ))}
            </select>
          </div>
          <div className="space-y-1">
            <Label className="text-xs">Ustun</Label>
            <select
              className="h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
              value={safeCol}
              onChange={(e) => setSelectedCol(Number(e.target.value))}
            >
              {Array.from({ length: cols }).map((_, i) => (
                <option key={i} value={i}>
                  {i + 1}
                </option>
              ))}
            </select>
          </div>
        </div>

        <DataSourceToggle value={cellSource} onChange={setCellDataSource} />

        {cellSource === "static" ? (
          <div className="space-y-1">
            <Label className="text-xs">Katak matni</Label>
            <Input
              value={cell.staticText ?? ""}
              onChange={(e) => patchCell({ staticText: e.target.value })}
              className="h-8"
            />
          </div>
        ) : (
          <>
            <div className="space-y-1">
              <Label className="text-xs">Backend maydon</Label>
              <BindingSelect
                value={cell.binding ?? ""}
                bindings={TEXT_BINDINGS}
                onChange={(binding) => patchCell({ binding })}
                allowEmpty={false}
              />
            </div>
            {isTodayBinding(cell.binding) ? (
              <DateFormatField
                value={cell.dateFormat ?? element.dateFormat ?? DEFAULT_LABEL_DATE_FORMAT}
                onChange={(dateFormat) => patchCell({ dateFormat })}
              />
            ) : null}
            <div className="space-y-1">
              <Label className="text-xs">Oldingi qism</Label>
              <Input
                value={cell.prefix ?? ""}
                onChange={(e) => patchCell({ prefix: e.target.value })}
                className="h-8"
                placeholder="S/N: "
              />
            </div>
          </>
        )}

        <NumberField
          label="Katak shrifti (pt)"
          value={cell.fontSize ?? element.fontSize ?? 8}
          onChange={(fontSize) => patchCell({ fontSize })}
        />
        <FontFamilyField
          label="Katak shrift turi"
          value={cell.fontFamily ?? element.fontFamily ?? "arial"}
          onChange={(fontFamily) => patchCell({ fontFamily })}
        />
        <div className="space-y-1">
          <Label className="text-xs">Tekislash</Label>
          <select
            className="h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
            value={cell.align ?? "center"}
            onChange={(e) => patchCell({ align: e.target.value as "left" | "center" | "right" })}
          >
            <option value="left">Chap</option>
            <option value="center">Markaz</option>
            <option value="right">O&apos;ng</option>
          </select>
        </div>
        <div className="space-y-1">
          <Label className="text-xs">Fon rangi</Label>
          <div className="flex items-center gap-2">
            <input
              type="color"
              className="h-8 w-12 cursor-pointer rounded border border-input"
              value={cell.fillColor?.match(/^#[0-9a-f]{6}$/i) ? cell.fillColor : "#ffff00"}
              onChange={(e) => patchCell({ fillColor: e.target.value })}
            />
            <Input
              value={cell.fillColor ?? ""}
              onChange={(e) => patchCell({ fillColor: e.target.value || undefined })}
              className="h-8"
              placeholder="Bo'sh = shaffof"
            />
          </div>
        </div>
      </PropertiesSection>
    </>
  )
}

function ImageProperties({
  element,
  onImageFile,
  onChange,
  zIndex,
  onZIndex,
}: {
  element: LabelElement
  onImageFile?: (file: File) => void
  onChange: (partial: Partial<LabelElement>) => void
  zIndex: number
  onZIndex: (zIndex: number) => void
}) {
  const dataSource = inferDataSource(element)
  const usesBrandLogo = dataSource === "backend" && (element.binding ?? "") === "brand_logo"
  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    accept: { "image/*": [".jpg", ".jpeg", ".png", ".webp"] },
    multiple: false,
    noClick: !onImageFile,
    disabled: !onImageFile,
    onDrop: (files) => {
      if (files[0] && onImageFile) {
        onImageFile(files[0])
      }
    },
  })

  return (
    <div className="space-y-2">
      <div className="rounded-md border border-dashed bg-muted/30 px-2 py-1.5 text-[10px] text-muted-foreground">
        Statik rasm (fayl etiketkada saqlanadi)
      </div>
      <label className="flex items-center gap-2 rounded-md border px-2 py-1.5 text-xs">
        <input
          type="checkbox"
          checked={usesBrandLogo}
          onChange={(e) => {
            if (e.target.checked) {
              onChange({ dataSource: "backend", binding: "brand_logo" })
              return
            }
            onChange({ dataSource: "static", binding: undefined })
          }}
        />
        Brand bo'yicha dinamik logo (<code>brand_logo</code>)
      </label>
      {usesBrandLogo ? (
        <p className="text-[10px] text-muted-foreground">
          Agar brand uchun logo topilmasa, quyidagi statik rasm fallback sifatida ishlatiladi.
        </p>
      ) : null}
      {element.src ? (
        <div className="overflow-hidden rounded-md border bg-muted/30 p-2">
          <img src={labelAssetUrl(element.src)} alt="" className="mx-auto max-h-24 object-contain" />
        </div>
      ) : null}

      <div
        {...getRootProps()}
        className={cn(
          "flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed p-4 text-center text-xs text-muted-foreground transition-colors",
          isDragActive && "border-primary bg-primary/5 text-primary",
          !onImageFile && "cursor-not-allowed opacity-50",
        )}
      >
        <input {...getInputProps()} />
        <Upload className="size-5" />
        {isDragActive ? "Rasmni qo'ying..." : "Rasmni sudrab tashlang yoki bosing"}
      </div>
      <ZIndexField value={zIndex} onChange={onZIndex} />
    </div>
  )
}
