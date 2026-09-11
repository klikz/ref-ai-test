import { useCallback, useEffect, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import { ArrowLeft, Loader2, Save, Trash2, Upload } from "lucide-react"
import { toast } from "sonner"

import { ElementPalette } from "@/components/label-editor/element-palette"
import { ElementProperties } from "@/components/label-editor/element-properties"
import { LabelCanvas } from "@/components/label-editor/label-canvas"
import { LabelPrinterPreview } from "@/components/label-editor/label-printer-preview"
import { TableSizeDialog } from "@/components/label-editor/table-size-dialog"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  applyLabelElementDragFromOrigins,
  createDefaultElement,
  createElementId,
  createTableElement,
  createVerticalLineElement,
  DEFAULT_LABEL_DEFINITION,
  expandSelectionWithGroups,
  groupLabelElements,
  normalizeElement,
  offsetAllLabelElements,
  parseDefinition,
  scaleAllLabelElements,
  scaleTableColWidthsToWidth,
  selectionHasGroup,
  ungroupLabelElements,
  type LabelDefinition,
  type LabelElement,
  type LabelElementType,
  type LabelPrintRotationDeg,
  type LabelTemplate,
} from "@/lib/label-types"
import { calcImageSizeMm, fileToDataUrl, getImageDimensions } from "@/lib/label-image"
import { buildLabelPreviewData } from "@/lib/label-sample-data"
import { Backend_Request } from "@/services/backend"

type LineItem = { line_id: number; name: string }
type BrandLogoMap = Record<string, string>

export default function LabelTemplateEditorPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const templateId = Number(id)

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [lines, setLines] = useState<LineItem[]>([])

  const [name, setName] = useState("")
  const [lineId, setLineId] = useState(0)
  const [widthMm, setWidthMm] = useState(100)
  const [heightMm, setHeightMm] = useState(50)
  const [dpi, setDpi] = useState(203)
  const [printRotationDeg, setPrintRotationDeg] = useState<LabelPrintRotationDeg>(0)
  const [density, setDensity] = useState(8)
  const [speed, setSpeed] = useState(4)
  const [gapMm, setGapMm] = useState(2)
  const [usePrinterDefaults, setUsePrinterDefaults] = useState(false)
  const [sizeOnly, setSizeOnly] = useState(false)
  const [definition, setDefinition] = useState<LabelDefinition>(DEFAULT_LABEL_DEFINITION)
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [uploadingImage, setUploadingImage] = useState(false)
  const [uploadingBrand, setUploadingBrand] = useState<string | null>(null)
  const [tableDropAt, setTableDropAt] = useState<{ xMm: number; yMm: number } | null>(null)
  const [brands, setBrands] = useState<string[]>([])
  const [brandLogos, setBrandLogos] = useState<BrandLogoMap>({})

  const previewSerial = "ABC123456789"
  const previewAccSerial = "ACC001234"

  const previewSampleData = buildLabelPreviewData({
    serial: previewSerial,
    acc_serial: previewAccSerial,
  })

  const selectedElement =
    selectedIds.length === 1
      ? (definition.elements.find((el) => el.id === selectedIds[0]) ?? null)
      : null
  const canGroup = selectedIds.length >= 2
  const canUngroup = selectionHasGroup(definition.elements, selectedIds)

  const loadTemplate = useCallback(async () => {
    if (!templateId) {
      return
    }
    setLoading(true)
    const result = await Backend_Request<LabelTemplate>({ id: templateId }, "/api/tech/label-templates/get")
    setLoading(false)

    if (result.result === "ok" && result.data) {
      const t = result.data
      setName(t.name)
      setLineId(t.line_id)
      setWidthMm(t.width_mm)
      setHeightMm(t.height_mm)
      setDpi(t.dpi)
      setPrintRotationDeg(t.print_rotation_deg === 90 ? 90 : 0)
      setDensity(typeof t.density === "number" ? t.density : 8)
      setSpeed(typeof t.speed === "number" ? t.speed : 4)
      setGapMm(typeof t.gap_mm === "number" ? t.gap_mm : 2)
      setUsePrinterDefaults(Boolean(t.use_printer_defaults))
      setSizeOnly(Boolean(t.size_only))
      setDefinition(parseDefinition(t.definition))
    } else {
      toast.error(result.error || "Shablon topilmadi")
      navigate("/label-templates")
    }
  }, [templateId, navigate])

  async function loadLines() {
    const result = await Backend_Request<LineItem[]>({}, "/api/lines/all")
    if (result.result === "ok") {
      setLines(result.data ?? [])
    }
  }

  async function loadBrandsAndLogos() {
    const [brandsResult, logosResult] = await Promise.all([
      Backend_Request<string[]>({}, "/api/tech/brands/all"),
      Backend_Request<BrandLogoMap>({}, "/api/tech/brand-logos/all"),
    ])
    if (brandsResult.result === "ok") {
      setBrands(brandsResult.data ?? [])
    }
    if (logosResult.result === "ok") {
      setBrandLogos(logosResult.data ?? {})
    }
  }

  async function saveTemplate() {
    if (!name.trim()) {
      toast.error("Nom kiriting")
      return
    }

    setSaving(true)
    const result = await Backend_Request<LabelTemplate>(
      {
        id: templateId,
        name: name.trim(),
        line_id: lineId,
        width_mm: widthMm,
        height_mm: heightMm,
        dpi,
        print_rotation_deg: printRotationDeg,
        density,
        speed,
        gap_mm: gapMm,
        use_printer_defaults: usePrinterDefaults,
        size_only: sizeOnly,
        definition,
      },
      "/api/tech/label-templates/update",
    )
    setSaving(false)

    if (result.result === "ok") {
      toast.success("Saqlandi")
    } else {
      toast.error(result.error || "Saqlashda xatolik")
    }
  }

  function addElementAt(type: LabelElementType | "vline", xMm: number, yMm: number) {
    if (type === "table") {
      setTableDropAt({ xMm, yMm })
      return
    }
    if (type === "vline") {
      placeElement(createVerticalLineElement(), xMm, yMm)
      return
    }
    placeElement(createDefaultElement(type), xMm, yMm)
  }

  function placeElement(el: LabelElement, xMm: number, yMm: number) {
    el.x = Math.max(0, Math.round((xMm - el.width / 2) * 10) / 10)
    el.y = Math.max(0, Math.round((yMm - el.height / 2) * 10) / 10)
    el.x = Math.min(el.x, Math.max(0, widthMm - el.width))
    el.y = Math.min(el.y, Math.max(0, heightMm - el.height))
    el.zIndex = definition.elements.length + 1
    appendElement(el)
  }

  function confirmTableCreate(rows: number, cols: number) {
    if (!tableDropAt) {
      return
    }
    const el = createTableElement(rows, cols)
    placeElement(el, tableDropAt.xMm, tableDropAt.yMm)
    setTableDropAt(null)
  }

  function appendElement(el: LabelElement) {
    setDefinition((prev) => ({
      ...prev,
      elements: [...prev.elements, el],
    }))
    setSelectedIds([el.id])
  }

  function updateElement(element: LabelElement) {
    setDefinition((prev) => ({
      ...prev,
      elements: prev.elements.map((el) => {
        if (el.id !== element.id) {
          return el
        }
        if (el.type === "table" && element.width !== el.width) {
          return normalizeElement({
            ...element,
            colWidths: scaleTableColWidthsToWidth(el.cols ?? 3, el.width, el.colWidths, element.width),
          })
        }
        return normalizeElement(element)
      }),
    }))
  }

  function offsetAllElements(dx: number, dy: number) {
    if (dx === 0 && dy === 0) {
      return
    }
    setDefinition((prev) => ({
      ...prev,
      elements: offsetAllLabelElements(prev.elements, dx, dy, widthMm, heightMm),
    }))
  }

  function scaleAllElements(factor: number) {
    setDefinition((prev) => ({
      ...prev,
      elements: scaleAllLabelElements(prev.elements, factor, widthMm, heightMm),
    }))
  }

  function applyDragFromOrigins(
    origins: Record<string, { x: number; y: number }>,
    dx: number,
    dy: number,
  ) {
    setDefinition((prev) => ({
      ...prev,
      elements: applyLabelElementDragFromOrigins(prev.elements, origins, dx, dy, widthMm, heightMm),
    }))
  }

  function deleteSelected() {
    if (selectedIds.length === 0) {
      return
    }
    const idSet = new Set(selectedIds)
    setDefinition((prev) => ({
      ...prev,
      elements: prev.elements.filter((el) => !idSet.has(el.id)),
    }))
    setSelectedIds([])
  }

  function groupSelected() {
    if (selectedIds.length < 2) {
      return
    }
    const nextElements = groupLabelElements(definition.elements, selectedIds)
    setDefinition((prev) => ({ ...prev, elements: nextElements }))
    setSelectedIds(expandSelectionWithGroups(nextElements, selectedIds))
  }

  function ungroupSelected() {
    if (selectedIds.length === 0) {
      return
    }
    setDefinition((prev) => ({
      ...prev,
      elements: ungroupLabelElements(prev.elements, selectedIds),
    }))
  }

  async function uploadImageFile(file: File): Promise<string | null> {
    const dataUrl = await fileToDataUrl(file)
    setUploadingImage(true)
    const result = await Backend_Request<string>(
      { template_id: templateId, file64: dataUrl },
      "/api/tech/label-templates/image/upload",
    )
    setUploadingImage(false)

    if (result.result === "ok" && result.data) {
      return result.data
    }
    toast.error(result.error || "Rasm yuklanmadi")
    return null
  }

  async function insertImageAt(file: File, xMm: number, yMm: number, existingId?: string) {
    try {
      const previewUrl = await fileToDataUrl(file)
      const { width: naturalWidth, height: naturalHeight } = await getImageDimensions(previewUrl)
      const size = calcImageSizeMm(naturalWidth, naturalHeight, widthMm * 0.4, heightMm * 0.4)
      const src = await uploadImageFile(file)
      if (!src) {
        return
      }

      if (existingId) {
        setDefinition((prev) => ({
          ...prev,
          elements: prev.elements.map((el) =>
            el.id === existingId ? { ...el, src, width: size.width, height: size.height } : el,
          ),
        }))
        setSelectedIds([existingId])
        return
      }

      const el: LabelElement = {
        id: createElementId(),
        type: "image",
        x: Math.max(0, Math.min(xMm, widthMm - size.width)),
        y: Math.max(0, Math.min(yMm, heightMm - size.height)),
        width: size.width,
        height: size.height,
        src,
        zIndex: definition.elements.length + 1,
      }

      setDefinition((prev) => ({
        ...prev,
        elements: [...prev.elements, el],
      }))
      setSelectedIds([el.id])
      toast.success("Rasm qo'shildi")
    } catch {
      toast.error("Rasmni qayta ishlashda xatolik")
    }
  }

  async function handleCanvasImageDrop(file: File, xMm: number, yMm: number) {
    await insertImageAt(file, xMm, yMm)
  }

  async function handleSelectedImageFile(file: File) {
    const el = definition.elements.find((item) => item.id === selectedIds[0])
    if (el?.type === "image") {
      await insertImageAt(file, el.x, el.y, el.id)
    }
  }

  useEffect(() => {
    void loadTemplate()
    void loadLines()
    void loadBrandsAndLogos()
  }, [loadTemplate])

  async function uploadBrandLogo(brand: string, file: File) {
    try {
      setUploadingBrand(brand)
      const dataUrl = await fileToDataUrl(file)
      const imageResult = await Backend_Request<string>(
        { template_id: templateId, file64: dataUrl },
        "/api/tech/label-templates/image/upload",
      )
      if (imageResult.result !== "ok" || !imageResult.data) {
        toast.error(imageResult.error || "Brand logosi yuklanmadi")
        return
      }
      const upsertResult = await Backend_Request(
        { brand, logo_src: imageResult.data },
        "/api/tech/brand-logos/upsert",
      )
      if (upsertResult.result !== "ok") {
        toast.error(upsertResult.error || "Brand logosi saqlanmadi")
        return
      }
      setBrandLogos((prev) => ({ ...prev, [brand]: imageResult.data as string }))
      toast.success("Brand logosi saqlandi")
    } finally {
      setUploadingBrand(null)
    }
  }

  async function deleteBrandLogo(brand: string) {
    const result = await Backend_Request({ brand }, "/api/tech/brand-logos/delete")
    if (result.result !== "ok") {
      toast.error(result.error || "Brand logosi o'chmadi")
      return
    }
    setBrandLogos((prev) => {
      const next = { ...prev }
      delete next[brand]
      return next
    })
    toast.success("Brand logosi o'chirildi")
  }

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      const target = e.target
      if (target instanceof HTMLElement) {
        const tag = target.tagName
        if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || target.isContentEditable) {
          return
        }
      }

      if (e.key === "Delete" && selectedIds.length > 0) {
        e.preventDefault()
        deleteSelected()
        return
      }

      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "g") {
        e.preventDefault()
        if (e.shiftKey) {
          ungroupSelected()
        } else {
          groupSelected()
        }
      }
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [selectedIds, definition.elements])

  if (loading) {
    return (
      <PageContainer title="Etiketka tahrirlash" description="">
        <div className="flex items-center justify-center gap-2 p-16 text-muted-foreground">
          <Loader2 className="size-6 animate-spin" />
          Yuklanmoqda...
        </div>
      </PageContainer>
    )
  }

  const templatePreview = {
    width_mm: widthMm,
    height_mm: heightMm,
    definition,
  }

  return (
    <PageContainer
      fullWidth
      title="Etiketka tahrirlash"
      description={name}
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" asChild>
            <Link to="/label-templates">
              <ArrowLeft className="size-4" />
              Orqaga
            </Link>
          </Button>
          <Button onClick={() => void saveTemplate()} disabled={saving || uploadingImage} className="gap-2">
            {saving || uploadingImage ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
            Saqlash
          </Button>
        </div>
      }
    >
      <Panel title="Sozlamalar" className="mb-4">
        <div className="space-y-2">
          <div className="grid grid-cols-2 gap-x-3 gap-y-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 xl:grid-cols-7">
            <div className="space-y-0.5 sm:col-span-2">
              <Label className="text-[11px] text-muted-foreground">Nomi</Label>
              <Input className="h-8" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="space-y-0.5">
              <Label className="text-[11px] text-muted-foreground">Liniya</Label>
              <select
                className="flex h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
                value={lineId}
                onChange={(e) => setLineId(Number(e.target.value))}
              >
                {lines.map((line) => (
                  <option key={line.line_id} value={line.line_id}>
                    {line.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-0.5">
              <Label className="text-[11px] text-muted-foreground">Kenglik (mm)</Label>
              <Input className="h-8" type="number" value={widthMm} onChange={(e) => setWidthMm(Number(e.target.value))} />
            </div>
            <div className="space-y-0.5">
              <Label className="text-[11px] text-muted-foreground">Balandlik (mm)</Label>
              <Input className="h-8" type="number" value={heightMm} onChange={(e) => setHeightMm(Number(e.target.value))} />
            </div>
            <div className="space-y-0.5">
              <Label className="text-[11px] text-muted-foreground">DPI</Label>
              <select
                className="flex h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
                value={dpi}
                onChange={(e) => setDpi(Number(e.target.value))}
              >
                <option value={203}>203</option>
                <option value={300}>300</option>
              </select>
            </div>
            <div className="space-y-0.5">
              <Label className="text-[11px] text-muted-foreground">Aylanish</Label>
              <select
                className="flex h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
                value={printRotationDeg}
                onChange={(e) => setPrintRotationDeg(Number(e.target.value) === 90 ? 90 : 0)}
              >
                <option value={0}>0°</option>
                <option value={90}>90°</option>
              </select>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-x-3 gap-y-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6">
            <div className="space-y-0.5">
              <Label className="text-[11px] text-muted-foreground">Density</Label>
              <Input
                className="h-8"
                type="number"
                min={0}
                max={30}
                value={density}
                disabled={usePrinterDefaults || sizeOnly}
                onChange={(e) => {
                  const n = Number(e.target.value)
                  setDensity(Number.isFinite(n) ? Math.max(0, Math.min(30, n)) : 0)
                }}
              />
            </div>
            <div className="space-y-0.5">
              <Label className="text-[11px] text-muted-foreground">Speed</Label>
              <Input
                className="h-8"
                type="number"
                min={1}
                max={14}
                value={speed}
                disabled={usePrinterDefaults || sizeOnly}
                onChange={(e) => setSpeed(Number(e.target.value))}
              />
            </div>
            <div className="space-y-0.5">
              <Label className="text-[11px] text-muted-foreground">Gap (mm)</Label>
              <Input
                className="h-8"
                type="number"
                min={0}
                step={0.5}
                value={gapMm}
                disabled={usePrinterDefaults || sizeOnly}
                onChange={(e) => setGapMm(Number(e.target.value))}
              />
            </div>
            <div className="col-span-2 flex flex-wrap items-end gap-x-4 gap-y-1 pb-1 sm:col-span-3 md:col-span-1 lg:col-span-3">
              <label className="flex h-8 items-center gap-1.5 text-xs whitespace-nowrap">
                <input
                  type="checkbox"
                  checked={usePrinterDefaults}
                  onChange={(e) => setUsePrinterDefaults(e.target.checked)}
                />
                Printer defaults
              </label>
              <label className="flex h-8 items-center gap-1.5 text-xs whitespace-nowrap">
                <input
                  type="checkbox"
                  checked={sizeOnly || usePrinterDefaults}
                  disabled={usePrinterDefaults}
                  onChange={(e) => setSizeOnly(e.target.checked)}
                />
                Faqat o&apos;lcham
              </label>
            </div>
          </div>
        </div>
      </Panel>

      <div className="grid min-h-[520px] gap-4 lg:grid-cols-[280px_1fr_280px]">
        <div className="flex flex-col gap-4">
          <Panel title="Elementlar" className="h-fit">
            <ElementPalette />
            <div className="mt-3 flex gap-3 text-[10px] text-muted-foreground">
              <span className="flex items-center gap-1">
                <span className="rounded bg-amber-100 px-1 font-medium text-amber-800">S</span>
                Statik
              </span>
              <span className="flex items-center gap-1">
                <span className="rounded bg-sky-100 px-1 font-medium text-sky-800">B</span>
                Backend
              </span>
            </div>
          </Panel>
          <Panel title="Brand logotiplari" className="h-fit">
            <div className="space-y-2">
              <p className="text-[10px] text-muted-foreground">
                Bitta shablonda backend image binding sifatida <code>brand_logo</code> ni ishlating.
              </p>
              <div className="max-h-72 space-y-2 overflow-y-auto pr-1">
                {brands.map((brand) => {
                  const logo = brandLogos[brand]
                  const busy = uploadingBrand === brand
                  return (
                    <div key={brand} className="rounded-md border p-2">
                      <div className="mb-2 flex items-center justify-between gap-2">
                        <div className="truncate text-xs font-medium">{brand}</div>
                        {logo ? (
                          <Button
                            size="icon"
                            variant="ghost"
                            className="size-7"
                            onClick={() => void deleteBrandLogo(brand)}
                            disabled={busy}
                          >
                            <Trash2 className="size-4 text-destructive" />
                          </Button>
                        ) : null}
                      </div>
                      {logo ? (
                        <div className="mb-2 overflow-hidden rounded border bg-muted/30 p-2">
                          <img src={logo} alt={brand} className="mx-auto max-h-14 object-contain" />
                        </div>
                      ) : null}
                      <label className="inline-flex cursor-pointer items-center gap-2 text-xs text-primary">
                        {busy ? <Loader2 className="size-4 animate-spin" /> : <Upload className="size-4" />}
                        {logo ? "Logoni almashtirish" : "Logo biriktirish"}
                        <input
                          type="file"
                          accept="image/*"
                          className="hidden"
                          onChange={(e) => {
                            const file = e.target.files?.[0]
                            if (file) {
                              void uploadBrandLogo(brand, file)
                            }
                            e.currentTarget.value = ""
                          }}
                          disabled={busy}
                        />
                      </label>
                    </div>
                  )
                })}
                {brands.length === 0 ? (
                  <p className="text-xs text-muted-foreground">Brandlar topilmadi</p>
                ) : null}
              </div>
            </div>
          </Panel>
        </div>

        <Panel title="Maket" noPadding className="overflow-hidden">
          <LabelCanvas
            template={templatePreview}
            selectedIds={selectedIds}
            onSelectIds={setSelectedIds}
            onApplyDragFromOrigins={applyDragFromOrigins}
            onUpdateElement={updateElement}
            onOffsetAllElements={offsetAllElements}
            onScaleAllElements={scaleAllElements}
            onImageDrop={(file, x, y) => void handleCanvasImageDrop(file, x, y)}
            onElementDrop={addElementAt}
            sampleData={previewSampleData}
          />
        </Panel>

        <div className="flex flex-col gap-4">
          <Panel title="Xususiyatlar" className="h-fit">
            <ElementProperties
              element={selectedElement}
              selectedCount={selectedIds.length}
              canGroup={canGroup}
              canUngroup={canUngroup}
              onGroup={groupSelected}
              onUngroup={ungroupSelected}
              onChange={updateElement}
              onDelete={deleteSelected}
              labelWidthMm={widthMm}
              labelHeightMm={heightMm}
              onImageFile={selectedElement?.type === "image" ? (file) => void handleSelectedImageFile(file) : undefined}
            />
          </Panel>

          <Panel title="Printer ko'rinishi" className="h-fit">
            <LabelPrinterPreview
              templateId={templateId}
              widthMm={widthMm}
              heightMm={heightMm}
              dpi={dpi}
              printRotationDeg={printRotationDeg}
              definition={definition}
              previewSerial={previewSerial}
              previewAccSerial={previewAccSerial}
            />
          </Panel>
        </div>
      </div>

      <TableSizeDialog
        open={tableDropAt !== null}
        onOpenChange={(open) => {
          if (!open) {
            setTableDropAt(null)
          }
        }}
        onConfirm={confirmTableCreate}
      />
    </PageContainer>
  )
}
