import { useCallback, useEffect, useMemo, useState } from "react"
import { saveAs } from "file-saver"
import { buildConsumptionNormWorkbook } from "@/lib/consumption-norm-export"
import {
  ArrowLeft,
  ChevronDown,
  ChevronRight,
  ChevronsUpDown,
  Download,
  FileSpreadsheet,
  ListTree,
  Plus,
  Search,
  Trash2,
  Upload,
} from "lucide-react"
import { useDropzone } from "react-dropzone"
import { useNavigate, useParams } from "react-router-dom"
import {
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
} from "@tanstack/react-table"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { PageSearchInput } from "@/components/layout/page-search-input"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
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
import { flexRender } from "@tanstack/react-table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import { cn } from "@/lib/utils"

type ModelRow = {
  id: number
  model_nomi?: string
  modeli?: string
  qisqa_nomi?: string
  seriya_raqami?: string
  brend?: string
}

type NormItem = {
  id: number
  model_id: number
  sort_order: number
  group_level: number
  component_id: number
  factory_code?: string
  odoo_code?: string
  standard_name_uz?: string
  quantity: number
  consume_line_id?: number
  consume_line_name?: string
  receive_line_id?: number
  receive_line_name?: string
}

type UploadConflict = {
  row: number
  field?: string
  value?: string
  message?: string
}

type LineOption = {
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

const lineSelectClassName = cn(
  "h-9 min-w-[150px] max-w-[220px] rounded-lg border border-input bg-background px-2 text-sm shadow-sm",
  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
)

function normalizeSearchValue(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

function formatQuantity(value: number) {
  if (!Number.isFinite(value)) {
    return ""
  }
  return String(value)
}

type DisplayRow = {
  item: NormItem
  index: number
  level: number
}

function rowHasChildren(rows: DisplayRow[], index: number) {
  if (index >= rows.length - 1) {
    return false
  }
  return rows[index + 1].level > rows[index].level
}

function rowIsVisible(rows: DisplayRow[], index: number, collapsed: Set<number>) {
  let currentLevel = rows[index].level
  for (let parent = index - 1; parent >= 0; parent--) {
    const parentLevel = rows[parent].level
    if (parentLevel < currentLevel) {
      if (collapsed.has(parent)) {
        return false
      }
      currentLevel = parentLevel
    }
  }
  return true
}

function countDescendants(rows: DisplayRow[], index: number) {
  const level = rows[index].level
  let count = 0
  for (let child = index + 1; child < rows.length; child++) {
    if (rows[child].level <= level) {
      break
    }
    count += 1
  }
  return count
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

function collectParentIndexes(rows: DisplayRow[]) {
  const parents = new Set<number>()
  for (let index = 0; index < rows.length; index++) {
    if (rowHasChildren(rows, index)) {
      parents.add(index)
    }
  }
  return parents
}

function toDisplayRows(items: NormItem[]) {
  return items.map((item, index) => ({
    item,
    index,
    level: item.group_level,
  }))
}

const columnHelper = createColumnHelper<NormItem>()

export default function ConsumptionNormIdPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const modelId = Number(id)

  const [model, setModel] = useState<ModelRow | null>(null)
  const [items, setItems] = useState<NormItem[]>([])
  const [globalFilter, setGlobalFilter] = useState("")
  const [sorting, setSorting] = useState<SortingState>([])
  const [loading, setLoading] = useState(true)
  const [exporting, setExporting] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [uploadConflicts, setUploadConflicts] = useState<UploadConflict[]>([])
  const [conflictDialogOpen, setConflictDialogOpen] = useState(false)
  const [groupingEnabled, setGroupingEnabled] = useState(true)
  const [collapsedRows, setCollapsedRows] = useState<Set<number>>(new Set())
  const [lines, setLines] = useState<LineOption[]>([])
  const [allComponents, setAllComponents] = useState<ProductionComponent[]>([])
  const [savingLineItemId, setSavingLineItemId] = useState<number | null>(null)
  const [addParentId, setAddParentId] = useState(0)
  const [parentPickerOpen, setParentPickerOpen] = useState(false)
  const [parentPickerCollapsed, setParentPickerCollapsed] = useState<Set<number>>(new Set())
  const [addComponentSearch, setAddComponentSearch] = useState("")
  const [componentPickerOpen, setComponentPickerOpen] = useState(false)
  const [selectedAddComponent, setSelectedAddComponent] = useState<ProductionComponent | null>(null)
  const [addQuantity, setAddQuantity] = useState("")
  const [adding, setAdding] = useState(false)
  const [deletingItem, setDeletingItem] = useState<NormItem | null>(null)
  const [deleting, setDeleting] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    const [modelsResult, itemsResult, linesResult, componentsResult] = await Promise.all([
      Backend_Request<ModelRow[]>({}, "/api/tech/models/all"),
      Backend_Request<NormItem[]>({ model_id: modelId }, "/api/production/consumption-norm/items"),
      Backend_Request<LineOption[]>({}, "/api/production/consumption-norm/lines"),
      Backend_Request<ProductionComponent[]>({}, "/api/production/components/all"),
    ])
    setLoading(false)

    if (modelsResult.result === "ok") {
      const found = (modelsResult.data ?? []).find((item) => item.id === modelId)
      setModel(found ?? null)
    } else {
      ShowErrorToast(modelsResult.error || "Model yuklanmadi")
    }

    if (itemsResult.result === "ok") {
      const loadedItems = itemsResult.data ?? []
      setItems(loadedItems)
      setCollapsedRows(collectParentIndexes(toDisplayRows(loadedItems)))
    } else {
      ShowErrorToast(itemsResult.error || "Sarf normasi yuklanmadi")
    }

    if (linesResult.result === "ok") {
      setLines(linesResult.data ?? [])
    }

    if (componentsResult.result === "ok") {
      setAllComponents(componentsResult.data ?? [])
    }
  }, [modelId])

  useEffect(() => {
    if (!Number.isFinite(modelId) || modelId <= 0) {
      return
    }
    void loadData()
  }, [loadData, modelId])

  const columns = useMemo(
    () => [
      columnHelper.accessor("group_level", {
        header: "Guruh",
        cell: () => null,
      }),
      columnHelper.accessor("factory_code", {
        header: "Factory product code",
        cell: ({ row }) => row.original.factory_code || row.original.odoo_code || "",
      }),
      columnHelper.accessor("standard_name_uz", { header: "Komponent" }),
      columnHelper.accessor("quantity", {
        header: "Miqdori",
        cell: ({ getValue }) => formatQuantity(getValue()),
      }),
      columnHelper.accessor("consume_line_name", { header: "Ishlatilish joyi" }),
      columnHelper.accessor("receive_line_name", { header: "i/ch joyi" }),
      columnHelper.display({
        id: "actions",
        header: "",
        cell: () => null,
      }),
    ],
    [],
  )

  const table = useReactTable({
    data: items,
    columns,
    state: { sorting, globalFilter },
    onSortingChange: setSorting,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    globalFilterFn: (row, _columnId, filterValue) => {
      const query = normalizeSearchValue(filterValue)
      if (!query) {
        return true
      }
      const item = row.original
      return [
        item.group_level,
        item.factory_code,
        item.odoo_code,
        item.standard_name_uz,
        item.quantity,
        item.consume_line_name,
        item.receive_line_name,
      ]
        .map(normalizeSearchValue)
        .join(" ")
        .includes(query)
    },
  })

  const filteredTableRows = table.getFilteredRowModel().rows

  const displayRows = useMemo<DisplayRow[]>(
    () =>
      filteredTableRows.map((row, index) => ({
        item: row.original,
        index,
        level: row.original.group_level,
      })),
    [filteredTableRows],
  )

  const visibleRows = useMemo(() => {
    if (!groupingEnabled) {
      return displayRows
    }
    return displayRows.filter((_, index) => rowIsVisible(displayRows, index, collapsedRows))
  }, [displayRows, groupingEnabled, collapsedRows])

  const parentIndexes = useMemo(() => collectParentIndexes(displayRows), [displayRows])

  const allDisplayRows = useMemo(() => toDisplayRows(items), [items])

  const parentPickerParentIndexes = useMemo(
    () => collectParentIndexes(allDisplayRows),
    [allDisplayRows],
  )

  const visibleParentPickerRows = useMemo(
    () =>
      allDisplayRows.filter((_, index) =>
        rowIsVisible(allDisplayRows, index, parentPickerCollapsed),
      ),
    [allDisplayRows, parentPickerCollapsed],
  )

  const selectedParentLabel = useMemo(() => {
    if (addParentId === 0) {
      return "— Asosiy daraja (yangi guruh)"
    }
    const row = allDisplayRows.find((displayRow) => displayRow.item.id === addParentId)
    if (!row) {
      return "Ota qatorni tanlang"
    }
    const code = row.item.factory_code || row.item.odoo_code || "—"
    const name = row.item.standard_name_uz || ""
    return `${row.level} · ${code}${name ? ` — ${name}` : ""}`
  }, [addParentId, allDisplayRows])

  const filteredAddComponents = useMemo(() => {
    const query = normalizeSearchValue(addComponentSearch)
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
  }, [allComponents, addComponentSearch])

  function toggleGrouping() {
    setGroupingEnabled((current) => {
      if (current) {
        setCollapsedRows(new Set())
        return false
      }
      setCollapsedRows(collectParentIndexes(displayRows))
      return true
    })
  }

  function collapseAll() {
    setCollapsedRows(collectParentIndexes(displayRows))
  }

  function expandAll() {
    setCollapsedRows(new Set())
  }

  function toggleRow(index: number) {
    setCollapsedRows((current) => {
      const next = new Set(current)
      if (next.has(index)) {
        next.delete(index)
      } else {
        next.add(index)
      }
      return next
    })
  }

  function toggleParentPickerRow(index: number) {
    setParentPickerCollapsed((current) => {
      const next = new Set(current)
      if (next.has(index)) {
        next.delete(index)
      } else {
        next.add(index)
      }
      return next
    })
  }

  function handleParentPickerOpenChange(open: boolean) {
    setParentPickerOpen(open)
    if (open) {
      setParentPickerCollapsed(collectParentIndexes(allDisplayRows))
    }
  }

  function lineNameById(lineId?: number) {
    if (!lineId) {
      return ""
    }
    return lines.find((line) => line.line_id === lineId)?.name ?? ""
  }

  async function addItem() {
    if (!selectedAddComponent?.id) {
      ShowErrorToast("Komponentni tanlang")
      return
    }

    const quantity = Number(addQuantity.replace(",", "."))
    if (!Number.isFinite(quantity) || quantity <= 0) {
      ShowErrorToast("Miqdor 0 dan katta bo'lishi kerak")
      return
    }

    setAdding(true)
    const result = await Backend_Request(
      {
        model_id: modelId,
        component_id: selectedAddComponent.id,
        parent_id: addParentId,
        quantity,
      },
      "/api/production/consumption-norm/add",
    )
    setAdding(false)

    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Qo'shish xatolik")
      return
    }

    ShowOKToast("Komponent qo'shildi")
    const previousParentId = addParentId
    setAddParentId(0)
    setSelectedAddComponent(null)
    setAddComponentSearch("")
    setAddQuantity("")
    if (previousParentId > 0) {
      const parentIndex = allDisplayRows.findIndex((row) => row.item.id === previousParentId)
      if (parentIndex >= 0) {
        setCollapsedRows((current) => {
          const next = new Set(current)
          next.delete(parentIndex)
          return next
        })
      }
    }
    void loadData()
  }

  async function confirmDeleteItem() {
    if (!deletingItem) {
      return
    }

    setDeleting(true)
    const result = await Backend_Request(
      { id: deletingItem.id },
      "/api/production/consumption-norm/delete",
    )
    setDeleting(false)

    if (result.result !== "ok") {
      ShowErrorToast(result.error || "O'chirish xatolik")
      return
    }

    const data = result.data as { deleted?: number } | null
    ShowOKToast(`O'chirildi: ${data?.deleted ?? 1} ta qator`)
    setDeletingItem(null)
    void loadData()
  }

  async function updateItemLines(
    item: NormItem,
    field: "consume_line_id" | "receive_line_id",
    lineId: number,
  ) {
    const consumeLineId = field === "consume_line_id" ? lineId : (item.consume_line_id ?? 0)
    const receiveLineId = field === "receive_line_id" ? lineId : (item.receive_line_id ?? 0)

    if (
      consumeLineId === (item.consume_line_id ?? 0) &&
      receiveLineId === (item.receive_line_id ?? 0)
    ) {
      return
    }

    setSavingLineItemId(item.id)
    const result = await Backend_Request(
      {
        id: item.id,
        consume_line_id: consumeLineId,
        receive_line_id: receiveLineId,
      },
      "/api/production/consumption-norm/update-lines",
    )
    setSavingLineItemId(null)

    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Liniya yangilanmadi")
      return
    }

    setItems((current) =>
      current.map((row) =>
        row.id === item.id
          ? {
              ...row,
              consume_line_id: consumeLineId || undefined,
              consume_line_name: lineNameById(consumeLineId),
              receive_line_id: receiveLineId || undefined,
              receive_line_name: lineNameById(receiveLineId),
            }
          : row,
      ),
    )
  }

  const { getRootProps, getInputProps, isDragActive, open } = useDropzone({
    multiple: false,
    noClick: true,
    accept: {
      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": [".xlsx"],
    },
    onDrop: (files) => setSelectedFile(files[0] ?? null),
  })

  async function downloadTemplate() {
    try {
      const workbook = await buildConsumptionNormWorkbook(
        {
          id: model?.id || modelId,
          model_nomi: model?.qisqa_nomi,
          modeli: model?.modeli,
        },
        [],
        lines,
      )
      const buffer = await workbook.xlsx.writeBuffer()
      saveAs(
        new Blob([buffer], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
        `bom_list_template_${model?.modeli || modelId}.xlsx`,
      )
    } catch {
      ShowErrorToast("Template yuklanmadi")
    }
  }

  async function exportNorm() {
    const visibleItems = table.getFilteredRowModel().rows.map((row) => row.original)
    if (visibleItems.length === 0) {
      ShowErrorToast("Export uchun qatorlar yo'q")
      return
    }

    setExporting(true)
    try {
      const workbook = await buildConsumptionNormWorkbook(
        {
          id: model?.id || modelId,
          model_nomi: model?.qisqa_nomi,
          modeli: model?.modeli,
        },
        visibleItems,
        lines,
      )
      const buffer = await workbook.xlsx.writeBuffer()
      saveAs(
        new Blob([buffer], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
        `bom_list_${model?.modeli || modelId}.xlsx`,
      )
    } catch {
      ShowErrorToast("Export xatolik")
    } finally {
      setExporting(false)
    }
  }

  function extractConflicts(data: unknown) {
    const list = Array.isArray(data)
      ? data
      : data && typeof data === "object" && Array.isArray((data as { conflicts?: UploadConflict[] }).conflicts)
        ? (data as { conflicts: UploadConflict[] }).conflicts
        : []
    return (list as UploadConflict[]).filter(
      (item) => item && typeof item === "object" && (item.row > 0 || item.field || item.value || item.message),
    )
  }

  async function uploadXlsx() {
    if (!selectedFile) {
      return
    }

    setUploading(true)
    const reader = new FileReader()
    reader.readAsDataURL(selectedFile)
    reader.onload = async () => {
      const result = await Backend_Request(
        { file64: reader.result, model_id: modelId },
        "/api/production/consumption-norm/upload",
      )
      setUploading(false)

      if (result.result !== "ok") {
        const conflicts = extractConflicts(result.data)
        if (conflicts.length > 0) {
          setUploadConflicts(conflicts)
          setConflictDialogOpen(true)
        }
        ShowErrorToast(result.error || "XLSX import xatolik")
        return
      }

      const data = result.data as { imported?: number; conflicts?: UploadConflict[] } | null
      const conflicts = extractConflicts(data)
      if (conflicts.length > 0) {
        setUploadConflicts(conflicts)
        setConflictDialogOpen(true)
      }
      ShowOKToast(`Import qilindi: ${data?.imported ?? 0} qator`)
      setSelectedFile(null)
      void loadData()
    }
    reader.onerror = () => {
      setUploading(false)
      ShowErrorToast("Fayl o'qilmadi")
    }
  }

  const modelTitle = model?.modeli || model?.qisqa_nomi || "Model"

  return (
    <PageContainer
      title="Sarf normasi"
      description={modelTitle}
      fullWidth
      center={<PageSearchInput value={globalFilter} onChange={setGlobalFilter} />}
      actions={
        <Button variant="outline" className="h-10 rounded-xl" onClick={() => navigate("/production/consumption-norm")}>
          <ArrowLeft className="size-4" />
          Orqaga
        </Button>
      }
    >
      <Panel
        title={modelTitle}
        description={`${groupingEnabled ? visibleRows.length : displayRows.length} / ${items.length} ta pozitsiya`}
        noPadding
        action={
          <div
            {...getRootProps()}
            className={cn(
              "flex shrink-0 flex-nowrap items-center gap-2",
              isDragActive && "rounded-xl ring-2 ring-primary/40",
            )}
          >
            <input {...getInputProps()} />
            <Button
              type="button"
              variant={groupingEnabled ? "default" : "outline"}
              className="h-10 shrink-0 rounded-xl px-3 font-semibold shadow-sm"
              disabled={displayRows.length === 0}
              onClick={toggleGrouping}
            >
              <ListTree className="size-4" />
              Guruhlash
            </Button>
            {groupingEnabled && (
              <>
                <Button
                  type="button"
                  variant="outline"
                  className="h-10 shrink-0 rounded-xl px-3 font-semibold shadow-sm"
                  onClick={expandAll}
                >
                  Ochish
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  className="h-10 shrink-0 rounded-xl px-3 font-semibold shadow-sm"
                  onClick={collapseAll}
                >
                  Yopish
                </Button>
              </>
            )}
            <Button
              type="button"
              variant="outline"
              className="h-10 shrink-0 rounded-xl px-3 font-semibold shadow-sm"
              disabled={exporting || table.getFilteredRowModel().rows.length === 0}
              onClick={() => void exportNorm()}
            >
              <Download className="size-4" />
              {exporting ? "Export..." : "Export XLSX"}
            </Button>
            <Button
              type="button"
              variant="outline"
              className="h-10 shrink-0 rounded-xl px-3 font-semibold shadow-sm"
              onClick={() => void downloadTemplate()}
            >
              <Download className="size-4" />
              Template
            </Button>
            <Button
              type="button"
              variant="outline"
              className="h-10 shrink-0 rounded-xl px-3 font-semibold shadow-sm"
              disabled={uploading}
              onClick={() => open()}
            >
              <FileSpreadsheet className="size-4" />
              {selectedFile ? selectedFile.name.slice(0, 18) : "Import"}
            </Button>
            <Button
              type="button"
              onClick={() => void uploadXlsx()}
              disabled={!selectedFile || uploading}
              className="h-10 shrink-0 rounded-xl bg-emerald-600 px-4 font-semibold text-white shadow-sm hover:bg-emerald-700"
            >
              <Upload className="size-4" />
              {uploading ? "Yuklanmoqda..." : "Upload"}
            </Button>
          </div>
        }
      >
        <div className="overflow-x-auto rounded-b-2xl pb-3">
          <Table>
            <TableHeader className="sticky top-0 z-10 bg-muted/90 backdrop-blur">
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id} className="border-b border-border/60 bg-muted/40 hover:bg-muted/40">
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      onClick={header.column.getToggleSortingHandler()}
                      className="sticky top-0 z-10 cursor-pointer select-none bg-muted/90 px-4 py-3 text-sm font-semibold backdrop-blur"
                    >
                      {flexRender(header.column.columnDef.header, header.getContext())}
                      {{
                        asc: " ↑",
                        desc: " ↓",
                      }[header.column.getIsSorted() as string] ?? null}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {visibleRows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={columns.length} className="h-24 text-center text-muted-foreground">
                    {loading ? "Yuklanmoqda..." : "Sarf normasi hali yuklanmagan"}
                  </TableCell>
                </TableRow>
              ) : (
                visibleRows.map((displayRow, visibleIndex) => {
                  const { item, index, level } = displayRow
                  const hasChildren = groupingEnabled && parentIndexes.has(index)
                  const isCollapsed = collapsedRows.has(index)
                  const tableRow = table.getFilteredRowModel().rows[index]

                  return (
                    <TableRow
                      key={`${item.id}-${item.sort_order}`}
                      className={cn(
                        "border-border/40",
                        visibleIndex % 2 === 0 ? "bg-transparent" : "bg-muted/20",
                        level <= 1 && "bg-primary/5",
                      )}
                    >
                      <TableCell className="px-4 py-4 text-sm">
                        <div
                          className="flex items-center gap-1"
                          style={{ paddingLeft: `${level * 16}px` }}
                        >
                          {groupingEnabled ? (
                            hasChildren ? (
                              <button
                                type="button"
                                onClick={() => toggleRow(index)}
                                className="inline-flex size-6 shrink-0 items-center justify-center rounded-md border border-border/60 bg-background hover:bg-muted"
                              >
                                {isCollapsed ? (
                                  <ChevronRight className="size-3.5" />
                                ) : (
                                  <ChevronDown className="size-3.5" />
                                )}
                              </button>
                            ) : (
                              <span className="inline-block size-6 shrink-0" />
                            )
                          ) : null}
                          <span
                            className={cn(
                              "min-w-6 font-medium tabular-nums",
                              level <= 1 && "font-semibold",
                            )}
                          >
                            {level}
                          </span>
                        </div>
                      </TableCell>
                      {tableRow.getVisibleCells().slice(1).map((cell) => (
                        <TableCell key={cell.id} className="px-4 py-4 text-sm">
                          {cell.column.id === "consume_line_name" ? (
                            <select
                              value={item.consume_line_id ?? 0}
                              disabled={savingLineItemId === item.id}
                              onClick={(event) => event.stopPropagation()}
                              onChange={(event) =>
                                void updateItemLines(
                                  item,
                                  "consume_line_id",
                                  Number(event.target.value),
                                )
                              }
                              className={lineSelectClassName}
                            >
                              <option value={0}>—</option>
                              {lines.map((line) => (
                                <option key={line.line_id} value={line.line_id}>
                                  {line.name}
                                </option>
                              ))}
                            </select>
                          ) : cell.column.id === "receive_line_name" ? (
                            <select
                              value={item.receive_line_id ?? 0}
                              disabled={savingLineItemId === item.id}
                              onClick={(event) => event.stopPropagation()}
                              onChange={(event) =>
                                void updateItemLines(
                                  item,
                                  "receive_line_id",
                                  Number(event.target.value),
                                )
                              }
                              className={lineSelectClassName}
                            >
                              <option value={0}>—</option>
                              {lines.map((line) => (
                                <option key={line.line_id} value={line.line_id}>
                                  {line.name}
                                </option>
                              ))}
                            </select>
                          ) : cell.column.id === "actions" ? (
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              className="text-destructive hover:text-destructive"
                              onClick={() => setDeletingItem(item)}
                            >
                              <Trash2 className="size-4" />
                            </Button>
                          ) : (
                            flexRender(cell.column.columnDef.cell, cell.getContext())
                          )}
                        </TableCell>
                      ))}
                    </TableRow>
                  )
                })
              )}
            </TableBody>
          </Table>

          <div className="border-t border-border/60 bg-muted/20 px-4 py-4">
            <div className="mb-3 text-sm font-semibold text-foreground">Qo'lda qo'shish</div>
            <div className="flex flex-col gap-3 lg:flex-row lg:flex-nowrap lg:items-end">
              <div className="min-w-[220px] flex-1 space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">Daraxt / ota qator</label>
                <Popover open={parentPickerOpen} onOpenChange={handleParentPickerOpenChange}>
                  <PopoverTrigger asChild>
                    <Button
                      type="button"
                      variant="outline"
                      className="h-10 w-full justify-between gap-2 px-3 font-normal"
                    >
                      <span className="truncate text-left">{selectedParentLabel}</span>
                      <ChevronDown className="size-4 shrink-0 opacity-50" />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent
                    side="bottom"
                    align="start"
                    sideOffset={4}
                    className="w-[var(--radix-popover-trigger-width)] gap-0 p-1"
                  >
                    <button
                      type="button"
                      className={cn(
                        "flex w-full rounded-md px-3 py-2 text-left text-sm hover:bg-muted",
                        addParentId === 0 && "bg-primary/10 font-medium",
                      )}
                      onClick={() => {
                        setAddParentId(0)
                        setParentPickerOpen(false)
                      }}
                    >
                      — Asosiy daraja (yangi guruh)
                    </button>
                    <div className="max-h-56 overflow-y-auto">
                      {visibleParentPickerRows.length === 0 ? (
                        <div className="px-3 py-4 text-center text-sm text-muted-foreground">
                          Daraxt bo'sh
                        </div>
                      ) : (
                        visibleParentPickerRows.map((displayRow) => {
                          const { item, index, level } = displayRow
                          const hasChildren = parentPickerParentIndexes.has(index)
                          const isCollapsed = parentPickerCollapsed.has(index)
                          const code = item.factory_code || item.odoo_code || "—"
                          const name = item.standard_name_uz || ""

                          return (
                            <div
                              key={item.id}
                              className={cn(
                                "flex items-center gap-1 rounded-md pr-2 hover:bg-muted",
                                addParentId === item.id && "bg-primary/10",
                              )}
                              style={{ paddingLeft: `${8 + level * 14}px` }}
                            >
                              {hasChildren ? (
                                <button
                                  type="button"
                                  onClick={(event) => {
                                    event.stopPropagation()
                                    toggleParentPickerRow(index)
                                  }}
                                  className="inline-flex size-6 shrink-0 items-center justify-center rounded-md border border-border/60 bg-background hover:bg-muted"
                                >
                                  {isCollapsed ? (
                                    <ChevronRight className="size-3.5" />
                                  ) : (
                                    <ChevronDown className="size-3.5" />
                                  )}
                                </button>
                              ) : (
                                <span className="inline-block size-6 shrink-0" />
                              )}
                              <button
                                type="button"
                                className="min-w-0 flex-1 py-2 text-left text-sm"
                                onClick={() => {
                                  setAddParentId(item.id)
                                  setParentPickerOpen(false)
                                }}
                              >
                                <span className="font-medium tabular-nums">{level}</span>
                                <span className="mx-1 text-muted-foreground">·</span>
                                <span>{code}</span>
                                {name ? (
                                  <span className="text-muted-foreground">{` — ${name}`}</span>
                                ) : null}
                              </button>
                            </div>
                          )
                        })
                      )}
                    </div>
                  </PopoverContent>
                </Popover>
              </div>

              <div className="min-w-[260px] flex-[1.3] space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">Komponent</label>
                <Popover
                  open={componentPickerOpen}
                  onOpenChange={(open) => {
                    setComponentPickerOpen(open)
                    if (!open) {
                      setAddComponentSearch("")
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
                          value={addComponentSearch}
                          onChange={(event) => setAddComponentSearch(event.target.value)}
                          placeholder="Qidirish..."
                          className="h-9 pl-8"
                          autoFocus
                        />
                      </div>
                    </div>
                    <div className="max-h-56 overflow-y-auto p-1">
                      {filteredAddComponents.length === 0 ? (
                        <div className="px-3 py-6 text-center text-sm text-muted-foreground">
                          Komponent topilmadi
                        </div>
                      ) : (
                        filteredAddComponents.map((component) => (
                          <button
                            key={component.id}
                            type="button"
                            className={cn(
                              "block w-full rounded-md px-3 py-2 text-left text-sm hover:bg-muted",
                              selectedAddComponent?.id === component.id && "bg-primary/10",
                            )}
                            onClick={() => {
                              setSelectedAddComponent(component)
                              setAddComponentSearch("")
                              setComponentPickerOpen(false)
                            }}
                          >
                            {componentLabel(component)}
                          </button>
                        ))
                      )}
                    </div>
                  </PopoverContent>
                </Popover>
              </div>

              <div className="w-full space-y-1.5 sm:w-[140px] sm:shrink-0">
                <label className="text-xs font-medium text-muted-foreground">Miqdori</label>
                <Input
                  value={addQuantity}
                  onChange={(event) => setAddQuantity(event.target.value)}
                  placeholder="0.00"
                  inputMode="decimal"
                  className="h-10"
                />
              </div>

              <Button
                type="button"
                className="h-10 shrink-0 rounded-xl px-4"
                disabled={adding}
                onClick={() => void addItem()}
              >
                <Plus className="size-4" />
                {adding ? "Qo'shilmoqda..." : "Qo'shish"}
              </Button>
            </div>
          </div>
        </div>
      </Panel>

      <Dialog open={Boolean(deletingItem)} onOpenChange={(open) => !open && setDeletingItem(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Komponentni o'chirish</DialogTitle>
            <DialogDescription>
              {deletingItem
                ? `${deletingItem.factory_code || deletingItem.odoo_code || deletingItem.standard_name_uz} o'chirilsinmi?`
                : ""}
              {deletingItem && groupingEnabled && (() => {
                const rowIndex = allDisplayRows.findIndex((row) => row.item.id === deletingItem.id)
                const childCount = rowIndex >= 0 ? countDescendants(allDisplayRows, rowIndex) : 0
                return childCount > 0
                  ? ` Ichki ${childCount} ta qator ham o'chiriladi.`
                  : ""
              })()}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setDeletingItem(null)}>
              Bekor qilish
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={deleting}
              onClick={() => void confirmDeleteItem()}
            >
              {deleting ? "O'chirilmoqda..." : "O'chirish"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={conflictDialogOpen} onOpenChange={setConflictDialogOpen}>
        <DialogContent className="max-h-[92svh] overflow-y-auto sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>Import xatoliklari</DialogTitle>
            <DialogDescription>
              To'g'ri qatorlar import qilindi. Quyidagi qatorlarda xatolik bor.
            </DialogDescription>
          </DialogHeader>
          <div className="overflow-auto rounded-xl border">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-3 py-2 text-left">Qator</th>
                  <th className="px-3 py-2 text-left">Maydon</th>
                  <th className="px-3 py-2 text-left">Qiymat</th>
                  <th className="px-3 py-2 text-left">Xabar</th>
                </tr>
              </thead>
              <tbody>
                {uploadConflicts.map((conflict, index) => (
                  <tr key={`${conflict.row}-${index}`} className="border-t">
                    <td className="px-3 py-2">{conflict.row}</td>
                    <td className="px-3 py-2">{conflict.field}</td>
                    <td className="px-3 py-2">{conflict.value}</td>
                    <td className="px-3 py-2">{conflict.message}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </DialogContent>
      </Dialog>
    </PageContainer>
  )
}
