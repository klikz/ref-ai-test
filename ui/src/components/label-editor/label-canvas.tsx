import { useCallback, useEffect, useRef, useState, type CSSProperties, type PointerEvent as ReactPointerEvent } from "react"
import { ArrowDown, ArrowLeft, ArrowRight, ArrowUp, ImagePlus, Minus, Move, Plus, ZoomIn, ZoomOut } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { getBindingLabel } from "@/lib/label-bindings"
import { pickImageFile, labelAssetUrl } from "@/lib/label-image"
import { LABEL_SAMPLE_DATA, type LabelSampleData, resolveElementText } from "@/lib/label-sample-data"
import type { LabelElement, LabelElementType, LabelResizeHandle, LabelTableCell, LabelTemplate } from "@/lib/label-types"
import {
  expandSelectionWithGroups,
  inferDataSource,
  inferTableCellDataSource,
  isLabelCornerResizeHandle,
  isLabelPaletteDropType,
  labelFontFamilyCss,
  normalizeElementRotationDeg,
  normalizeTableColWidths,
  placeBoxKeepingCenter,
  resizeLabelElementBox,
  screenDeltaToLocalElementDelta,
  LABEL_ELEMENT_DRAG_MIME,
  LABEL_ELEMENT_SCALE_STEP,
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
  selectedIds: string[]
  onSelectIds: (ids: string[]) => void
  onApplyDragFromOrigins: (
    origins: Record<string, { x: number; y: number }>,
    dx: number,
    dy: number,
  ) => void
  onUpdateElement: (element: LabelElement) => void
  onOffsetAllElements?: (dx: number, dy: number) => void
  onScaleAllElements?: (factor: number) => void
  onImageDrop?: (file: File, xMm: number, yMm: number) => void
  onElementDrop?: (type: LabelElementType | "vline", xMm: number, yMm: number) => void
  preview?: boolean
  sampleData?: LabelSampleData
}

const RESIZE_HANDLES: {
  id: LabelResizeHandle
  cursor: string
  style: CSSProperties
}[] = [
  { id: "nw", cursor: "nwse-resize", style: { left: 0, top: 0, transform: "translate(-50%, -50%)" } },
  { id: "n", cursor: "ns-resize", style: { left: "50%", top: 0, transform: "translate(-50%, -50%)" } },
  { id: "ne", cursor: "nesw-resize", style: { right: 0, top: 0, transform: "translate(50%, -50%)" } },
  { id: "e", cursor: "ew-resize", style: { right: 0, top: "50%", transform: "translate(50%, -50%)" } },
  { id: "se", cursor: "nwse-resize", style: { right: 0, bottom: 0, transform: "translate(50%, 50%)" } },
  { id: "s", cursor: "ns-resize", style: { left: "50%", bottom: 0, transform: "translate(-50%, 50%)" } },
  { id: "sw", cursor: "nesw-resize", style: { left: 0, bottom: 0, transform: "translate(-50%, 50%)" } },
  { id: "w", cursor: "ew-resize", style: { left: 0, top: "50%", transform: "translate(-50%, -50%)" } },
]

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
  selectedIds,
  onSelectIds,
  onApplyDragFromOrigins,
  onUpdateElement,
  onOffsetAllElements,
  onScaleAllElements,
  onImageDrop,
  onElementDrop,
  preview = true,
  sampleData,
}: LabelCanvasProps) {
  const canvasRef = useRef<HTMLDivElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const [dragOver, setDragOver] = useState<DragOverKind>(null)
  const [zoom, setZoom] = useState(1)

  const selectedSet = useRef(new Set(selectedIds))
  selectedSet.current = new Set(selectedIds)

  const dragRef = useRef<{
    startX: number
    startY: number
    origins: Record<string, { x: number; y: number }>
  } | null>(null)

  const resizeRef = useRef<{
    id: string
    handle: LabelResizeHandle
    startX: number
    startY: number
    origin: { x: number; y: number; width: number; height: number }
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
      if (onElementDrop && isLabelPaletteDropType(elementType)) {
        onElementDrop(elementType as LabelElementType | "vline", x, y)
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

  const resolveClickSelection = useCallback(
    (el: LabelElement, additive: boolean): string[] => {
      const elements = template.definition.elements
      const clickedIds = expandSelectionWithGroups(elements, [el.id])

      if (!additive) {
        return clickedIds
      }

      const current = new Set(selectedSet.current)
      const allSelected = clickedIds.every((id) => current.has(id))
      if (allSelected) {
        for (const id of clickedIds) {
          current.delete(id)
        }
      } else {
        for (const id of clickedIds) {
          current.add(id)
        }
      }
      return [...current]
    },
    [template.definition.elements],
  )

  const handlePointerDown = useCallback(
    (e: ReactPointerEvent, el: LabelElement) => {
      e.stopPropagation()
      const additive = e.ctrlKey || e.metaKey || e.shiftKey
      let nextIds = resolveClickSelection(el, additive)

      // Dragging an already-selected member keeps the whole selection.
      if (!additive && selectedSet.current.has(el.id) && selectedSet.current.size > 1) {
        nextIds = [...selectedSet.current]
      } else if (!additive && el.groupId) {
        nextIds = expandSelectionWithGroups(template.definition.elements, [el.id])
      }

      onSelectIds(nextIds)

      const origins: Record<string, { x: number; y: number }> = {}
      const dragIds = nextIds.length > 0 ? nextIds : [el.id]
      for (const item of template.definition.elements) {
        if (dragIds.includes(item.id)) {
          origins[item.id] = { x: item.x, y: item.y }
        }
      }
      // Ensure the clicked element is always in the drag set.
      if (!origins[el.id]) {
        origins[el.id] = { x: el.x, y: el.y }
      }

      dragRef.current = {
        startX: e.clientX,
        startY: e.clientY,
        origins,
      }
      ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
    },
    [onSelectIds, resolveClickSelection, template.definition.elements],
  )

  const handlePointerMove = useCallback(
    (e: ReactPointerEvent) => {
      if (resizeRef.current) {
        const screenDx = pxToMm(e.clientX - resizeRef.current.startX, zoom)
        const screenDy = pxToMm(e.clientY - resizeRef.current.startY, zoom)
        const el = template.definition.elements.find((item) => item.id === resizeRef.current?.id)
        if (!el) {
          return
        }
        const rotationDeg = normalizeElementRotationDeg(el.rotationDeg)
        const { dx, dy } =
          rotationDeg === 0
            ? { dx: screenDx, dy: screenDy }
            : screenDeltaToLocalElementDelta(screenDx, screenDy, rotationDeg)
        let nextBox = resizeLabelElementBox(
          resizeRef.current.origin,
          resizeRef.current.handle,
          dx,
          dy,
          template.width_mm,
          template.height_mm,
          isLabelCornerResizeHandle(resizeRef.current.handle),
        )
        if (rotationDeg !== 0) {
          nextBox = placeBoxKeepingCenter(
            resizeRef.current.origin,
            nextBox.width,
            nextBox.height,
            template.width_mm,
            template.height_mm,
          )
        }
        onUpdateElement({
          ...el,
          ...nextBox,
        })
        return
      }
      if (!dragRef.current) {
        return
      }
      const dx = pxToMm(e.clientX - dragRef.current.startX, zoom)
      const dy = pxToMm(e.clientY - dragRef.current.startY, zoom)
      onApplyDragFromOrigins(dragRef.current.origins, dx, dy)
    },
    [onApplyDragFromOrigins, onUpdateElement, template.definition.elements, template.height_mm, template.width_mm, zoom],
  )

  const handlePointerUp = useCallback(() => {
    dragRef.current = null
    resizeRef.current = null
    groupDragRef.current = null
  }, [])

  const handleResizePointerDown = useCallback(
    (e: ReactPointerEvent, el: LabelElement, handle: LabelResizeHandle) => {
      e.stopPropagation()
      e.preventDefault()
      onSelectIds([el.id])
      dragRef.current = null
      resizeRef.current = {
        id: el.id,
        handle,
        startX: e.clientX,
        startY: e.clientY,
        origin: { x: el.x, y: el.y, width: el.width, height: el.height },
      }
      ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
    },
    [onSelectIds],
  )

  const handleGroupPointerDown = useCallback(
    (e: ReactPointerEvent) => {
      if (!onOffsetAllElements || template.definition.elements.length === 0) {
        onSelectIds([])
        return
      }
      onSelectIds([])
      groupDragRef.current = {
        startX: e.clientX,
        startY: e.clientY,
        lastDx: 0,
        lastDy: 0,
      }
      ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
    },
    [onOffsetAllElements, onSelectIds, template.definition.elements.length],
  )

  const handleGroupPointerMove = useCallback(
    (e: ReactPointerEvent) => {
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
  const selectedLookup = new Set(selectedIds)

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
        {onOffsetAllElements || onScaleAllElements ? (
          <div className="flex flex-wrap items-center gap-1">
            <span className="mr-1 flex items-center gap-1 text-xs text-muted-foreground">
              <Move className="size-3.5" />
              Barcha elementlar
            </span>
            {onScaleAllElements ? (
              <>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  className="size-8"
                  disabled={template.definition.elements.length === 0}
                  onClick={() => onScaleAllElements(1 / LABEL_ELEMENT_SCALE_STEP)}
                  title="Elementlarni kichraytirish (10%)"
                >
                  <Minus className="size-4" />
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  className="size-8"
                  disabled={template.definition.elements.length === 0}
                  onClick={() => onScaleAllElements(LABEL_ELEMENT_SCALE_STEP)}
                  title="Elementlarni kattalashtirish (10%)"
                >
                  <Plus className="size-4" />
                </Button>
              </>
            ) : null}
            {onOffsetAllElements ? (
              <>
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
              </>
            ) : null}
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
          onClick={() => onSelectIds([])}
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

          {sorted.map((el) => {
            const isSelected = selectedLookup.has(el.id)
            const showResizeHandles = isSelected && selectedIds.length === 1
            return (
            <div
              key={el.id}
              className={cn(
                "absolute cursor-move select-none border border-transparent",
                showResizeHandles ? "overflow-visible" : "overflow-hidden",
                isSelected && "border-primary ring-1 ring-primary",
              )}
              style={{
                left: mmToPx(el.x, zoom),
                top: mmToPx(el.y, zoom),
                width: mmToPx(el.width, zoom),
                height: Math.max(mmToPx(el.height, zoom), 2),
                zIndex: el.zIndex ?? 1,
                transform: el.rotationDeg ? `rotate(${el.rotationDeg}deg)` : undefined,
                transformOrigin: "center center",
              }}
              onPointerDown={(e) => handlePointerDown(e, el)}
              onPointerMove={handlePointerMove}
              onPointerUp={handlePointerUp}
              onClick={(e) => e.stopPropagation()}
            >
              <div className="h-full w-full overflow-hidden">
                <ElementPreview element={el} preview={preview} zoom={zoom} sampleData={previewData} />
              </div>
              {showResizeHandles
                ? RESIZE_HANDLES.map((handle) => (
                    <button
                      key={handle.id}
                      type="button"
                      aria-label={`Resize ${handle.id}`}
                      title={
                        isLabelCornerResizeHandle(handle.id)
                          ? "Burchak: proporsiya saqlanadi"
                          : "Yon: erkin o'lcham"
                      }
                      className={cn(
                        "absolute z-20 flex size-3 items-center justify-center rounded-sm border border-primary bg-background shadow-sm",
                        "hover:bg-primary hover:text-primary-foreground",
                        isLabelCornerResizeHandle(handle.id) && "size-3.5 rounded-[2px]",
                      )}
                      style={{ ...handle.style, cursor: handle.cursor }}
                      onPointerDown={(e) => handleResizePointerDown(e, el, handle.id)}
                      onPointerMove={handlePointerMove}
                      onPointerUp={handlePointerUp}
                      onClick={(e) => e.stopPropagation()}
                    >
                      <span
                        className={cn(
                          "block bg-primary",
                          handle.id === "n" || handle.id === "s"
                            ? "h-0.5 w-2"
                            : handle.id === "e" || handle.id === "w"
                              ? "h-2 w-0.5"
                              : "size-1.5 rotate-45",
                        )}
                      />
                    </button>
                  ))
                : null}
            </div>
            )
          })}

          <div className="pointer-events-none absolute bottom-1 right-2 text-[10px] text-muted-foreground">
            {template.width_mm}×{template.height_mm} mm
          </div>
        </div>

        {(onImageDrop || onElementDrop) && !dragOver && (
          <p className="mt-3 text-center text-xs text-muted-foreground">
            Element yoki rasmni maket ustiga sudrab tashlang · tanlanganda burchak/yon tutqichlar =
            o&apos;lcham (burchak = proporsiya) · Ctrl+click = ko&apos;p tanlov · Ctrl+G = guruhlash ·
            bo&apos;sh joydan sudrab barcha elementlarni siljiting · Ctrl + g&apos;ildirak = zoom
          </p>
        )}
      </div>
    </div>
  )
}

function SqueezeText({
  text,
  fontPx,
  fontFamily,
  fontWeight,
  align,
  className,
}: {
  text: string
  fontPx: number
  fontFamily: string
  fontWeight: string | number
  align: "left" | "center" | "right"
  className?: string
}) {
  const boxRef = useRef<HTMLDivElement>(null)
  const measureRef = useRef<HTMLSpanElement>(null)
  const [scaleX, setScaleX] = useState(1)

  const recompute = useCallback(() => {
    const measure = measureRef.current
    const box = boxRef.current
    if (!measure || !box) {
      return
    }
    // Subtract horizontal padding (px-0.5 ≈ 2px each side) so scale matches the
    // printable content width inside the clipped box.
    const padX = 2
    const available = Math.max(0, box.clientWidth - padX * 2)
    const needed = measure.scrollWidth
    if (available <= 0 || needed <= 0) {
      setScaleX(1)
      return
    }
    setScaleX(needed > available ? Math.max(0.15, available / needed) : 1)
  }, [])

  useEffect(() => {
    recompute()
  }, [text, fontPx, fontFamily, fontWeight, align, recompute])

  useEffect(() => {
    const box = boxRef.current
    if (!box || typeof ResizeObserver === "undefined") {
      return
    }
    const ro = new ResizeObserver(() => {
      recompute()
    })
    ro.observe(box)
    return () => ro.disconnect()
  }, [recompute])

  const origin = align === "center" ? "center" : align === "right" ? "right" : "left"

  return (
    <div
      ref={boxRef}
      className={cn(
        "relative flex h-full w-full items-center overflow-hidden px-0.5 leading-tight text-foreground",
        className,
      )}
      style={{
        justifyContent: align === "center" ? "center" : align === "right" ? "flex-end" : "flex-start",
      }}
    >
      {/* Unconstrained measure: must not inherit maxWidth or the live scaleX. */}
      <span
        ref={measureRef}
        aria-hidden
        className="pointer-events-none absolute left-0 top-0 whitespace-pre opacity-0"
        style={{
          fontSize: `${fontPx}px`,
          fontFamily,
          fontWeight,
          visibility: "hidden",
          maxWidth: "none",
          width: "max-content",
        }}
      >
        {text}
      </span>
      <span
        style={{
          fontSize: `${fontPx}px`,
          fontFamily,
          fontWeight,
          textAlign: align,
          whiteSpace: "pre",
          display: "inline-block",
          transform: scaleX < 1 ? `scaleX(${scaleX})` : undefined,
          transformOrigin: origin,
        }}
      >
        {text}
      </span>
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
    const align = element.align ?? "left"

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
        <SqueezeText
          text={text}
          fontPx={fontPx}
          fontFamily={labelFontFamilyCss(element.fontFamily)}
          fontWeight={element.fontWeight ?? "normal"}
          align={align}
        />
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
    const vertical = element.orientation === "vertical"
    return (
      <div
        className="bg-foreground"
        style={
          vertical
            ? {
                width: Math.max(mmToPx(element.strokeWidth ?? 0.3, zoom), 1),
                height: "100%",
                marginLeft: "auto",
                marginRight: "auto",
              }
            : {
                width: "100%",
                height: Math.max(mmToPx(element.strokeWidth ?? 0.3, zoom), 1),
                marginTop: "auto",
                marginBottom: "auto",
              }
        }
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
          const align = cell.align ?? "center"

          return (
            <div
              key={index}
              className={cn(
                "relative min-h-0 min-w-0 overflow-hidden border-foreground leading-tight text-foreground",
                row < rows - 1 && "border-b",
                col < cols - 1 && "border-r",
              )}
              style={{
                backgroundColor: cell.fillColor ?? "transparent",
                borderWidth: strokePx,
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
              <SqueezeText
                text={text}
                fontPx={fontPx}
                fontFamily={labelFontFamilyCss(cell.fontFamily ?? element.fontFamily)}
                fontWeight={cell.fontWeight ?? "normal"}
                align={align}
              />
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
