import { useDeferredValue, useEffect, useMemo, useState, type ClipboardEvent } from "react"
import { saveAs } from "file-saver"
import {
  Download,
  FileSpreadsheet,
  ImagePlus,
  Plus,
  Search,
  Trash2,
  Upload,
} from "lucide-react"
import { useDropzone } from "react-dropzone"
import {
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
} from "@tanstack/react-table"
import { PanelVirtualTable } from "@/components/layout/panel-virtual-table"
import { Backend_Request, Backend_Request_Blob } from "@/services/backend"
import { Global_Data } from "@/config/config"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
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

type IdName = {
  id: number
  name: string
}

type ProductionComponent = {
  id?: number
  detal_turi_kodi?: string
  factory_code?: string
  manufacturer_code: string
  full_name_uz: string
  standard_name_uz: string
  standard_name_ru: string
  specification_uz: string
  type?: string
  type_id: number
  unit?: string
  unit_id: number
  net_weight_pcs?: number | string
  net_weight_set?: number | string
  technological_waste_pcs?: number | string
  technological_waste_set?: number | string
  technological_waste?: number | string
  comment: string
  odoo_code: string
  net_weight_kg?: number | string
  photo_path?: string
}

type UploadConflict = {
  row: number
  field: string
  value: string
  message: string
  existing_id?: number
}

const emptyComponent: ProductionComponent = {
  detal_turi_kodi: "",
  factory_code: "",
  manufacturer_code: "",
  full_name_uz: "",
  standard_name_uz: "",
  standard_name_ru: "",
  specification_uz: "",
  type_id: 0,
  unit_id: 0,
  net_weight_pcs: 0,
  net_weight_set: 0,
  technological_waste_pcs: 0,
  technological_waste_set: 0,
  comment: "",
  odoo_code: "",
  photo_path: "",
}

const formFields = [
  { key: "detal_turi_kodi", label: "Detal Turi kodi" },
  { key: "odoo_code", label: "ODOO kod" },
  { key: "factory_code", label: "Mahsulotning korxona kodi", required: true },
  { key: "manufacturer_code", label: "Manufacturer code" },
  { key: "full_name_uz", label: "Mahsulotning to'liq nomi (O'zb)" },
  { key: "standard_name_uz", label: "Mahsulotning standart nomi (O'zb)" },
  { key: "standard_name_ru", label: "Mahsulotning standart nomi (Ru)" },
  { key: "specification_uz", label: "Xususiyatlari (O'zb)" },
  { key: "net_weight_pcs", label: "O'girligi (dona)", decimal: true },
  { key: "net_weight_set", label: "O'girligi (Set)", decimal: true },
  { key: "technological_waste_pcs", label: "Texnologik chiqit (dona)", decimal: true },
  { key: "technological_waste_set", label: "Texnologik chiqit (Set)", decimal: true },
] as const

const decimalFields: Array<keyof ProductionComponent> = [
  "net_weight_pcs",
  "net_weight_set",
  "technological_waste_pcs",
  "technological_waste_set",
]

const EXPORT_COLORS = {
  headerBg: "FF0891B2",
  headerText: "FFFFFFFF",
  idHeaderBg: "FF475569",
  idCellBg: "FFF1F5F9",
  subHeaderBg: "FFE0F2FE",
  weightHeaderBg: "FFF59E0B",
  weightHeaderText: "FF1F2937",
  weightCellBg: "FFFFFBEB",
  zebra: "FFF8FAFC",
  border: "FFCBD5E1",
} as const

const EXPORT_LOCKED_COLUMN = 1
const EXPORT_PHOTO_COLUMN = 12
const EXPORT_MAX_ROW = 2000
const EXPORT_PHOTO_SIZE = 54

const exportHeaderRow1 = [
  "ID",
  "Detal Turi kodi",
  "ODOO kod",
  "Mahsulotning korxona kodi",
  "Manufacturer code",
  "Mahsulotning to'liq nomi (O'zb)",
  "Mahsulotning standart nomi (O'zb)",
  "Mahsulotning standart nomi (Ru)",
  "O`lchov birligi",
  "Xarid turi (import, local yoki production)",
  "Xususiyatlari (O'zb)",
  "photo",
  "comment",
  "O'girligi (dona)",
  "O'girligi (Set)",
  "Texnologik chiqit (dona)",
  "Texnologik chiqit (Set)",
]

const exportHeaderRow2 = [
  "",
  "", "", "", "", "", "", "", "", "", "", "", "",
  "dona",
  "Set",
  "dona",
  "Set",
]

function applyExportBorder(cell: { border?: object }) {
  cell.border = {
    top: { style: "thin", color: { argb: EXPORT_COLORS.border } },
    left: { style: "thin", color: { argb: EXPORT_COLORS.border } },
    bottom: { style: "thin", color: { argb: EXPORT_COLORS.border } },
    right: { style: "thin", color: { argb: EXPORT_COLORS.border } },
  }
}

function componentExportRow(component: ProductionComponent) {
  const manufacturerCode =
    component.manufacturer_code?.trim() ||
    component.factory_code?.trim() ||
    ""

  return [
    component.id ?? "",
    component.detal_turi_kodi || "",
    component.odoo_code || "",
    component.factory_code || "",
    manufacturerCode,
    component.full_name_uz || "",
    component.standard_name_uz || "",
    component.standard_name_ru || "",
    component.unit || "",
    component.type || "",
    component.specification_uz || "",
    "",
    component.comment || "",
    component.net_weight_pcs ?? component.net_weight_kg ?? "",
    component.net_weight_set ?? "",
    component.technological_waste_pcs ?? component.technological_waste ?? "",
    component.technological_waste_set ?? "",
  ]
}

function componentPhotoUrl(path?: string) {
  if (!path) {
    return ""
  }
  if (path.startsWith("http")) {
    return path
  }
  return `${Global_Data.server_ip}${path}`
}

async function loadPhotoForExcel(photoPath?: string) {
  if (!photoPath?.trim()) {
    return null
  }

  try {
    const response = await fetch(componentPhotoUrl(photoPath), { credentials: "include" })
    if (!response.ok) {
      return null
    }
    const blob = await response.blob()
    const dataUrl = await new Promise<string>((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => resolve(String(reader.result || ""))
      reader.onerror = () => reject(reader.error)
      reader.readAsDataURL(blob)
    })
    const comma = dataUrl.indexOf(",")
    if (comma === -1) {
      return null
    }

    const meta = dataUrl.slice(0, comma).toLowerCase()
    const base64 = dataUrl.slice(comma + 1)
    let extension: "png" | "jpeg" | "gif" = "jpeg"
    if (meta.includes("png")) {
      extension = "png"
    } else if (meta.includes("gif")) {
      extension = "gif"
    }
    return { base64, extension }
  } catch {
    return null
  }
}

async function buildComponentsWorkbook(components: ProductionComponent[]) {
  const ExcelJS = await import("exceljs")
  const workbook = new ExcelJS.Workbook()
  const worksheet = workbook.addWorksheet("Base for Bom list", {
    views: [{ state: "frozen", ySplit: 2, xSplit: 1 }],
  })

  const headerRow1 = worksheet.addRow(exportHeaderRow1)
  const headerRow2 = worksheet.addRow(exportHeaderRow2)
  components.forEach((component) => worksheet.addRow(componentExportRow(component)))

  const columnCount = exportHeaderRow1.length
  // 1-based Excel columns: 14-15 net weight, 16-17 technological waste
  const weightGroups = [
    { start: 14, end: 15, title: "Net weight [kg]" },
    { start: 16, end: 17, title: "Technological waste" },
  ]

  weightGroups.forEach((group) => {
    worksheet.mergeCells(1, group.start, 1, group.end)
    const cell = worksheet.getCell(1, group.start)
    cell.value = group.title
    cell.alignment = { vertical: "middle", horizontal: "center", wrapText: true }
  })

  worksheet.columns.forEach((column, index) => {
    const header = exportHeaderRow1[index] ?? ""
    column.width = Math.max(12, Math.min(44, header.split("\n")[0]?.length + 5 || 14))
  })
  if (worksheet.getColumn(1)) {
    worksheet.getColumn(1).width = 10
  }
  if (worksheet.getColumn(EXPORT_PHOTO_COLUMN)) {
    worksheet.getColumn(EXPORT_PHOTO_COLUMN).width = 12
  }

  headerRow1.height = 28
  headerRow2.height = 22

  await Promise.all(
    components.map(async (component, index) => {
      const photo = await loadPhotoForExcel(component.photo_path)
      if (!photo) {
        return
      }

      const rowNumber = index + 3
      const row = worksheet.getRow(rowNumber)
      row.height = Math.max(row.height ?? 15, EXPORT_PHOTO_SIZE * 0.75)

      const imageId = workbook.addImage({
        base64: photo.base64,
        extension: photo.extension,
      })

      worksheet.addImage(imageId, {
        tl: { col: EXPORT_PHOTO_COLUMN - 1, row: rowNumber - 1 },
        ext: { width: EXPORT_PHOTO_SIZE, height: EXPORT_PHOTO_SIZE },
      })
    }),
  )

  worksheet.eachRow((row, rowNumber) => {
    for (let colNumber = 1; colNumber <= columnCount; colNumber++) {
      const cell = row.getCell(colNumber)
      const isIdColumn = colNumber === EXPORT_LOCKED_COLUMN
      const isWeightColumn = colNumber >= 14 && colNumber <= 17
      applyExportBorder(cell)

      if (rowNumber <= 2) {
        cell.font = {
          bold: true,
          color: {
            argb: isWeightColumn && rowNumber === 1
              ? EXPORT_COLORS.weightHeaderText
              : isIdColumn
                ? EXPORT_COLORS.headerText
                : EXPORT_COLORS.headerText,
          },
        }
        cell.alignment = { vertical: "middle", horizontal: "center", wrapText: true }
        cell.fill = {
          type: "pattern",
          pattern: "solid",
          fgColor: {
            argb: isIdColumn
              ? EXPORT_COLORS.idHeaderBg
              : isWeightColumn
                ? EXPORT_COLORS.weightHeaderBg
                : rowNumber === 2
                  ? EXPORT_COLORS.subHeaderBg
                  : EXPORT_COLORS.headerBg,
          },
        }
        continue
      }

      if (isIdColumn) {
        cell.fill = {
          type: "pattern",
          pattern: "solid",
          fgColor: { argb: EXPORT_COLORS.idCellBg },
        }
        cell.font = { bold: true, color: { argb: "FF334155" } }
        cell.alignment = { vertical: "middle", horizontal: "center" }
        continue
      }

      if (isWeightColumn) {
        cell.fill = {
          type: "pattern",
          pattern: "solid",
          fgColor: { argb: EXPORT_COLORS.weightCellBg },
        }
      } else if (rowNumber % 2 === 0) {
        cell.fill = {
          type: "pattern",
          pattern: "solid",
          fgColor: { argb: EXPORT_COLORS.zebra },
        }
      }

      cell.alignment = {
        vertical: "middle",
        horizontal: typeof cell.value === "number" ? "right" : "left",
        wrapText: colNumber >= 6 && colNumber <= 13,
      }
    }
  })

  for (let rowNumber = 1; rowNumber <= EXPORT_MAX_ROW; rowNumber++) {
    for (let colNumber = 1; colNumber <= columnCount; colNumber++) {
      worksheet.getCell(rowNumber, colNumber).protection = {
        locked: colNumber === EXPORT_LOCKED_COLUMN,
        hidden: false,
      }
    }
  }

  await worksheet.protect("premier", {
    selectLockedCells: true,
    selectUnlockedCells: true,
  })

  const excelBuffer = await workbook.xlsx.writeBuffer()
  return new Blob([excelBuffer], {
    type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  })
}

function downloadBlob(blob: Blob, filename: string) {
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

function normalizeSearchValue(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

function componentSearchText(component: ProductionComponent) {
  return [
    component.id,
    component.detal_turi_kodi,
    component.factory_code,
    component.odoo_code,
    component.manufacturer_code,
    component.full_name_uz,
    component.standard_name_uz,
    component.standard_name_ru,
    component.specification_uz,
    component.type,
    component.unit,
    component.comment,
  ]
    .map(normalizeSearchValue)
    .join(" ")
}

const columnHelper = createColumnHelper<ProductionComponent>()

const PRODUCTION_COMPONENTS_GRID_COLUMNS =
  "minmax(0, 1.1fr) minmax(0, 1fr) minmax(0, 1.3fr) minmax(0, 1.6fr) minmax(0, 0.9fr) minmax(0, 0.8fr)"

const COMPONENTS_TABLE_VIEWPORT_CLASS = "h-[min(70svh,calc(100svh-15rem))]"

export default function ProductionComponentsPage() {
  const [components, setComponents] = useState<ProductionComponent[]>([])
  const [types, setTypes] = useState<IdName[]>([])
  const [units, setUnits] = useState<IdName[]>([])
  const [globalFilter, setGlobalFilter] = useState("")
  const deferredFilter = useDeferredValue(globalFilter)
  const [sorting, setSorting] = useState<SortingState>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<ProductionComponent | null>(null)
  const [form, setForm] = useState<ProductionComponent>(emptyComponent)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [saving, setSaving] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [photoUploading, setPhotoUploading] = useState(false)
  const [photoPreview, setPhotoPreview] = useState("")
  const [photoFile64, setPhotoFile64] = useState("")
  const [uploadConflicts, setUploadConflicts] = useState<UploadConflict[]>([])
  const [conflictDialogOpen, setConflictDialogOpen] = useState(false)
  const [deletingComponent, setDeletingComponent] = useState(false)
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false)
  const [exporting, setExporting] = useState(false)

  function photoUrl(path?: string) {
    if (!path) {
      return ""
    }
    if (path.startsWith("http")) {
      return path
    }
    return `${Global_Data.server_ip}${path}`
  }

  async function loadComponents() {
    const result = await Backend_Request<ProductionComponent[]>({}, "/api/production/components/all")
    if (result.result === "ok") {
      setComponents(result.data ?? [])
    } else {
      ShowErrorToast(result.error || "Komponentlar yuklanmadi")
    }
  }

  async function loadDictionaries() {
    const [typesResult, unitsResult] = await Promise.all([
      Backend_Request<IdName[]>({}, "/api/production/components/types"),
      Backend_Request<IdName[]>({}, "/api/production/components/units"),
    ])
    if (typesResult.result === "ok") {
      setTypes(typesResult.data ?? [])
    }
    if (unitsResult.result === "ok") {
      setUnits(unitsResult.data ?? [])
    }
  }

  useEffect(() => {
    loadComponents()
    loadDictionaries()
  }, [])

  const searchIndex = useMemo(() => {
    const map = new Map<number | string, string>()
    for (const component of components) {
      const key = component.id ?? component.factory_code ?? component.odoo_code ?? map.size
      map.set(key, componentSearchText(component))
    }
    return map
  }, [components])

  const columns = useMemo(
    () => [
      columnHelper.accessor((row) => row.factory_code ?? "", {
        id: "factory_code",
        header: "Korxona kodi",
        cell: (info) => info.getValue() || "—",
      }),
      columnHelper.accessor((row) => row.odoo_code ?? "", {
        id: "odoo_code",
        header: "ODOO kod",
        cell: (info) => info.getValue() || "—",
      }),
      columnHelper.accessor((row) => row.manufacturer_code ?? "", {
        id: "manufacturer_code",
        header: "Manufacturer code",
        cell: (info) => info.getValue() || "—",
      }),
      columnHelper.accessor((row) => row.standard_name_uz ?? "", {
        id: "standard_name_uz",
        header: "Standard UZ",
        cell: (info) => info.getValue() || "—",
      }),
      columnHelper.accessor((row) => row.type ?? "", {
        id: "type",
        header: "Type",
        cell: (info) => info.getValue() || "—",
      }),
      columnHelper.accessor((row) => row.unit ?? "", {
        id: "unit",
        header: "Unit",
        cell: (info) => info.getValue() || "—",
      }),
    ],
    [],
  )

  const table = useReactTable({
    data: components,
    columns,
    state: { sorting, globalFilter: deferredFilter },
    onSortingChange: setSorting,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getRowId: (row, index) => String(row.id ?? `idx-${index}`),
    globalFilterFn: (row, _columnId, value) => {
      const tokens = normalizeSearchValue(value).split(/\s+/).filter(Boolean)
      if (tokens.length === 0) {
        return true
      }
      const key = row.original.id ?? row.original.factory_code ?? row.original.odoo_code ?? row.id
      const text = searchIndex.get(key) ?? componentSearchText(row.original)
      return tokens.every((token) => text.includes(token))
    },
  })

  const rows = table.getRowModel().rows
  const filteredCount = rows.length
  const totalCount = components.length

  function openAdd() {
    setEditing(null)
    setForm(emptyComponent)
    setPhotoPreview("")
    setPhotoFile64("")
    setDialogOpen(true)
  }

  function openEdit(component: ProductionComponent) {
    setEditing(component)
    setForm({
      ...emptyComponent,
      ...component,
      net_weight_pcs: component.net_weight_pcs ?? component.net_weight_kg ?? 0,
      technological_waste_pcs:
        component.technological_waste_pcs ?? component.technological_waste ?? 0,
    })
    setPhotoPreview(photoUrl(component.photo_path))
    setPhotoFile64("")
    setDialogOpen(true)
  }

  function updateForm(key: keyof ProductionComponent, value: string) {
    setForm((current) => ({
      ...current,
      [key]: ["type_id", "unit_id"].includes(key)
        ? Number(value)
        : decimalFields.includes(key)
          ? value.replace(",", ".")
          : value,
    }))
  }

  function normalizeDecimal(value: number | string) {
    if (typeof value === "number") {
      return value
    }
    const normalized = Number(value.replace(",", "."))
    return Number.isFinite(normalized) ? normalized : 0
  }

  async function saveComponent() {
    if (!form.factory_code?.trim()) {
      ShowErrorToast("Korxona kodi (Factory product code) kiritilishi shart")
      return
    }
    if (!form.type_id || !form.unit_id) {
      ShowErrorToast("Type va Unit tanlanishi shart")
      return
    }

    const payload: ProductionComponent = {
      ...form,
      net_weight_pcs: normalizeDecimal(form.net_weight_pcs),
      net_weight_set: normalizeDecimal(form.net_weight_set),
      technological_waste_pcs: normalizeDecimal(form.technological_waste_pcs),
      technological_waste_set: normalizeDecimal(form.technological_waste_set),
    }

    setSaving(true)
    const result = await Backend_Request(
      payload,
      editing ? "/api/production/components/update" : "/api/production/components/add",
    )
    setSaving(false)

    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Saqlanmadi")
      return
    }

    ShowOKToast(editing ? "Komponent yangilandi" : "Komponent qo'shildi")
    setDialogOpen(false)
    loadComponents()
  }

  function readPhotoFile(file: File) {
    if (!file.type.startsWith("image/")) {
      ShowErrorToast("Faqat rasm fayl tanlang")
      return
    }
    const reader = new FileReader()
    reader.readAsDataURL(file)
    reader.onload = () => {
      const value = String(reader.result || "")
      setPhotoFile64(value)
      setPhotoPreview(value)
    }
    reader.onerror = () => ShowErrorToast("Rasm o'qilmadi")
  }

  function handlePhotoPaste(event: ClipboardEvent<HTMLDivElement>) {
    const item = Array.from(event.clipboardData.items).find((clipboardItem) =>
      clipboardItem.type.startsWith("image/"),
    )
    const file = item?.getAsFile()
    if (file) {
      event.preventDefault()
      readPhotoFile(file)
    }
  }

  async function uploadPhoto() {
    if (!editing?.id) {
      ShowErrorToast("Avval komponentni saqlang")
      return
    }
    if (!photoFile64) {
      ShowErrorToast("Rasm tanlanmagan")
      return
    }

    setPhotoUploading(true)
    const result = await Backend_Request<string>(
      { component_id: editing.id, file64: photoFile64 },
      "/api/production/components/photo/upload",
    )
    setPhotoUploading(false)
    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Rasm yuklanmadi")
      return
    }
    const path = result.data || ""
    setPhotoFile64("")
    setPhotoPreview(photoUrl(path))
    setForm((current) => ({ ...current, photo_path: path }))
    setEditing((current) => (current ? { ...current, photo_path: path } : current))
    ShowOKToast("Rasm yangilandi")
    loadComponents()
  }

  const { getRootProps, getInputProps, isDragActive, open } = useDropzone({
    multiple: false,
    noClick: true,
    accept: {
      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": [".xlsx"],
    },
    onDrop: (files) => {
      setSelectedFile(files[0] ?? null)
    },
  })

  async function uploadXlsx() {
    if (!selectedFile) {
      return
    }

    setUploading(true)
    const reader = new FileReader()
    reader.readAsDataURL(selectedFile)
    reader.onload = async () => {
      const result = await Backend_Request(
        { file64: reader.result },
        "/api/production/components/upload",
      )
      setUploading(false)

      const extractConflicts = (data: unknown) => {
        const list = Array.isArray(data)
          ? data
          : data && typeof data === "object" && Array.isArray((data as { conflicts?: UploadConflict[] }).conflicts)
            ? (data as { conflicts: UploadConflict[] }).conflicts
            : []
        return (list as UploadConflict[]).filter(
          (item) =>
            item &&
            typeof item === "object" &&
            (item.row > 0 || item.field || item.value || item.message),
        )
      }

      if (result.result !== "ok") {
        const conflicts = extractConflicts(result.data)
        if (conflicts.length > 0) {
          setUploadConflicts(conflicts)
          setConflictDialogOpen(true)
        }
        ShowErrorToast(result.error || "XLSX import xatolik")
        return
      }

      if (result.data && typeof result.data === "object") {
        const data = result.data as {
          inserted?: number
          updated?: number
          conflicts?: UploadConflict[]
        }
        const conflicts = extractConflicts(data)
        if (conflicts.length > 0) {
          setUploadConflicts(conflicts)
          setConflictDialogOpen(true)
        }
        ShowOKToast(`Qo'shildi: ${data.inserted ?? 0}, yangilandi: ${data.updated ?? 0}`)
      } else {
        ShowOKToast(`Import qilindi: ${result.data}`)
      }
      setSelectedFile(null)
      loadComponents()
    }
    reader.onerror = () => {
      setUploading(false)
      ShowErrorToast("Fayl o'qilmadi")
    }
  }

  async function deleteComponent() {
    if (!editing?.id) {
      return
    }

    setDeletingComponent(true)
    const result = await Backend_Request<{ id?: number }>(
      { id: editing.id },
      "/api/production/components/delete",
    )
    setDeletingComponent(false)

    if (result.result !== "ok") {
      ShowErrorToast(result.error || "Komponent o'chirilmadi")
      return
    }

    setDeleteConfirmOpen(false)
    setDialogOpen(false)
    setEditing(null)
    setForm(emptyComponent)
    ShowOKToast("Komponent o'chirildi va kodlari yangilandi")
    loadComponents()
  }

  async function downloadTemplate() {
    const result = await Backend_Request_Blob({}, "/api/production/components/template")
    if (result.result !== "ok" || !result.data) {
      ShowErrorToast(result.error || "Template yuklanmadi")
      return
    }
    saveAs(result.data, "components_list_template.xlsx")
  }

  async function exportComponents() {
    const visibleComponents = rows.map((row) => row.original)
    if (visibleComponents.length === 0) {
      ShowErrorToast("Export uchun komponentlar yo'q")
      return
    }

    setExporting(true)
    try {
      const fileData = await buildComponentsWorkbook(visibleComponents)
      downloadBlob(fileData, "components_list.xlsx")
    } catch {
      ShowErrorToast("Export xatolik")
    } finally {
      setExporting(false)
    }
  }

  return (
    <PageContainer
      title="Production Components"
      description="Ishlab chiqarish komponentlari katalogi"
      actions={
        <div className="flex flex-wrap gap-2">
          <Button onClick={openAdd} className="rounded-xl">
            <Plus className="size-4" />
            Add component
          </Button>
        </div>
      }
    >
      <Panel
        title="Components table"
        description={`${filteredCount} / ${totalCount} ta komponent`}
        noPadding
        className="overflow-hidden"
        action={
          <div
            {...getRootProps()}
            className={cn(
              "flex flex-wrap items-center gap-2",
              isDragActive && "rounded-xl ring-2 ring-primary/40",
            )}
          >
            <input {...getInputProps()} />
            <Button
              onClick={() => void exportComponents()}
              variant="outline"
              className="h-10 rounded-xl"
              disabled={exporting || filteredCount === 0}
            >
              <Download className="size-4" />
              {exporting ? "Export..." : "Export XLSX"}
            </Button>
            <Button
              onClick={downloadTemplate}
              variant="outline"
              className="h-10 rounded-xl"
            >
              <Download className="size-4" />
              Template
            </Button>
            <Button
              type="button"
              variant="outline"
              className="h-10 rounded-xl"
              disabled={uploading}
              onClick={() => open()}
            >
              <FileSpreadsheet className="size-4" />
              {selectedFile ? selectedFile.name.slice(0, 18) : "Import"}
            </Button>
            <Button
              onClick={uploadXlsx}
              disabled={!selectedFile || uploading}
              className="h-10 rounded-xl bg-emerald-600 text-white hover:bg-emerald-700"
            >
              <Upload className="size-4" />
              {uploading ? "Yuklanmoqda..." : "Upload"}
            </Button>
            <div className="relative">
              <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={globalFilter}
                onChange={(event) => setGlobalFilter(event.target.value)}
                placeholder="Search..."
                className="h-10 w-72 rounded-xl pl-9"
              />
            </div>
          </div>
        }
      >
        <PanelVirtualTable
          table={table}
          gridTemplateColumns={PRODUCTION_COMPONENTS_GRID_COLUMNS}
          emptyMessage="Komponent topilmadi"
          rowHeight={45}
          overscan={12}
          viewportClassName={COMPONENTS_TABLE_VIEWPORT_CLASS}
          onRowClick={(row) => openEdit(row.original)}
          getRowClassName={(_row, index) => (index % 2 === 0 ? "bg-background" : "bg-muted/20")}
        />
      </Panel>

      <Dialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open)
          if (!open) {
            setDeleteConfirmOpen(false)
          }
        }}
      >
        <DialogContent className="max-h-[92svh] overflow-y-auto sm:max-w-5xl">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit component" : "Add component"}</DialogTitle>
            <DialogDescription>
              Component fields will be reused later in connected production tables.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-3 rounded-2xl border bg-muted/20 p-3 md:grid-cols-[220px_1fr]">
            <div
              tabIndex={0}
              onPaste={handlePhotoPaste}
              className="flex aspect-square items-center justify-center overflow-hidden rounded-2xl border border-dashed bg-background outline-none ring-primary/30 transition focus:ring-2"
            >
              {photoPreview ? (
                <img src={photoPreview} alt="Component" className="h-full w-full object-cover" />
              ) : (
                <div className="px-4 text-center text-sm text-muted-foreground">
                  <ImagePlus className="mx-auto mb-2 size-8" />
                  Rasm yo'q
                </div>
              )}
            </div>
            <div className="flex flex-col justify-center gap-3">
              <div>
                <div className="font-semibold">Component photo</div>
                <div className="text-sm text-muted-foreground">
                  Rasmni tanlang yoki preview joyiga bosib Ctrl+V orqali clipboarddan joylang.
                </div>
              </div>
              <div className="flex flex-wrap gap-2">
                <Input
                  type="file"
                  accept="image/png,image/jpeg,image/webp"
                  onChange={(event) => {
                    const file = event.target.files?.[0]
                    if (file) {
                      readPhotoFile(file)
                    }
                    event.target.value = ""
                  }}
                  className="max-w-sm rounded-xl"
                />
                <Button
                  type="button"
                  onClick={uploadPhoto}
                  disabled={!editing?.id || !photoFile64 || photoUploading}
                  className="rounded-xl"
                >
                  {photoUploading ? "Uploading..." : "Save photo"}
                </Button>
              </div>
              {!editing?.id ? (
                <div className="text-xs text-muted-foreground">
                  Yangi komponent uchun avval asosiy ma'lumotlarni saqlang, keyin rasm yuklang.
                </div>
              ) : null}
            </div>
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            {formFields.map((field) => (
              <div key={field.key} className="space-y-1.5">
                <Label>
                  {field.label}
                  {"required" in field && field.required ? <span className="text-destructive"> *</span> : null}
                </Label>
                <Input
                  type="text"
                  inputMode={"decimal" in field && field.decimal ? "decimal" : undefined}
                  value={String(form[field.key] ?? "")}
                  onChange={(event) => updateForm(field.key, event.target.value)}
                  className="rounded-xl"
                />
              </div>
            ))}

            <div className="space-y-1.5">
              <Label>Type (import, local, production)</Label>
              <select
                value={form.type_id || ""}
                onChange={(event) => updateForm("type_id", event.target.value)}
                className="h-10 w-full rounded-xl border border-input bg-background px-3 text-sm"
              >
                <option value="">Select type</option>
                {types.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </div>

            <div className="space-y-1.5">
              <Label>Unit of measurement</Label>
              <select
                value={form.unit_id || ""}
                onChange={(event) => updateForm("unit_id", event.target.value)}
                className="h-10 w-full rounded-xl border border-input bg-background px-3 text-sm"
              >
                <option value="">Select unit</option>
                {units.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </div>

            <div className="space-y-1.5 md:col-span-2">
              <Label>Comment</Label>
              <textarea
                value={form.comment}
                onChange={(event) => updateForm("comment", event.target.value)}
                className="min-h-24 w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
              />
            </div>
          </div>

          <DialogFooter className="flex flex-row items-center justify-between gap-2 sm:justify-between">
            {editing?.id ? (
              <Button
                variant="outline"
                onClick={() => setDeleteConfirmOpen(true)}
                disabled={deletingComponent || saving}
                className="border-red-300 bg-red-50 text-red-700 hover:bg-red-100 hover:text-red-800"
              >
                <Trash2 className="size-4" />
                Delete
              </Button>
            ) : (
              <div />
            )}
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => setDialogOpen(false)}>
                Cancel
              </Button>
              <Button onClick={saveComponent} disabled={saving || deletingComponent}>
                {saving ? "Saving..." : "Save"}
              </Button>
            </div>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteConfirmOpen} onOpenChange={setDeleteConfirmOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Komponentni o'chirish</DialogTitle>
            <DialogDescription>
              {editing?.factory_code || editing?.standard_name_uz || `ID ${editing?.id}`} komponenti o'chirishni tasdiqlaysizmi?
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:justify-end">
            <Button variant="outline" onClick={() => setDeleteConfirmOpen(false)} disabled={deletingComponent}>
              Bekor qilish
            </Button>
            <Button
              onClick={() => void deleteComponent()}
              disabled={deletingComponent}
              className="bg-red-600 text-white hover:bg-red-700"
            >
              {deletingComponent ? "O'chirilmoqda..." : "Ha, o'chirish"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={conflictDialogOpen} onOpenChange={setConflictDialogOpen}>
        <DialogContent className="max-h-[92svh] overflow-y-auto sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle>Import conflicts</DialogTitle>
            <DialogDescription>
              To'g'ri qatorlar import qilindi. Quyidagi qatorlarda xatolik bor — ularni XLSX faylda to'g'rilab qayta import qiling.
            </DialogDescription>
          </DialogHeader>

          <div className="overflow-auto rounded-xl border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-20">Row</TableHead>
                  <TableHead>Field</TableHead>
                  <TableHead>Value</TableHead>
                  <TableHead>Reason</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {uploadConflicts.map((conflict, index) => (
                  <TableRow key={`${conflict.row}-${conflict.field}-${conflict.value}-${index}`}>
                    <TableCell className="font-medium">{conflict.row}</TableCell>
                    <TableCell>{conflict.field}</TableCell>
                    <TableCell className="font-mono text-xs">{conflict.value}</TableCell>
                    <TableCell>{conflict.message}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          <DialogFooter>
            <Button onClick={() => setConflictDialogOpen(false)}>OK</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageContainer>
  )
}
