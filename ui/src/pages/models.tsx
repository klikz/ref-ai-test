import { Backend_Request } from "@/services/backend"
import { useCallback, useDeferredValue, useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import { useDropzone } from "react-dropzone"
import { Button } from "@/components/ui/button"
import { PageContainer } from "@/components/layout/page-container"
import { PageSearchInput } from "@/components/layout/page-search-input"
import { Panel } from "@/components/layout/panel"
import { PanelVirtualTable } from "@/components/layout/panel-virtual-table"
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
import { toast } from "sonner"
import { cn } from "@/lib/utils"
import { Download, FileSpreadsheet, Upload } from "lucide-react"
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  createColumnHelper,
  type SortingState,
  getFilteredRowModel,
} from "@tanstack/react-table"
import { modelFieldLabel } from "@/lib/model-field-labels"

type ModelUploadConflict = {
  row: number
  field: string
  value: string
  existing_id: number
  message: string
}

type ModelRow = {
  id: number
  qisqa_nomi?: string
  modeli?: string
  seriya_raqami?: string
  sovutgich_turi?: string
  umumiy_hajmi_l?: string
  rangi?: string
  odoo_code?: string
  brend?: string
  door_code?: string
  acc_serial?: string
  compressor_serial?: string
  gs1_ean13?: string
  [key: string]: string | number | boolean | null | undefined
}

const MODELS_GRID_COLUMNS =
  "minmax(0, 1.2fr) minmax(0, 1.3fr) minmax(0, 1fr) minmax(0, 1fr) minmax(0, 0.8fr) minmax(0, 0.7fr)"

const MODELS_TABLE_VIEWPORT_CLASS = "h-[min(70svh,calc(100svh-15rem))]"

const columnHelper = createColumnHelper<ModelRow>()

const modelExportFields = [
  "id",
  "seriya_raqami",
  "acc_serial",
  "modeli",
  "sovutgich_turi",
  "qisqa_nomi",
  "rangi",
  "sotuv_turi",
  "gs1_ean13",
  "gost",
  "taminot_kuchlanishi_v",
  "xladagent_miqdori_g",
  "energiya_samaradorlik_sarfi",
  "kompressor_nomi",
  "maxalliy_sertifikat",
  "eac_sertifikati",
  "ce_sertifikat",
  "ishlab_chiqaruvchi_mamlakat",
  "korxon_nomi",
  "manzil",
  "brend",
  "local_export",
  "netto",
  "brutto",
  "qadoq_hajmi",
  "mahsulot_hajmi",
  "iqlim_sharoitlari",
  "elektr_toki_kuchlanishi_va_turi",
  "yoritgich_lampaning_quvvati_vt",
  "umumiy_hajmi_l",
  "sovutgich_kamera_hajmi_l",
  "muzlatgich_kamera_hajmi_l",
  "muzlatish_quvvati",
  "nominal_tok_quvvati_w",
  "freon",
  "shovqin_darajasi_db",
  "odoo_code",
  "door_code",
  "compressor_serial",
  "comment",
]

const accSerialExportColumn = modelExportFields.indexOf("acc_serial") + 1
const exportLabelRow = 1
const exportKeyRow = 2

function exportValue(value: unknown) {
  return value === null || value === undefined ? " " : value
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

function modelSearchText(model: ModelRow) {
  return [
    model.id,
    model.qisqa_nomi,
    model.modeli,
    model.seriya_raqami,
    model.sovutgich_turi,
    model.rangi,
    model.odoo_code,
    model.brend,
    model.door_code,
    model.acc_serial,
    model.compressor_serial,
    model.gs1_ean13,
    model.umumiy_hajmi_l,
  ]
    .map(normalizeSearchValue)
    .join(" ")
}

export default function ModelsPage() {
  const navigate = useNavigate()

  const [models, setModels] = useState<ModelRow[]>([])
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [uploadConflicts, setUploadConflicts] = useState<ModelUploadConflict[]>([])
  const [conflictDialogOpen, setConflictDialogOpen] = useState(false)
  const [globalFilter, setGlobalFilter] = useState("")
  const deferredFilter = useDeferredValue(globalFilter)
  const [sorting, setSorting] = useState<SortingState>([])

  const onDrop = useCallback((files: File[]) => {
    if (files.length > 0) setSelectedFile(files[0])
  }, [])

  const { getRootProps, getInputProps, isDragActive, open } = useDropzone({
    onDrop,
    multiple: false,
    noClick: true,
  })

  function showOkToast(text: string) {
    toast(text, {
      style: {
        backgroundColor: "rgba(8, 113, 8, 0.5)",
        color: "white",
      },
      position: "top-right",
    })
  }

  function showErrorToast(text: string) {
    toast(text, {
      style: {
        backgroundColor: "rgba(31, 41, 55, 0.9)",
        color: "white",
      },
      position: "top-center",
    })
  }

  async function modelsGetAll() {
    const result = await Backend_Request<ModelRow[]>({}, "/api/tech/models/all")

    if (result.result === "ok" && Array.isArray(result.data)) {
      setModels(result.data)
    } else {
      showErrorToast(result.error || "Modellar yuklanmadi")
    }
  }

  async function handleSend() {
    if (!selectedFile) return

    setUploading(true)
    const reader = new FileReader()
    reader.readAsDataURL(selectedFile)

    reader.onload = async () => {
      const result = await Backend_Request(
        { file64: reader.result },
        "/api/tech/models/add",
      )

      if (result.result === "ok") {
        const data = result.data as { inserted?: number; updated?: number } | undefined
        showOkToast(
          data && typeof data === "object"
            ? `Qo'shildi: ${data.inserted ?? 0}, yangilandi: ${data.updated ?? 0}`
            : "Ma'lumot yangilandi",
        )
        modelsGetAll()
      } else {
        if (Array.isArray(result.data)) {
          setUploadConflicts(result.data as ModelUploadConflict[])
          setConflictDialogOpen(true)
        }
        showErrorToast(result.error)
      }
      setUploading(false)
    }
    reader.onerror = () => {
      setUploading(false)
      showErrorToast("Fayl o'qilmadi")
    }

    setSelectedFile(null)
  }

  const searchIndex = useMemo(() => {
    const map = new Map<number | string, string>()
    for (const model of models) {
      map.set(model.id ?? map.size, modelSearchText(model))
    }
    return map
  }, [models])

  const columns = useMemo(
    () => [
      columnHelper.accessor((row) => String(row.qisqa_nomi ?? ""), {
        id: "qisqa_nomi",
        header: "Qisqa nomi",
        cell: (info) => info.getValue() || "—",
      }),
      columnHelper.accessor((row) => String(row.modeli ?? ""), {
        id: "modeli",
        header: "Modeli",
        cell: (info) => info.getValue() || "—",
      }),
      columnHelper.accessor((row) => String(row.seriya_raqami ?? ""), {
        id: "seriya_raqami",
        header: "Seriya raqami",
        cell: (info) => info.getValue() || "—",
      }),
      columnHelper.accessor((row) => String(row.sovutgich_turi ?? ""), {
        id: "sovutgich_turi",
        header: "Sovutgich turi",
        cell: (info) => info.getValue() || "—",
      }),
      columnHelper.accessor((row) => String(row.umumiy_hajmi_l ?? ""), {
        id: "umumiy_hajmi_l",
        header: "Hajmi (L)",
        cell: (info) => info.getValue() || "—",
      }),
      columnHelper.accessor((row) => String(row.rangi ?? ""), {
        id: "rangi",
        header: "Rangi",
        cell: (info) => info.getValue() || "—",
      }),
    ],
    [],
  )

  const table = useReactTable({
    data: models,
    columns,
    state: {
      sorting,
      globalFilter: deferredFilter,
    },
    onSortingChange: setSorting,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getRowId: (row, index) => String(row.id ?? `idx-${index}`),
    globalFilterFn: (row, _id, value) => {
      const tokens = normalizeSearchValue(value).split(/\s+/).filter(Boolean)
      if (tokens.length === 0) {
        return true
      }
      const text = searchIndex.get(row.original.id) ?? modelSearchText(row.original)
      return tokens.every((token) => text.includes(token))
    },
  })

  const rows = table.getRowModel().rows
  const filteredCount = rows.length
  const totalCount = models.length

  useEffect(() => {
    modelsGetAll()
  }, [])

  async function buildModelsWorkbook(dataRows: unknown[][]) {
    const ExcelJS = await import("exceljs")
    const workbook = new ExcelJS.Workbook()
    const worksheet = workbook.addWorksheet("Models")
    worksheet.addRow(modelExportFields.map((field) => modelFieldLabel(field)))
    worksheet.addRow(modelExportFields)
    dataRows.forEach((row) => worksheet.addRow(row))

    const lockedColumn = 1
    const columnCount = modelExportFields.length
    const exportMaxRow = 1000

    worksheet.columns.forEach((column, index) => {
      const field = modelExportFields[index] ?? ""
      const label = modelFieldLabel(field)
      column.width = Math.max(14, label.length + 2)
    })

    if (accSerialExportColumn > 0) {
      const accSerialHeader = worksheet.getCell(exportLabelRow, accSerialExportColumn)
      accSerialHeader.note =
        "Majburiy maydon. Bir nechta prefix: har biri yangi qatorga, vergul yoki ; bilan."
    }

    worksheet.eachRow((row, rowNumber) => {
      for (let colNumber = 1; colNumber <= columnCount; colNumber++) {
        const cell = row.getCell(colNumber)
        const isIDColumn = colNumber === lockedColumn
        const isAccSerialColumn = colNumber === accSerialExportColumn
        if (rowNumber === exportLabelRow) {
          cell.font = { bold: true }
          cell.fill = {
            type: "pattern",
            pattern: "solid",
            fgColor: {
              argb: isIDColumn
                ? "FFE5E7EB"
                : isAccSerialColumn
                  ? "FFFFF7ED"
                  : "FFEEF2FF",
            },
          }
        } else if (rowNumber === exportKeyRow) {
          cell.font = { italic: true, color: { argb: "FF6B7280" }, size: 9 }
          cell.fill = {
            type: "pattern",
            pattern: "solid",
            fgColor: { argb: "FFF9FAFB" },
          }
        } else if (isIDColumn) {
          cell.fill = {
            type: "pattern",
            pattern: "solid",
            fgColor: { argb: "FFF3F4F6" },
          }
        }
      }
    })

    for (let rowNumber = 1; rowNumber <= exportMaxRow; rowNumber++) {
      for (let colNumber = 1; colNumber <= columnCount; colNumber++) {
        worksheet.getCell(rowNumber, colNumber).protection = {
          locked:
            colNumber === lockedColumn ||
            rowNumber === exportLabelRow ||
            rowNumber === exportKeyRow,
          hidden: false,
        }
      }
    }

    await worksheet.protect("premier", {
      selectLockedCells: true,
      selectUnlockedCells: true,
    })

    const buffer = await workbook.xlsx.writeBuffer()
    return new Blob([buffer], {
      type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    })
  }

  async function exportVisibleModels() {
    const visibleModels = rows.map((row) => row.original)
    if (visibleModels.length === 0) {
      showErrorToast("Export uchun model yo'q")
      return
    }

    const fileData = await buildModelsWorkbook(
      visibleModels.map((model) => modelExportFields.map((field) => exportValue(model[field]))),
    )
    downloadBlob(fileData, "models_visible.xlsx")
  }

  async function downloadEmptyTemplate() {
    const fileData = await buildModelsWorkbook([])
    downloadBlob(fileData, "models_template.xlsx")
  }

  return (
    <PageContainer
      title="Modellar"
      description={`${filteredCount} / ${totalCount} ta model`}
      fullWidth
      center={<PageSearchInput value={globalFilter} onChange={setGlobalFilter} />}
      actions={
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
            onClick={() => void exportVisibleModels()}
            variant="outline"
            className="h-10 shrink-0 rounded-xl border-primary/30 bg-primary/5 px-3 font-semibold text-primary shadow-sm hover:bg-primary/10"
            disabled={filteredCount === 0}
          >
            <Download className="size-4" />
            Export XLSX
          </Button>
          <Button
            type="button"
            onClick={() => void downloadEmptyTemplate()}
            variant="outline"
            className="h-10 shrink-0 rounded-xl px-3 font-semibold shadow-sm"
          >
            <Download className="size-4" />
            Template
          </Button>
          <Button
            type="button"
            variant="outline"
            className="ml-[30px] h-10 shrink-0 rounded-xl px-3 font-semibold shadow-sm"
            disabled={uploading}
            onClick={() => open()}
          >
            <FileSpreadsheet className="size-4" />
            {selectedFile ? selectedFile.name.slice(0, 18) : "Import"}
          </Button>
          <Button
            type="button"
            onClick={() => void handleSend()}
            disabled={!selectedFile || uploading}
            className="h-10 shrink-0 rounded-xl bg-emerald-600 px-4 font-semibold text-white shadow-sm hover:bg-emerald-700"
          >
            <Upload className="size-4" />
            {uploading ? "Yuklanmoqda..." : "Upload / Update"}
          </Button>
        </div>
      }
    >
      <Panel noPadding className="overflow-hidden">
        <PanelVirtualTable
          table={table}
          gridTemplateColumns={MODELS_GRID_COLUMNS}
          emptyMessage="Model topilmadi"
          rowHeight={45}
          overscan={12}
          viewportClassName={MODELS_TABLE_VIEWPORT_CLASS}
          onRowClick={(row) => navigate(`/models/${row.original.id}`)}
          getRowClassName={(_row, index) => (index % 2 === 0 ? "bg-background" : "bg-muted/20")}
        />
      </Panel>

      <Dialog open={conflictDialogOpen} onOpenChange={setConflictDialogOpen}>
        <DialogContent className="max-h-[92svh] overflow-y-auto sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle>Import conflicts</DialogTitle>
            <DialogDescription>
              Faylda ID yo'q, lekin quyidagi unikal ma'lumotlar bazada bor. Yangilash uchun XLSX
              faylga mos model ID ni kiriting.
            </DialogDescription>
          </DialogHeader>

          <div className="overflow-auto rounded-xl border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-20">Row</TableHead>
                  <TableHead>Field</TableHead>
                  <TableHead>Value</TableHead>
                  <TableHead>Existing ID</TableHead>
                  <TableHead>Reason</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {uploadConflicts.map((conflict, index) => (
                  <TableRow key={`${conflict.row}-${conflict.field}-${conflict.value}-${index}`}>
                    <TableCell className="font-medium">{conflict.row}</TableCell>
                    <TableCell>{conflict.field}</TableCell>
                    <TableCell className="font-mono text-xs">{conflict.value}</TableCell>
                    <TableCell>{conflict.existing_id}</TableCell>
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
