import { useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import { ChevronsUpDown, Plus, Search, Trash2 } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import { cn } from "@/lib/utils"
import { randomId } from "@/lib/random-id"
import { formatQty, normalizedQty } from "@/lib/ware-quantity"

type NormModel = {
  id: number
  modeli?: string
  qisqa_nomi?: string
  model_nomi?: string
  item_count?: number
  status?: boolean
}

type Line = {
  line_id: number
  name: string
}

type ProductionComponent = {
  id?: number
  factory_code?: string
  odoo_code?: string
  standard_name_uz?: string
  manufacturer_code?: string
}

type ApiRequestRow = {
  component_id: number
  component_name: string
  manufacturer_code?: string
  factory_code: string
  line_id: number
  line_name: string
  norm_quantity: number
  line_quantity: number
  ware_quantity: number
  needed_quantity: number
}

type WareStockRow = {
  component_id: number
  quantity: number
}

type WareStockShortage = {
  component_id: number
  manufacturer_code?: string
  component_name?: string
  line_name?: string
  required_quantity: number
  available_quantity: number
  missing_quantity: number
}

type WareInvalidQuantityItem = {
  component_id: number
  manufacturer_code?: string
  component_name?: string
  line_name?: string
  quantity: number
}

type UnselectedLineItem = {
  component_id: number
  manufacturer_code?: string
  component_name?: string
  factory_code?: string
}

const INVALID_QUANTITY_ERROR = "Detal soni 0 bo'lishi mumkin emas"
const UNSELECTED_LINE_ERROR = "Liniya tanlanmagan"

type BalanceItem = {
  component_id: number
  quantity: number
}

type RequestRow = ApiRequestRow & {
  localId: string
}

const filterCardClassName =
  "flex flex-col gap-2 rounded-2xl border border-border/60 bg-card/80 p-3 shadow-sm"

const lineSelectClassName = cn(
  "h-9 min-w-[140px] max-w-[200px] rounded-lg border border-input bg-background px-2 text-sm shadow-sm",
  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
)

function normalizeSearchValue(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

function collectInvalidQuantityRows(rows: RequestRow[]): WareInvalidQuantityItem[] {
  return rows
    .filter((row) => normalizedQty(row.needed_quantity) <= 0)
    .map((row) => ({
      component_id: row.component_id,
      manufacturer_code: row.manufacturer_code || row.factory_code,
      component_name: row.component_name,
      line_name: row.line_name,
      quantity: row.needed_quantity,
    }))
}

function collectUnselectedLineRows(rows: RequestRow[]): UnselectedLineItem[] {
  return rows
    .filter((row) => normalizedQty(row.needed_quantity) > 0 && row.line_id <= 0)
    .map((row) => ({
      component_id: row.component_id,
      manufacturer_code: row.manufacturer_code || row.factory_code,
      component_name: row.component_name,
      factory_code: row.factory_code,
    }))
}

function componentLabel(component: ProductionComponent) {
  const code =
    component.factory_code?.trim() ||
    component.odoo_code?.trim() ||
    component.manufacturer_code?.trim() ||
    ""
  const name = component.standard_name_uz?.trim() || ""
  if (code && name) {
    return `${code} — ${name}`
  }
  return code || name || `ID ${component.id}`
}

function toRequestRows(items: ApiRequestRow[]) {
  return items.map((item) => ({
    ...item,
    localId: `${item.component_id}-${item.line_id}-${randomId()}`,
  }))
}

export default function WareRequestPage() {
  const navigate = useNavigate()
  const [models, setModels] = useState<NormModel[]>([])
  const [lines, setLines] = useState<Line[]>([])
  const [allComponents, setAllComponents] = useState<ProductionComponent[]>([])

  const [modelPickerOpen, setModelPickerOpen] = useState(false)
  const [modelSearch, setModelSearch] = useState("")
  const [selectedModel, setSelectedModel] = useState<NormModel | null>(null)
  const [modelQuantity, setModelQuantity] = useState("1")
  const [building, setBuilding] = useState(false)

  const [rows, setRows] = useState<RequestRow[]>([])
  const [wareStockByComponentId, setWareStockByComponentId] = useState<Map<number, number>>(
    () => new Map(),
  )

  const [componentPickerOpen, setComponentPickerOpen] = useState(false)
  const [componentSearch, setComponentSearch] = useState("")
  const [selectedAddComponent, setSelectedAddComponent] = useState<ProductionComponent | null>(null)
  const [selectedAddLineId, setSelectedAddLineId] = useState(0)
  const [addNeededQty, setAddNeededQty] = useState("")
  const [adding, setAdding] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [shortageDialogOpen, setShortageDialogOpen] = useState(false)
  const [shortages, setShortages] = useState<WareStockShortage[]>([])
  const [invalidQuantityDialogOpen, setInvalidQuantityDialogOpen] = useState(false)
  const [invalidQuantities, setInvalidQuantities] = useState<WareInvalidQuantityItem[]>([])
  const [unselectedLineDialogOpen, setUnselectedLineDialogOpen] = useState(false)
  const [unselectedLines, setUnselectedLines] = useState<UnselectedLineItem[]>([])

  useEffect(() => {
    void loadMeta()
  }, [])

  function resetForm() {
    setModelPickerOpen(false)
    setModelSearch("")
    setSelectedModel(null)
    setModelQuantity("1")
    setRows([])
    setComponentPickerOpen(false)
    setComponentSearch("")
    setSelectedAddComponent(null)
    setSelectedAddLineId(0)
    setAddNeededQty("")
    setShortageDialogOpen(false)
    setShortages([])
    setInvalidQuantityDialogOpen(false)
    setInvalidQuantities([])
    setUnselectedLineDialogOpen(false)
    setUnselectedLines([])
  }

  async function loadMeta() {
    const [allModelsResult, normModelsResult, linesResult, componentsResult, stockResult] =
      await Promise.all([
        Backend_Request<NormModel[]>({}, "/api/tech/models/all"),
        Backend_Request<NormModel[]>({}, "/api/production/consumption-norm/models"),
        Backend_Request<Line[]>({}, "/api/lines/all"),
        Backend_Request<ProductionComponent[]>({}, "/api/production/components/all"),
        Backend_Request<WareStockRow[]>({}, "/api/ware/stock/all"),
      ])

    if (allModelsResult.result === "ok") {
      const normCountById = new Map(
        (normModelsResult.result === "ok" ? normModelsResult.data ?? [] : []).map(
          (model) => [model.id, model.item_count ?? 0] as const,
        ),
      )
      setModels(
        (allModelsResult.data ?? [])
          .filter((model) => model.status !== false)
          .map((model) => ({
            id: model.id,
            modeli: model.modeli,
            qisqa_nomi: model.qisqa_nomi,
            model_nomi: model.qisqa_nomi || model.model_nomi,
            item_count: normCountById.get(model.id) ?? 0,
          })),
      )
    } else {
      ShowErrorToast(allModelsResult.error || "Modellar yuklanmadi")
    }

    if (linesResult.result === "ok") {
      setLines(linesResult.data ?? [])
    }

    if (componentsResult.result === "ok") {
      setAllComponents(componentsResult.data ?? [])
    }

    if (stockResult.result === "ok") {
      setWareStockByComponentId(
        new Map(
          (stockResult.data ?? []).map((item) => [item.component_id, item.quantity] as const),
        ),
      )
    }
  }

  const filteredModels = useMemo(() => {
    const query = normalizeSearchValue(modelSearch)
    return models.filter((model) => {
      if (!query) {
        return true
      }
      return [model.modeli, model.qisqa_nomi, model.model_nomi, model.id]
        .map(normalizeSearchValue)
        .join(" ")
        .includes(query)
    })
  }, [models, modelSearch])

  const filteredAddComponents = useMemo(() => {
    const query = normalizeSearchValue(componentSearch)
    return allComponents
      .filter((component) => component.id)
      .filter((component) => {
        if (!query) {
          return true
        }
        return [
          component.factory_code,
          component.odoo_code,
          component.manufacturer_code,
          component.standard_name_uz,
        ]
          .map(normalizeSearchValue)
          .join(" ")
          .includes(query)
      })
  }, [allComponents, componentSearch])

  async function buildRequest() {
    if (!selectedModel?.id) {
      ShowErrorToast("Modelni tanlang")
      return
    }
    if ((selectedModel.item_count ?? 0) <= 0) {
      ShowErrorToast("Bu model uchun sarf normasi kiritilmagan")
      return
    }
    const quantity = Number(modelQuantity.replace(",", "."))
    if (!Number.isFinite(quantity) || quantity <= 0) {
      ShowErrorToast("Miqdor 0 dan katta bo'lishi kerak")
      return
    }

    setBuilding(true)
    const result = await Backend_Request<ApiRequestRow[]>(
      { model_id: selectedModel.id, quantity },
      "/api/ware/request/build",
    )
    setBuilding(false)

    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Ro'yxat shakllantirilmadi")
      return
    }

    const built = toRequestRows(result.data ?? [])
    setRows(built)
    ShowOKToast(`Ro'yxat: ${built.length} ta komponent`)
  }

  function updateNeededQuantity(localId: string, value: string) {
    const parsed = Number(value.replace(",", "."))
    setRows((current) =>
      current.map((row) =>
        row.localId === localId
          ? { ...row, needed_quantity: Number.isFinite(parsed) ? parsed : 0 }
          : row,
      ),
    )
  }

  function removeRow(localId: string) {
    setRows((current) => current.filter((row) => row.localId !== localId))
  }

  async function updateRowLine(localId: string, lineId: number) {
    const line = lines.find((item) => item.line_id === lineId) ?? null
    if (!lineId || !line) {
      setRows((current) =>
        current.map((row) =>
          row.localId === localId ? { ...row, line_id: 0, line_name: "", line_quantity: 0 } : row,
        ),
      )
      return
    }

    const row = rows.find((item) => item.localId === localId)
    if (!row) {
      return
    }

    const lineQty = await fetchLineQuantity(lineId, row.component_id)
    setRows((current) =>
      current.map((item) =>
        item.localId === localId
          ? { ...item, line_id: line.line_id, line_name: line.name, line_quantity: lineQty }
          : item,
      ),
    )
  }

  async function fetchLineQuantity(lineId: number, componentId: number) {
    if (!lineId || !componentId) {
      return 0
    }
    const result = await Backend_Request<BalanceItem[]>({ line_id: lineId }, "/api/lines/balance")
    if (result.result !== "ok") {
      return 0
    }
    return (result.data ?? []).find((item) => item.component_id === componentId)?.quantity ?? 0
  }

  async function addManualRow() {
    if (!selectedAddComponent?.id) {
      ShowErrorToast("Komponentni tanlang")
      return
    }
    const selectedAddLine = lines.find((line) => line.line_id === selectedAddLineId) ?? null
    if (!selectedAddLine?.line_id) {
      ShowErrorToast("Liniyani tanlang")
      return
    }

    const needed = Number(addNeededQty.replace(",", "."))
    if (!Number.isFinite(needed) || needed <= 0) {
      ShowErrorToast("Kerakli miqdor 0 dan katta bo'lishi kerak")
      return
    }

    const exists = rows.some(
      (row) =>
        row.component_id === selectedAddComponent.id && row.line_id === selectedAddLine.line_id,
    )
    if (exists) {
      ShowErrorToast("Bu komponent va liniya allaqachon ro'yxatda")
      return
    }

    setAdding(true)
    const lineQty = await fetchLineQuantity(selectedAddLine.line_id, selectedAddComponent.id)
    setAdding(false)

    const newRow: RequestRow = {
      localId: `${selectedAddComponent.id}-${selectedAddLine.line_id}-${randomId()}`,
      component_id: selectedAddComponent.id,
      component_name: selectedAddComponent.standard_name_uz || "",
      factory_code:
        selectedAddComponent.factory_code ||
        selectedAddComponent.odoo_code ||
        selectedAddComponent.manufacturer_code ||
        "",
      line_id: selectedAddLine.line_id,
      line_name: selectedAddLine.name,
      norm_quantity: 0,
      line_quantity: lineQty,
      ware_quantity: wareStockByComponentId.get(selectedAddComponent.id) ?? 0,
      needed_quantity: needed,
    }

    setRows((current) => [...current, newRow])
    setSelectedAddComponent(null)
    setSelectedAddLineId(0)
    setAddNeededQty("")
    setComponentSearch("")
  }

  async function confirmDeliveryNote() {
    const unselectedLineRows = collectUnselectedLineRows(rows)
    if (unselectedLineRows.length > 0) {
      setUnselectedLines(unselectedLineRows)
      setUnselectedLineDialogOpen(true)
      return
    }

    const invalidRows = collectInvalidQuantityRows(rows)
    if (invalidRows.length > 0) {
      setInvalidQuantities(invalidRows)
      setInvalidQuantityDialogOpen(true)
      return
    }

    const items = rows
      .filter((row) => normalizedQty(row.needed_quantity) > 0)
      .map((row) => ({
        component_id: row.component_id,
        line_id: row.line_id,
        line_name: row.line_name,
        component_name: row.component_name,
        manufacturer_code: row.manufacturer_code || row.factory_code,
        factory_code: row.factory_code,
        needed_quantity: row.needed_quantity,
      }))

    if (items.length === 0) {
      ShowErrorToast("Kerakli miqdorli komponentlar yo'q")
      return
    }

    setConfirming(true)
    const result = await Backend_Request<{ id: number; item_count: number }>(
      {
        model_id: selectedModel?.id ?? 0,
        model_name: selectedModel?.modeli || selectedModel?.qisqa_nomi || "",
        model_quantity: Number(modelQuantity.replace(",", ".")) || 0,
        items,
      },
      "/api/ware/request/confirm",
    )
    setConfirming(false)

    if (result.result !== "ok") {
      if (result.error === INVALID_QUANTITY_ERROR && Array.isArray(result.data)) {
        setInvalidQuantities(result.data as WareInvalidQuantityItem[])
        setInvalidQuantityDialogOpen(true)
        return
      }
      const maybeShortages = Array.isArray(result.data) ? (result.data as WareStockShortage[]) : []
      if (maybeShortages.length > 0 && maybeShortages[0]?.missing_quantity !== undefined) {
        setShortages(maybeShortages)
        setShortageDialogOpen(true)
        return
      }
      ShowErrorToast(result.error || "Nakladnoma yaratilmadi")
      return
    }
    if (!result.data) {
      ShowErrorToast("Nakladnoma yaratilmadi")
      return
    }

    ShowOKToast(`Nakladnoma #${result.data.id} yaratildi (${result.data.item_count} ta)`)
    resetForm()
    void loadMeta()
  }

  return (
    <PageContainer
      title="Ombor buyurtmasi"
      description="Sarf normasi va liniya balansi asosida komponentlar ro'yxati"
      scrollable
      actions={
        <Button
          type="button"
          variant="outline"
          className="h-10 rounded-xl"
          onClick={() => navigate("/ombor/buyurtma/tarix")}
        >
          Tarix
        </Button>
      }
    >
      <Panel>
        <div className="grid gap-3 lg:grid-cols-3">
          <div className={cn(filterCardClassName, "lg:col-span-2")}>
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-2">
                <div className="text-sm font-medium text-muted-foreground">Model</div>
                <Popover
                  open={modelPickerOpen}
                  onOpenChange={(open) => {
                    setModelPickerOpen(open)
                    if (!open) {
                      setModelSearch("")
                    }
                  }}
                >
                  <PopoverTrigger asChild>
                    <Button
                      type="button"
                      variant="outline"
                      className="h-11 w-full justify-between gap-2 px-3 font-normal"
                    >
                      <span className="truncate text-left">
                        {selectedModel
                          ? selectedModel.modeli || selectedModel.qisqa_nomi || `ID ${selectedModel.id}`
                          : "Modelni tanlang"}
                      </span>
                      <ChevronsUpDown className="size-4 shrink-0 opacity-50" />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent
                    side="bottom"
                    align="start"
                    sideOffset={4}
                    className="w-[var(--radix-popover-trigger-width)] gap-0 p-0"
                  >
                    <div className="border-b p-2">
                      <div className="relative">
                        <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
                        <Input
                          value={modelSearch}
                          onChange={(event) => setModelSearch(event.target.value)}
                          placeholder="Model qidirish..."
                          className="h-9 pl-8"
                          autoFocus
                        />
                      </div>
                    </div>
                    <div className="max-h-56 overflow-y-auto p-1">
                      {filteredModels.length === 0 ? (
                        <div className="px-3 py-6 text-center text-sm text-muted-foreground">
                          Model topilmadi
                        </div>
                      ) : (
                        filteredModels.map((model) => (
                          <button
                            key={model.id}
                            type="button"
                            className={cn(
                              "block w-full rounded-md px-3 py-2 text-left text-sm hover:bg-muted",
                              selectedModel?.id === model.id && "bg-primary/10",
                              (model.item_count ?? 0) <= 0 && "text-muted-foreground",
                            )}
                            onClick={() => {
                              setSelectedModel(model)
                              setModelSearch("")
                              setModelPickerOpen(false)
                            }}
                          >
                            {model.modeli || model.qisqa_nomi || `ID ${model.id}`}
                            <span className="ml-2 text-muted-foreground">
                              ({model.item_count ?? 0} poz.)
                            </span>
                          </button>
                        ))
                      )}
                    </div>
                  </PopoverContent>
                </Popover>
              </div>

              <div className="space-y-2">
                <div className="text-sm font-medium text-muted-foreground">Miqdori (dona)</div>
                <Input
                  value={modelQuantity}
                  onChange={(event) => setModelQuantity(event.target.value)}
                  placeholder="1"
                  inputMode="decimal"
                  className="h-11 rounded-xl"
                />
              </div>
            </div>
          </div>

          <div className={filterCardClassName}>
            <div className="h-5" />
            <Button
              type="button"
              className="h-11 rounded-xl"
              disabled={building}
              onClick={() => void buildRequest()}
            >
              {building ? "Hisoblanmoqda..." : "Hisoblash"}
            </Button>
          </div>
        </div>
      </Panel>

      <Panel
        title="Komponentlar ro'yxati"
        description={`${rows.length} ta detal`}
        noPadding
      >
        <div className="overflow-x-auto rounded-b-2xl pb-3">
          <Table>
            <TableHeader className="sticky top-0 z-10 bg-muted/90 backdrop-blur">
              <TableRow className="border-b border-border/60 bg-muted/40 hover:bg-muted/40">
                <TableHead className="px-4 py-3 text-sm font-semibold">Komponent</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Liniya</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Norma</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Liniyada</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Omborda</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Kerakli miqdor</TableHead>
                <TableHead className="w-12 px-4 py-3" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                    Model tanlang va Hisoblash tugmasini bosing
                  </TableCell>
                </TableRow>
              ) : (
                rows.map((row, index) => (
                  <TableRow
                    key={row.localId}
                    className={cn(
                      "border-border/40",
                      index % 2 === 0 ? "bg-transparent" : "bg-muted/20",
                    )}
                  >
                    <TableCell className="px-4 py-3 text-sm">
                      <div className="font-medium">
                        {row.component_name || row.factory_code || `ID ${row.component_id}`}
                      </div>
                      {row.factory_code ? (
                        <div className="text-xs text-muted-foreground">{row.factory_code}</div>
                      ) : null}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm">
                      <select
                        value={row.line_id || 0}
                        onClick={(event) => event.stopPropagation()}
                        onChange={(event) =>
                          void updateRowLine(row.localId, Number(event.target.value))
                        }
                        className={cn(
                          lineSelectClassName,
                          row.line_id <= 0 && "border-destructive text-destructive",
                        )}
                      >
                        <option value={0}>—</option>
                        {lines.map((line) => (
                          <option key={line.line_id} value={line.line_id}>
                            {line.name}
                          </option>
                        ))}
                      </select>
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums">
                      {formatQty(row.norm_quantity)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums">
                      {formatQty(row.line_quantity)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums">
                      {formatQty(row.ware_quantity)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm">
                      <Input
                        value={String(row.needed_quantity)}
                        onChange={(event) =>
                          updateNeededQuantity(row.localId, event.target.value)
                        }
                        inputMode="decimal"
                        className={cn(
                          "h-9 w-28 rounded-lg tabular-nums",
                          row.needed_quantity > row.ware_quantity &&
                            "border-destructive text-destructive focus-visible:ring-destructive/30",
                        )}
                      />
                    </TableCell>
                    <TableCell className="px-4 py-3">
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive"
                        onClick={() => removeRow(row.localId)}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        <div className="border-t border-border/60 bg-muted/20 px-4 py-4">
          <div className="grid gap-3 lg:grid-cols-[minmax(240px,1.4fr)_minmax(200px,1fr)_140px_auto] lg:items-end">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Komponent</label>
              <Popover
                open={componentPickerOpen}
                onOpenChange={(open) => {
                  setComponentPickerOpen(open)
                  if (!open) {
                    setComponentSearch("")
                  }
                }}
              >
                <PopoverTrigger asChild>
                  <Button
                    type="button"
                    variant="outline"
                    className="h-10 w-full justify-between gap-2 px-3 font-normal"
                  >
                    <span className="truncate text-left">
                      {selectedAddComponent
                        ? componentLabel(selectedAddComponent)
                        : "Komponentni tanlang"}
                    </span>
                    <ChevronsUpDown className="size-4 shrink-0 opacity-50" />
                  </Button>
                </PopoverTrigger>
                <PopoverContent
                  side="bottom"
                  align="start"
                  sideOffset={4}
                  className="w-[var(--radix-popover-trigger-width)] gap-0 p-0"
                >
                  <div className="border-b p-2">
                    <div className="relative">
                      <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
                      <Input
                        value={componentSearch}
                        onChange={(event) => setComponentSearch(event.target.value)}
                        placeholder="Qidirish..."
                        className="h-9 pl-8"
                        autoFocus
                      />
                    </div>
                  </div>
                  <div className="max-h-56 overflow-y-auto p-1">
                    {filteredAddComponents.map((component) => (
                      <button
                        key={component.id}
                        type="button"
                        className="block w-full rounded-md px-3 py-2 text-left text-sm hover:bg-muted"
                        onClick={() => {
                          setSelectedAddComponent(component)
                          setComponentSearch("")
                          setComponentPickerOpen(false)
                        }}
                      >
                        {componentLabel(component)}
                      </button>
                    ))}
                  </div>
                </PopoverContent>
              </Popover>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Liniya</label>
              <select
                value={selectedAddLineId}
                onChange={(event) => setSelectedAddLineId(Number(event.target.value))}
                className={cn(lineSelectClassName, "h-10 w-full max-w-none")}
              >
                <option value={0}>Liniyani tanlang</option>
                {lines.map((line) => (
                  <option key={line.line_id} value={line.line_id}>
                    {line.name}
                  </option>
                ))}
              </select>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Kerakli miqdor</label>
              <Input
                value={addNeededQty}
                onChange={(event) => setAddNeededQty(event.target.value)}
                placeholder="0.00"
                inputMode="decimal"
                className="h-10"
              />
            </div>

            <Button
              type="button"
              className="h-10 shrink-0 rounded-xl px-4"
              disabled={adding}
              onClick={() => void addManualRow()}
            >
              <Plus className="size-4" />
              {adding ? "..." : "Qo'shish"}
            </Button>
          </div>

          <div className="mt-4 flex justify-end border-t border-border/60 pt-4">
            <Button
              type="button"
              className="h-11 min-w-40 rounded-xl px-6"
              disabled={confirming || rows.length === 0}
              onClick={() => void confirmDeliveryNote()}
            >
              {confirming ? "Yaratilmoqda..." : "Tasdiqlash"}
            </Button>
          </div>
        </div>
      </Panel>

      <Dialog open={unselectedLineDialogOpen} onOpenChange={setUnselectedLineDialogOpen}>
        <DialogContent className="max-h-[85svh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{UNSELECTED_LINE_ERROR}</DialogTitle>
            <DialogDescription>
              Quyidagi komponentlar uchun liniya tanlanishi kerak. Har bir qatorda liniyani belgilang.
            </DialogDescription>
          </DialogHeader>

          <div className="mt-2 overflow-hidden rounded-xl border border-border/60">
            <Table>
              <TableHeader className="bg-muted/40">
                <TableRow className="border-border/60 hover:bg-transparent">
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Komponent</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold text-destructive">
                    Liniya
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {unselectedLines.map((item, index) => (
                  <TableRow key={`${item.component_id}-${index}`} className="border-border/40">
                    <TableCell className="px-4 py-3 text-sm">
                      <div className="font-medium">
                        {item.component_name?.trim() ||
                          item.manufacturer_code?.trim() ||
                          `ID ${item.component_id}`}
                      </div>
                      {item.factory_code ? (
                        <div className="text-xs text-muted-foreground">{item.factory_code}</div>
                      ) : null}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm text-destructive">Tanlanmagan</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          <DialogFooter className="mt-4">
            <Button type="button" className="rounded-xl" onClick={() => setUnselectedLineDialogOpen(false)}>
              Yopish
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={invalidQuantityDialogOpen} onOpenChange={setInvalidQuantityDialogOpen}>
        <DialogContent className="max-h-[85svh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{INVALID_QUANTITY_ERROR}</DialogTitle>
            <DialogDescription>
              Quyidagi komponentlar uchun detal soni 0 yoki undan kichik. Miqdorni to&apos;g&apos;rilang.
            </DialogDescription>
          </DialogHeader>

          <div className="mt-2 overflow-hidden rounded-xl border border-border/60">
            <Table>
              <TableHeader className="bg-muted/40">
                <TableRow className="border-border/60 hover:bg-transparent">
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Komponent</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Liniya</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold text-destructive">
                    Detal soni
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {invalidQuantities.map((item, index) => (
                  <TableRow key={`${item.component_id}-${item.line_name ?? index}`} className="border-border/40">
                    <TableCell className="px-4 py-3 text-sm">
                      <div className="font-medium">
                        {item.component_name?.trim() || item.manufacturer_code?.trim() || `ID ${item.component_id}`}
                      </div>
                      {item.manufacturer_code ? (
                        <div className="text-xs text-muted-foreground">{item.manufacturer_code}</div>
                      ) : null}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm">{item.line_name?.trim() || "—"}</TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums text-destructive">
                      {formatQty(item.quantity)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          <DialogFooter className="mt-4">
            <Button type="button" className="rounded-xl" onClick={() => setInvalidQuantityDialogOpen(false)}>
              Yopish
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={shortageDialogOpen} onOpenChange={setShortageDialogOpen}>
        <DialogContent className="max-h-[85svh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>Omborda komponent yetarli emas</DialogTitle>
            <DialogDescription>
              Quyidagi komponentlar bo'yicha qoldiq yetmaydi. Qoldiqni to'ldiring yoki kerakli miqdorni kamaytiring.
            </DialogDescription>
          </DialogHeader>

          <div className="mt-2 overflow-hidden rounded-xl border border-border/60">
            <Table>
              <TableHeader className="bg-muted/40">
                <TableRow className="border-border/60 hover:bg-transparent">
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Komponent</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Liniya</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Omborda</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold">Kerak</TableHead>
                  <TableHead className="px-4 py-2.5 text-sm font-semibold text-destructive">
                    Yetmaydi
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {shortages.map((item, index) => (
                  <TableRow key={`${item.component_id}-${item.line_name ?? index}`} className="border-border/40">
                    <TableCell className="px-4 py-3 text-sm">
                      <div className="font-medium">
                        {item.component_name?.trim() || item.manufacturer_code?.trim() || `ID ${item.component_id}`}
                      </div>
                      {item.manufacturer_code ? (
                        <div className="text-xs text-muted-foreground">{item.manufacturer_code}</div>
                      ) : null}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm">
                      {item.line_name?.trim() || "—"}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums">
                      {formatQty(item.available_quantity)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums">
                      {formatQty(item.required_quantity)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm tabular-nums text-destructive">
                      {formatQty(item.missing_quantity)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          <DialogFooter className="mt-4">
            <Button type="button" className="rounded-xl" onClick={() => setShortageDialogOpen(false)}>
              Yopish
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageContainer>
  )
}
