import { useCallback, useEffect, useRef, useState } from "react"
import { ArrowDown, ArrowLeft, ArrowRight, ArrowUp, ImagePlus, Move, ZoomIn, ZoomOut } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { getBindingLabel } from "@/lib/label-bindings"
import { pickImageFile, labelAssetUrl } from "@/lib/label-image"
import { LABEL_SAMPLE_DATA, type LabelSampleData, resolveElementText } from "@/lib/label-sample-data"
import type { LabelElement, LabelElementType, LabelTableCell, LabelTemplate } from "@/lib/label-types"
import {
  inferDataSource,
  inferTableCellDataSource,
  isLabelElementType,
  labelFontFamilyCss,
  normalizeTableColWidths,
  LABEL_ELEMENT_DRAG_MIME,
  LABEL_ZOOM_MAX,
  LABEL_ZOOM_MIN,
  LABEL_ZOOM_STEP,
  mmToPx,
  ptToCanvasPx,
  pxToMm,
} from "@/lib/label-types"

type DragOverKind = "image" | "element" | null

type LabelCanvasProps = {
  template: Pick<LabelTemplate, "width_mm" | "height_mm" | "definition">
  selectedId: string | null
  onSelect: (id: string | null) => void
  onUpdateElement: (element: LabelElement) => void
  onOffsetAllElements?: (dx: number, dy: number) => void
  onImageDrop?: (file: File, xMm: number, yMm: number) => void
  onElementDrop?: (type: LabelElementType, xMm: number, yMm: number) => void
  preview?: boolean
  sampleData?: LabelSampleData
}

function clampZoom(value: number): number {
  return Math.min(LABEL_ZOOM_MAX, Math.max(LABEL_ZOOM_MIN, Math.round(value * 100) / 100))
}

/** Matches Go print order: backgrounds first, then text/barcodes on top */
function sortElementsForRender(elements: LabelElement[]): LabelElement[] {
  const backgrounds: LabelElement[] = []
  const foreground: LabelElement[] = []
  for (const el of elements) {
    if (el.type === "text" || el.type === "barcode" || el.type === "datamatrix" || el.type === "qrcode") {
      foreground.push(el)
    } else {
      backgrounds.push(el)
    }
  }
  backgrounds.sort((a, b) => (a.zIndex ?? 0) - (b.zIndex ?? 0))
  foreground.sort((a, b) => (a.zIndex ?? 0) - (b.zIndex ?? 0))
  return [...backgrounds, ...foreground]
}

export function LabelCanvas({
  template,
  selectedId,
  onSelect,
  onUpdateElement,
  onOffsetAllElements,
  onImageDrop,
  onElementDrop,
  preview = true,
  sampleData,
}: LabelCanvasProps) {
  const canvasRef = useRef<HTMLDivElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const [dragOver, setDragOver] = useState<DragOverKind>(null)
  const [zoom, setZoom] = useState(1)

  const dragRef = useRef<{
    id: string
    startX: number
    startY: number
    origX: number
    origY: number
  } | null>(null)

  const groupDragRef = useRef<{
    startX: number
    startY: number
    lastDx: number
    lastDy: number
  } | null>(null)

  const previewData = sampleData ?? LABEL_SAMPLE_DATA

  const widthPx = mmToPx(template.width_mm, zoom)
  const heightPx = mmToPx(template.height_mm, zoom)

  const zoomIn = useCallback(() => {
    setZoom((z) => clampZoom(z + LABEL_ZOOM_STEP))
  }, [])

  const zoomOut = useCallback(() => {
    setZoom((z) => clampZoom(z - LABEL_ZOOM_STEP))
  }, [])

  const zoomReset = useCallback(() => {
    setZoom(1)
  }, [])

  const getDropPositionMm = useCallback(
    (clientX: number, clientY: number) => {
      const rect = canvasRef.current?.getBoundingClientRect()
      if (!rect) {
        return { x: 5, y: 5 }
      }
      const xPx = Math.max(0, Math.min(clientX - rect.left, widthPx))
      const yPx = Math.max(0, Math.min(clientY - rect.top, heightPx))
      return {
        x: pxToMm(xPx, zoom),
        y: pxToMm(yPx, zoom),
      }
    },
    [widthPx, heightPx, zoom],
  )

  const handleWheel = useCallback((e: React.WheelEvent) => {
    if (!e.ctrlKey) {
      return
    }
    e.preventDefault()
    setZoom((z) => clampZoom(z + (e.deltaY < 0 ? LABEL_ZOOM_STEP : -LABEL_ZOOM_STEP)))
  }, [])

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (!(e.ctrlKey || e.metaKey)) {
        return
      }
      if (e.key === "=" || e.key === "+") {
        e.preventDefault()
        zoomIn()
      } else if (e.key === "-") {
        e.preventDefault()
        zoomOut()
      } else if (e.key === "0") {
        e.preventDefault()
        zoomReset()
      }
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [zoomIn, zoomOut, zoomReset])

  const handleDragOver = useCallback(
    (e: React.DragEvent) => {
      const types = Array.from(e.dataTransfer.types)
      if (onElementDrop && types.includes(LABEL_ELEMENT_DRAG_MIME)) {
        e.preventDefault()
        e.dataTransfer.dropEffect = "copy"
        setDragOver("element")
        return
      }
      if (onImageDrop && types.includes("Files")) {
        e.preventDefault()
        e.dataTransfer.dropEffect = "copy"
        setDragOver("image")
      }
    },
    [onImageDrop, onElementDrop],
  )

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    if (e.currentTarget.contains(e.relatedTarget as Node)) {
      return
    }
    setDragOver(null)
  }, [])

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      const kind = dragOver
      setDragOver(null)

      const { x, y } = getDropPositionMm(e.clientX, e.clientY)

      const elementType = e.dataTransfer.getData(LABEL_ELEMENT_DRAG_MIME)
      if (onElementDrop && isLabelElementType(elementType)) {
        onElementDrop(elementType, x, y)
        return
      }

      if (kind === "image" && onImageDrop) {
        const file = pickImageFile(e.dataTransfer.files)
        if (file) {
          onImageDrop(file, x, y)
        }
      }
    },
    [dragOver, getDropPositionMm, onElementDrop, onImageDrop],
  )

  const handlePointerDown = useCallback(
    (e: React.PointerEvent, el: LabelElement) => {
      e.stopPropagation()
      onSelect(el.id)
      dragRef.current = {
        id: el.id,
        startX: e.clientX,
        startY: e.clientY,
        origX: el.x,
        origY: el.y,
      }
      ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
    },
    [onSelect],
  )

  const handlePointerMove = useCallback(
    (e: React.PointerEvent, el: LabelElement) => {
      if (!dragRef.current || dragRef.current.id !== el.id) {
        return
      }
      const dx = pxToMm(e.clientX - dragRef.current.startX, zoom)
      const dy = pxToMm(e.clientY - dragRef.current.startY, zoom)
      onUpdateElement({
        ...el,
        x: Math.max(0, Math.round((dragRef.current.origX + dx) * 10) / 10),
        y: Math.max(0, Math.round((dragRef.current.origY + dy) * 10) / 10),
      })
    },
    [onUpdateElement, zoom],
  )

  const handlePointerUp = useCallback(() => {
    dragRef.current = null
    groupDragRef.current = null
  }, [])

  const handleGroupPointerDown = useCallback(
    (e: React.PointerEvent) => {
      if (!onOffsetAllElements || template.definition.elements.length === 0) {
        onSelect(null)
        return
      }
      onSelect(null)
      groupDragRef.current = {
        startX: e.clientX,
        startY: e.clientY,
        lastDx: 0,
        lastDy: 0,
      }
      ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
    },
    [onOffsetAllElements, onSelect, template.definition.elements.length],
  )

  const handleGroupPointerMove = useCallback(
    (e: React.PointerEvent) => {
      if (!groupDragRef.current || !onOffsetAllElements) {
        return
      }
      const dx = pxToMm(e.clientX - groupDragRef.current.startX, zoom)
      const dy = pxToMm(e.clientY - groupDragRef.current.startY, zoom)
      const stepDx = dx - groupDragRef.current.lastDx
      const stepDy = dy - groupDragRef.current.lastDy
      if (stepDx === 0 && stepDy === 0) {
        return
      }
      groupDragRef.current.lastDx = dx
      groupDragRef.current.lastDy = dy
      onOffsetAllElements(stepDx, stepDy)
    },
    [onOffsetAllElements, zoom],
  )

  const nudgeAll = useCallback(
    (dx: number, dy: number) => {
      onOffsetAllElements?.(dx, dy)
    },
    [onOffsetAllElements],
  )

  const sorted = sortElementsForRender(template.definition.elements)

  return (
    <div className="flex flex-col">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border/50 px-3 py-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs text-muted-foreground">Masshtab</span>
          <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="size-8"
            onClick={zoomOut}
            disabled={zoom <= LABEL_ZOOM_MIN}
            title="Kichiklashtirish (Ctrl+-)"
          >
            <ZoomOut className="size-4" />
          </Button>
          <button
            type="button"
            className="min-w-[4.5rem] rounded-md px-2 py-1 text-xs font-medium tabular-nums hover:bg-muted"
            onClick={zoomReset}
            title="100% (Ctrl+0)"
          >
            {Math.round(zoom * 100)}%
          </button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="size-8"
            onClick={zoomIn}
            disabled={zoom >= LABEL_ZOOM_MAX}
            title="Kattalashtirish (Ctrl++)"
          >
            <ZoomIn className="size-4" />
          </Button>
          </div>
        </div>
        {onOffsetAllElements ? (
          <div className="flex flex-wrap items-center gap-1">
            <span className="mr-1 flex items-center gap-1 text-xs text-muted-foreground">
              <Move className="size-3.5" />
              Barcha elementlar
            </span>
            <Button type="button" variant="outline" size="icon" className="size-8" onClick={() => nudgeAll(-1, 0)} title="Chapga 1 mm">
              <ArrowLeft className="size-4" />
            </Button>
            <Button type="button" variant="outline" size="icon" className="size-8" onClick={() => nudgeAll(1, 0)} title="O'ngga 1 mm">
              <ArrowRight className="size-4" />
            </Button>
            <Button type="button" variant="outline" size="icon" className="size-8" onClick={() => nudgeAll(0, -1)} title="Yuqoriga 1 mm">
              <ArrowUp className="size-4" />
            </Button>
            <Button type="button" variant="outline" size="icon" className="size-8" onClick={() => nudgeAll(0, 1)} title="Pastga 1 mm">
              <ArrowDown className="size-4" />
            </Button>
          </div>
        ) : null}
      </div>

      <div
        ref={scrollRef}
        className="max-h-[min(70vh,720px)] overflow-auto rounded-b-xl bg-muted/30 p-6"
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onWheel={handleWheel}
      >
        <div
          ref={canvasRef}
          className={cn(
            "relative mx-auto bg-white shadow-md transition-shadow",
            dragOver && "ring-2 ring-primary ring-offset-2",
          )}
          style={{ width: widthPx, height: heightPx }}
          onClick={() => onSelect(null)}
        >
          <div
            className={cn(
              "absolute inset-0 z-0",
              onOffsetAllElements && template.definition.elements.length > 0 ? "cursor-grab active:cursor-grabbing" : "",
            )}
            onPointerDown={handleGroupPointerDown}
            onPointerMove={handleGroupPointerMove}
            onPointerUp={handlePointerUp}
          />
          <div
            className="pointer-events-none absolute inset-0 opacity-20"
            style={{
              backgroundImage:
                "linear-gradient(to right, #ccc 1px, transparent 1px), linear-gradient(to bottom, #ccc 1px, transparent 1px)",
              backgroundSize: `${mmToPx(5, zoom)}px ${mmToPx(5, zoom)}px`,
            }}
          />

          {dragOver && (
            <div className="pointer-events-none absolute inset-0 z-50 flex flex-col items-center justify-center gap-2 bg-primary/10 text-primary">
              {dragOver === "image" ? (
                <>
                  <ImagePlus className="size-8" />
                  <span className="text-sm font-medium">Rasmni shu yerga tashlang</span>
                </>
              ) : (
                <span className="text-sm font-medium">Elementni shu yerga tashlang</span>
              )}
            </div>
          )}

          {sorted.map((el) => (
            <div
              key={el.id}
              className={cn(
                "absolute cursor-move select-none overflow-hidden border border-transparent",
                selectedId === el.id && "border-primary ring-1 ring-primary",
              )}
              style={{
                left: mmToPx(el.x, zoom),
                top: mmToPx(el.y, zoom),
                width: mmToPx(el.width, zoom),
                height: Math.max(mmToPx(el.height, zoom), 2),
                zIndex: el.zIndex ?? 1,
              }}
              onPointerDown={(e) => handlePointerDown(e, el)}
              onPointerMove={(e) => handlePointerMove(e, el)}
              onPointerUp={handlePointerUp}
              onClick={(e) => e.stopPropagation()}
            >
              <ElementPreview element={el} preview={preview} zoom={zoom} sampleData={previewData} />
            </div>
          ))}

          <div className="pointer-events-none absolute bottom-1 right-2 text-[10px] text-muted-foreground">
            {template.width_mm}×{template.height_mm} mm
          </div>
        </div>

        {(onImageDrop || onElementDrop) && !dragOver && (
          <p className="mt-3 text-center text-xs text-muted-foreground">
            Element yoki rasmni maket ustiga sudrab tashlang · bo&apos;sh joydan sudrab barcha elementlarni siljiting · Ctrl + g&apos;ildirak = zoom
          </p>
        )}
      </div>
    </div>
  )
}

function ElementPreview({
  element,
  preview,
  zoom,
  sampleData,
}: {
  element: LabelElement
  preview: boolean
  zoom: number
  sampleData: LabelSampleData
}) {
  if (element.type === "text") {
    const dataSource = inferDataSource(element)
    const text = preview
      ? resolveElementText(
          sampleData,
          dataSource,
          element.staticText,
          element.binding,
          element.prefix,
          element.dateFormat,
        )
      : dataSource === "static"
        ? element.staticText ?? "Statik matn"
        : element.binding
          ? `{{${element.binding}}}`
          : "Backend"

    const fontPx = ptToCanvasPx(element.fontSize ?? 10, zoom)

    return (
      <div className="relative h-full w-full">
        <span
          className={cn(
            "absolute -top-2 left-0 z-10 rounded px-0.5 font-medium leading-none",
            dataSource === "static"
              ? "bg-amber-100 text-amber-800"
              : "bg-sky-100 text-sky-800",
          )}
          style={{ fontSize: Math.max(7, 7 * zoom) }}
        >
          {dataSource === "static" ? "S" : "B"}
        </span>
        <div
          className="flex h-full w-full items-center overflow-hidden px-0.5 leading-tight text-foreground"
          style={{
            fontSize: `${fontPx}px`,
            fontFamily: labelFontFamilyCss(element.fontFamily),
            fontWeight: element.fontWeight ?? "normal",
            textAlign: element.align ?? "left",
            justifyContent:
              element.align === "center" ? "center" : element.align === "right" ? "flex-end" : "flex-start",
            whiteSpace: "pre-wrap",
          }}
        >
          {text}
        </div>
      </div>
    )
  }

  if (element.type === "barcode") {
    return (
      <div className="relative flex h-full w-full flex-col items-center justify-center bg-white">
        <span
          className="absolute -top-2 left-0 z-10 rounded bg-sky-100 px-0.5 font-medium text-sky-800"
          style={{ fontSize: Math.max(7, 7 * zoom) }}
        >
          B
        </span>
        <div className="flex h-3/4 w-[90%] items-end justify-center gap-px">
          {Array.from({ length: 24 }).map((_, i) => (
            <div key={i} className="bg-black" style={{ width: 2, height: `${40 + (i % 5) * 12}%` }} />
          ))}
        </div>
        <span className="mt-0.5 text-muted-foreground" style={{ fontSize: Math.max(8, 8 * zoom) }}>
          {element.binding ? getBindingLabel(element.binding) : element.format?.toUpperCase() ?? "BARCODE"}
        </span>
      </div>
    )
  }

  if (element.type === "datamatrix") {
    return (
      <div className="relative h-full w-full">
        <span
          className="absolute -top-2 left-0 z-10 rounded bg-sky-100 px-0.5 font-medium text-sky-800"
          style={{ fontSize: Math.max(7, 7 * zoom) }}
        >
          B
        </span>
        <div className="grid h-full w-full grid-cols-6 grid-rows-6 gap-px bg-white p-0.5">
          {Array.from({ length: 36 }).map((_, i) => (
            <div key={i} className={cn("bg-black", i % 3 === 0 ? "opacity-100" : "opacity-30")} />
          ))}
        </div>
      </div>
    )
  }

  if (element.type === "qrcode") {
    return (
      <div className="relative h-full w-full bg-white p-1">
        <span
          className="absolute -top-2 left-0 z-10 rounded bg-sky-100 px-0.5 font-medium text-sky-800"
          style={{ fontSize: Math.max(7, 7 * zoom) }}
        >
          B
        </span>
        <div className="relative h-full w-full">
          {[
            { left: "0%", top: "0%" },
            { left: "70%", top: "0%" },
            { left: "0%", top: "70%" },
          ].map((pos, index) => (
            <div
              key={index}
              className="absolute border-2 border-black bg-white"
              style={{
                left: pos.left,
                top: pos.top,
                width: "22%",
                height: "22%",
              }}
            >
              <div className="absolute inset-[18%] bg-black" />
            </div>
          ))}
          <div className="grid h-full w-full grid-cols-8 grid-rows-8 gap-px p-0.5">
            {Array.from({ length: 64 }).map((_, i) => (
              <div key={i} className={cn("bg-black", (i + i * 3) % 5 === 0 ? "opacity-100" : "opacity-20")} />
            ))}
          </div>
        </div>
      </div>
    )
  }

  if (element.type === "image") {
    if (element.src) {
      return (
        <img
          src={labelAssetUrl(element.src)}
          alt=""
          className="pointer-events-none h-full w-full object-contain"
          draggable={false}
        />
      )
    }
    return (
      <div
        className="flex h-full w-full flex-col items-center justify-center gap-1 bg-muted text-muted-foreground"
        style={{ fontSize: Math.max(10, 10 * zoom) }}
      >
        <ImagePlus className="size-4 opacity-50" style={{ width: 16 * zoom, height: 16 * zoom }} />
        Rasm
      </div>
    )
  }

  if (element.type === "line") {
    return (
      <div
        className="w-full bg-foreground"
        style={{
          height: Math.max(mmToPx(element.strokeWidth ?? 0.3, zoom), 1),
          marginTop: "auto",
          marginBottom: "auto",
        }}
      />
    )
  }

  if (element.type === "rect") {
    return (
      <div
        className="h-full w-full border border-foreground"
        style={{
          backgroundColor: element.fillColor ?? "transparent",
          borderWidth: Math.max(mmToPx(element.strokeWidth ?? 0.3, zoom), 1),
        }}
      />
    )
  }

  if (element.type === "table") {
    const rows = element.rows ?? 2
    const cols = element.cols ?? 3
    const strokePx = Math.max(mmToPx(element.strokeWidth ?? 0.3, zoom), 1)
    const defaultFontPt = element.fontSize ?? 8
    const colWidths = normalizeTableColWidths(cols, element.width, element.colWidths)
    const gridColumns = colWidths.map((w) => `${(w / element.width) * 100}%`).join(" ")

    return (
      <div
        className="grid h-full w-full overflow-hidden border border-foreground"
        style={{
          gridTemplateRows: `repeat(${rows}, 1fr)`,
          gridTemplateColumns: gridColumns,
          borderWidth: strokePx,
        }}
      >
        {Array.from({ length: rows * cols }).map((_, index) => {
          const row = Math.floor(index / cols)
          const col = index % cols
          const cell = element.cells?.[index] ?? { staticText: "" }
          const text = tableCellDisplayText(cell, preview, element.dateFormat, sampleData)
          const fontPt = cell.fontSize ?? defaultFontPt
          const fontPx = ptToCanvasPx(fontPt, zoom)
          const dataSource = inferTableCellDataSource(cell)

          return (
            <div
              key={index}
              className={cn(
                "relative flex min-h-0 min-w-0 items-center overflow-hidden border-foreground px-0.5 leading-tight text-foreground",
                row < rows - 1 && "border-b",
                col < cols - 1 && "border-r",
              )}
              style={{
                backgroundColor: cell.fillColor ?? "transparent",
                borderWidth: strokePx,
                fontSize: `${fontPx}px`,
                fontFamily: labelFontFamilyCss(cell.fontFamily ?? element.fontFamily),
                fontWeight: cell.fontWeight ?? "normal",
                textAlign: cell.align ?? "center",
                justifyContent:
                  cell.align === "left" ? "flex-start" : cell.align === "right" ? "flex-end" : "center",
              }}
            >
              {!preview && dataSource === "backend" && cell.binding ? (
                <span
                  className="absolute z-10 rounded bg-sky-100 px-0.5 font-medium text-sky-800"
                  style={{ fontSize: Math.max(6, 6 * zoom), marginTop: -8 * zoom }}
                >
                  B
                </span>
              ) : null}
              <span className="truncate">{text}</span>
            </div>
          )
        })}
      </div>
    )
  }

  return null
}

function tableCellDisplayText(
  cell: LabelTableCell,
  preview: boolean,
  defaultDateFormat: string | undefined,
  sampleData: LabelSampleData,
): string {
  const dataSource = inferTableCellDataSource(cell)
  if (preview) {
    return resolveElementText(
      sampleData,
      dataSource,
      cell.staticText,
      cell.binding,
      cell.prefix,
      cell.dateFormat ?? defaultDateFormat,
    )
  }
  if (dataSource === "backend" && cell.binding) {
    return `{{${cell.binding}}}`
  }
  return cell.staticText ?? ""
}
