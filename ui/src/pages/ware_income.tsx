import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useNavigate } from "react-router-dom"
import { saveAs } from "file-saver"
import { Download, FileSpreadsheet, ImagePlus, PackagePlus, Search, Upload } from "lucide-react"
import { useDropzone } from "react-dropzone"
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
} from "@tanstack/react-table"
import { useWindowTableVirtualizer, getVirtualRowStyle } from "@/hooks/use-window-table-virtualizer"
import {
  VirtualTableCell,
  VirtualTableHeader,
  VirtualTableHeaderCell,
  VirtualTableRow,
} from "@/components/layout/virtual-table"

const WARE_INCOME_GRID_COLUMNS = "56px 1.2fr 1fr 1.4fr 1.4fr 0.8fr"
import { Backend_Request } from "@/services/backend"
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
import { ShowErrorToast, ShowOKToast } from "@/components/showToast"
import { cn } from "@/lib/utils"
import { formatQty, normalizedQty, parseQty } from "@/lib/ware-quantity"

type ProductionComponent = {
  id?: number
  manufacturer_code: string
  standard_name_uz: string
  odoo_code?: string
  photo_path?: string
}

type WareStockRow = {
  component_id: number
  quantity: number
}

type WareComponentRow = {
  id: number
  manufacturer_code: string
  standard_name_uz: string
  odoo_code: string
  photo_path?: string
  quantity: number
}

type WareIncomeRow = {
  id: number
  component_id: number
  manufacturer_code: string
  standard_name_uz: string
  quantity: number
  quantity_after: number
  user_name: string
  user_login: string
  comment: string
  created_at: string
}

function photoUrl(path?: string) {
  if (!path) {
    return ""
  }
  if (path.startsWith("http")) {
    return path
  }
  return `${Global_Data.server_ip}${path}`
}

function normalizeSearchValue(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

const columnHelper = createColumnHelper<WareComponentRow>()

export default function WareIncomePage() {
  const navigate = useNavigate()
  const searchRef = useRef<HTMLInputElement>(null)
  const quantityRef = useRef<HTMLInputElement>(null)

  const [components, setComponents] = useState<ProductionComponent[]>([])
  const [stockMap, setStockMap] = useState<Record<number, number>>({})
  const [globalFilter, setGlobalFilter] = useState("")
  const [sorting, setSorting] = useState<SortingState>([])
  const [selectedComponent, setSelectedComponent] = useState<WareComponentRow | null>(null)
  const [incomeDialogOpen, setIncomeDialogOpen] = useState(false)
  const [quantity, setQuantity] = useState("")
  const [comment, setComment] = useState("")
  const [incomeHistory, setIncomeHistory] = useState<WareIncomeRow[]>([])
  const [historyLoading, setHistoryLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [downloadingTemplate, setDownloadingTemplate] = useState(false)
  const [exporting, setExporting] = useState(false)

  const loadComponents = useCallback(async () => {
    const result = await Backend_Request<ProductionComponent[]>({}, "/api/production/components/all")
    if (result.result === "ok") {
      setComponents(result.data ?? [])
    } else {
      ShowErrorToast(result.error || "Komponentlar yuklanmadi")
    }
  }, [])

  const loadStock = useCallback(async () => {
    const result = await Backend_Request<WareStockRow[]>({}, "/api/ware/stock/all")
    if (result.result === "ok") {
      const next: Record<number, number> = {}
      for (const row of result.data ?? []) {
        next[row.component_id] = row.quantity
      }
      setStockMap(next)
    } else {
      ShowErrorToast(result.error || "Qoldiq yuklanmadi")
    }
  }, [])

  const loadComponentHistory = useCallback(async (componentId: number) => {
    setHistoryLoading(true)
    const result = await Backend_Request<WareIncomeRow[]>(
      { component_id: componentId },
      "/api/ware/income/history",
    )
    setHistoryLoading(false)
    if (result.result === "ok") {
      setIncomeHistory(result.data ?? [])
    } else {
      setIncomeHistory([])
      ShowErrorToast(result.error || "Kirim tarixi yuklanmadi")
    }
  }, [])

  useEffect(() => {
    void loadComponents()
    void loadStock()
  }, [loadComponents, loadStock])

  const focusSearch = useCallback(() => {
    window.setTimeout(() => {
      searchRef.current?.focus()
    }, 0)
  }, [])

  const resetAfterDialogClose = useCallback(() => {
    setSelectedComponent(null)
    setIncomeHistory([])
    setQuantity("")
    setComment("")
    setGlobalFilter("")
    focusSearch()
  }, [focusSearch])

  useEffect(() => {
    focusSearch()
  }, [focusSearch])

  useEffect(() => {
    if (!incomeDialogOpen) {
      return
    }
    const timer = window.setTimeout(() => {
      quantityRef.current?.focus()
      quantityRef.current?.select()
    }, 50)
    return () => window.clearTimeout(timer)
  }, [incomeDialogOpen, selectedComponent?.id])

  const tableData = useMemo<WareComponentRow[]>(() => {
    return components
      .filter((component) => component.id)
      .map((component) => ({
        id: component.id!,
        manufacturer_code: component.manufacturer_code,
        standard_name_uz: component.standard_name_uz,
        odoo_code: component.odoo_code ?? "",
        photo_path: component.photo_path,
        quantity: stockMap[component.id!] ?? 0,
      }))
      .sort((a, b) => a.manufacturer_code.localeCompare(b.manufacturer_code))
  }, [components, stockMap])

  const columns = useMemo(
    () => [
      columnHelper.display({
        id: "photo",
        header: "Rasm",
        cell: ({ row }) =>
          row.original.photo_path ? (
            <img
              src={photoUrl(row.original.photo_path)}
              alt={row.original.manufacturer_code}
              className="size-8 rounded-md border object-cover"
            />
          ) : (
            <div className="flex size-8 items-center justify-center rounded-md border border-dashed bg-muted text-muted-foreground">
              <ImagePlus className="size-3.5" />
            </div>
          ),
      }),
      columnHelper.accessor("manufacturer_code", { header: "Manufacturer code" }),
      columnHelper.accessor("standard_name_uz", { header: "Standard name" }),
      columnHelper.accessor("odoo_code", { header: "ODOO code" }),
      columnHelper.accessor("quantity", {
        header: "Qoldiq",
        cell: ({ getValue }) => formatQty(Number(getValue())),
      }),
    ],
    [],
  )

  const table = useReactTable({
    data: tableData,
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
      return [item.manufacturer_code, item.standard_name_uz, item.odoo_code, item.quantity]
        .map(normalizeSearchValue)
        .join(" ")
        .includes(query)
    },
  })

  const rows = table.getFilteredRowModel().rows

  const { listRef, virtualizer: rowVirtualizer, scrollMargin } = useWindowTableVirtualizer(rows.length, 40)

  function openIncomeDialog(component: WareComponentRow) {
    setSelectedComponent(component)
    setQuantity("")
    setComment("")
    setIncomeDialogOpen(true)
    void loadComponentHistory(component.id)
  }

  function findComponentBySearch(query: string) {
    const normalized = normalizeSearchValue(query)
    if (!normalized) {
      return null
    }

    const exact = tableData.find(
      (component) => normalizeSearchValue(component.manufacturer_code) === normalized,
    )
    if (exact) {
      return exact
    }

    const filtered = table.getFilteredRowModel().rows
    return filtered[0]?.original ?? null
  }

  function handleSearchEnter() {
    const match = findComponentBySearch(globalFilter)
    if (!match) {
      ShowErrorToast("Komponent topilmadi")
      return
    }
    openIncomeDialog(match)
  }

  async function submitIncome() {
    if (!selectedComponent || saving) {
      return false
    }
    const qty = parseQty(quantity)
    if (normalizedQty(qty) <= 0) {
      ShowErrorToast("Miqdor noto'g'ri")
      quantityRef.current?.focus()
      return false
    }

    setSaving(true)
    const result = await Backend_Request(
      {
        component_id: selectedComponent.id,
        quantity: qty,
        comment: comment.trim(),
      },
      "/api/ware/income",
    )
    setSaving(false)

    if (result.result === "ok") {
      const data = result.data as { quantity_after?: number } | undefined
      const quantityAfter = data?.quantity_after ?? selectedComponent.quantity + qty
      ShowOKToast(
        `Kirim qo'shildi: ${selectedComponent.manufacturer_code} · ${formatQty(qty)} (qoldiq: ${formatQty(quantityAfter)})`,
      )
      setIncomeDialogOpen(false)
      await loadStock()
      resetAfterDialogClose()
      return true
    }

    ShowErrorToast(result.error || "Kirim xatolik")
    quantityRef.current?.focus()
    return false
  }

  async function downloadTemplate() {
    setDownloadingTemplate(true)
    try {
      const XLSX = await import("xlsx")
      const worksheet = XLSX.utils.aoa_to_sheet([
        ["factory_code", "quantity"],
        ["ABC-001", 100.5],
      ])
      const workbook = XLSX.utils.book_new()
      XLSX.utils.book_append_sheet(workbook, worksheet, "Kirim")
      const excelBuffer = XLSX.write(workbook, { bookType: "xlsx", type: "array" })
      saveAs(
        new Blob([excelBuffer], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
        "ware_income_template.xlsx",
      )
      ShowOKToast("Shablon yuklandi")
    } catch {
      ShowErrorToast("Shablon yaratishda xatolik")
    } finally {
      setDownloadingTemplate(false)
    }
  }

  async function exportVisibleXlsx() {
    if (!rows.length) {
      ShowErrorToast("Export uchun ma'lumot yo'q")
      return
    }

    setExporting(true)
    try {
      const XLSX = await import("xlsx")
      const data = rows.map((row) => [
        row.original.manufacturer_code,
        row.original.standard_name_uz,
        row.original.odoo_code,
        row.original.quantity,
      ])
      const worksheet = XLSX.utils.aoa_to_sheet([
        ["manufacturer_code", "standard_name_uz", "odoo_code", "quantity"],
        ...data,
      ])
      const workbook = XLSX.utils.book_new()
      XLSX.utils.book_append_sheet(workbook, worksheet, "Ombor")
      const excelBuffer = XLSX.write(workbook, { bookType: "xlsx", type: "array" })
      saveAs(
        new Blob([excelBuffer], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
        "ware_stock.xlsx",
      )
      ShowOKToast("Excel yuklandi")
    } catch {
      ShowErrorToast("Excel yaratishda xatolik")
    } finally {
      setExporting(false)
    }
  }

  const onDrop = useCallback(
    async (acceptedFiles: File[]) => {
      const file = acceptedFiles[0]
      if (!file) {
        return
      }
      setUploading(true)
      const reader = new FileReader()
      reader.onload = async () => {
        const base64 = String(reader.result ?? "").split(",")[1] ?? ""
        const result = await Backend_Request({ file64: base64 }, "/api/ware/upload")
        setUploading(false)
        if (result.result === "ok") {
          const added = (result.data as { added?: number } | undefined)?.added ?? 0
          ShowOKToast(`Excel import: ${added} ta kirim`)
          await loadStock()
          if (selectedComponent) {
            await loadComponentHistory(selectedComponent.id)
          }
        } else {
          ShowErrorToast(result.error || "Excel import xatolik")
        }
      }
      reader.readAsDataURL(file)
    },
    [loadComponentHistory, loadStock, selectedComponent],
  )

  const { getRootProps, getInputProps, open } = useDropzone({
    onDrop,
    accept: {
      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": [".xlsx"],
      "application/vnd.ms-excel": [".xls"],
    },
    multiple: false,
    disabled: uploading,
    noClick: true,
    noKeyboard: true,
  })

  const headers = table.getHeaderGroups()[0]?.headers ?? []

  return (
    <PageContainer
      title="Ombor kirim"
      fullWidth
      center={
        <label className="relative flex h-14 w-full max-w-xl cursor-text items-center gap-3 rounded-2xl border-2 border-cyan-500/35 bg-cyan-50/80 px-4 shadow-md ring-1 ring-cyan-500/15 dark:border-cyan-400/30 dark:bg-cyan-950/35 dark:ring-cyan-400/10">
          <Search className="size-5 shrink-0 text-cyan-700 dark:text-cyan-300" />
          <Input
            ref={searchRef}
            value={globalFilter}
            onChange={(event) => setGlobalFilter(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault()
                handleSearchEnter()
              }
            }}
            placeholder="Komponent bo'yicha qidirish... (Enter — kirim)"
            className="h-full min-h-0 border-0 bg-transparent px-0 text-base font-medium shadow-none placeholder:text-muted-foreground/80 focus-visible:ring-0"
          />
        </label>
      }
      actions={
        <div {...getRootProps()} className="flex flex-wrap items-center gap-2">
          <input {...getInputProps()} />
          <Button
            type="button"
            variant="outline"
            className="h-10 gap-2"
            disabled={exporting || rows.length === 0}
            onClick={() => void exportVisibleXlsx()}
          >
            <FileSpreadsheet className="size-4" />
            {exporting ? "Yuklanmoqda..." : "XLSX"}
          </Button>
          <Button
            type="button"
            variant="outline"
            className="h-10 gap-2"
            disabled={downloadingTemplate || uploading}
            onClick={() => void downloadTemplate()}
          >
            <Download className="size-4" />
            {downloadingTemplate ? "Yuklanmoqda..." : "Shablon"}
          </Button>
          <Button
            type="button"
            variant="outline"
            className="h-10 gap-2"
            disabled={uploading}
            onClick={() => open()}
          >
            <Upload className="size-4" />
            {uploading ? "Import..." : "Import"}
          </Button>
          <Button
            type="button"
            variant="outline"
            className="ml-5 h-10 px-4"
            onClick={() => navigate("/ombor/kirim/report")}
          >
            Hisobot
          </Button>
        </div>
      }
    >
      <Panel noPadding>
        <div className="flex items-center justify-between border-b border-border/50 px-4 py-2 text-xs text-muted-foreground">
          <span>{rows.length} ta komponent</span>
          <span>Komponent ustiga bosing — kirim qo'shish</span>
        </div>

        <div className="w-full overflow-x-auto">
          {rows.length ? (
            <div className="w-full min-w-[900px]">
              <VirtualTableHeader gridTemplateColumns={WARE_INCOME_GRID_COLUMNS} className="top-14 bg-muted/90">
                {headers.map((header) => (
                  <VirtualTableHeaderCell
                    key={header.id}
                    onClick={header.column.getToggleSortingHandler()}
                  >
                    {flexRender(header.column.columnDef.header, header.getContext())}
                    {{
                      asc: " ▲",
                      desc: " ▼",
                    }[header.column.getIsSorted() as string] ?? null}
                  </VirtualTableHeaderCell>
                ))}
              </VirtualTableHeader>

              <div
                ref={listRef}
                className="relative w-full"
                style={{
                  height: rowVirtualizer.getTotalSize(),
                }}
              >
                {rowVirtualizer.getVirtualItems().map((virtualRow) => {
                const row = rows[virtualRow.index]
                if (!row) return null
                return (
                  <VirtualTableRow
                    key={row.id}
                    gridTemplateColumns={WARE_INCOME_GRID_COLUMNS}
                    onClick={() => openIncomeDialog(row.original)}
                    style={getVirtualRowStyle(virtualRow, scrollMargin)}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <VirtualTableCell
                        key={cell.id}
                        className={cn(
                          "text-xs sm:text-sm",
                          cell.column.id === "photo" && "flex justify-center",
                        )}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </VirtualTableCell>
                    ))}
                  </VirtualTableRow>
                )
              })}
              </div>
            </div>
          ) : (
            <div className="flex h-32 items-center justify-center text-sm text-muted-foreground">
              Komponent topilmadi
            </div>
          )}
        </div>
      </Panel>

      <Dialog
        open={incomeDialogOpen}
        onOpenChange={(open) => {
          setIncomeDialogOpen(open)
          if (!open) {
            resetAfterDialogClose()
          }
        }}
      >
        <DialogContent className="gap-5 p-5 sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Kirim qo'shish</DialogTitle>
            <DialogDescription>
              {selectedComponent?.manufacturer_code} — {selectedComponent?.standard_name_uz}
            </DialogDescription>
          </DialogHeader>

          {selectedComponent ? (
            <div className="space-y-4">
              <div className="flex items-center gap-3 rounded-xl border bg-muted/20 p-3">
                {selectedComponent.photo_path ? (
                  <img
                    src={photoUrl(selectedComponent.photo_path)}
                    alt={selectedComponent.manufacturer_code}
                    className="size-14 rounded-lg border object-cover"
                  />
                ) : (
                  <div className="flex size-14 items-center justify-center rounded-lg border border-dashed bg-muted text-muted-foreground">
                    <ImagePlus className="size-5" />
                  </div>
                )}
                <div className="min-w-0 flex-1">
                  <p className="truncate font-semibold">{selectedComponent.manufacturer_code}</p>
                  <p className="truncate text-sm text-muted-foreground">
                    {selectedComponent.standard_name_uz}
                  </p>
                  <p className="mt-1 text-sm">
                    Joriy qoldiq:{" "}
                    <span className="font-semibold text-primary">
                      {formatQty(selectedComponent.quantity)}
                    </span>
                  </p>
                </div>
              </div>

              <div className="grid gap-3 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="income-quantity">Kirim miqdori</Label>
                  <Input
                    ref={quantityRef}
                    id="income-quantity"
                    inputMode="decimal"
                    value={quantity}
                    onChange={(event) => setQuantity(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter") {
                        event.preventDefault()
                        void submitIncome()
                      }
                    }}
                    placeholder="Masalan: 100 yoki 0.125"
                    className="h-10"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="income-comment">Izoh</Label>
                  <Input
                    id="income-comment"
                    value={comment}
                    onChange={(event) => setComment(event.target.value)}
                    placeholder="Ixtiyoriy"
                    className="h-10"
                  />
                </div>
              </div>

              <div className="rounded-xl border">
                <div className="border-b px-3 py-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  Kirim tarixi
                </div>
                <div className="max-h-44 overflow-auto">
                  {historyLoading ? (
                    <p className="px-3 py-4 text-sm text-muted-foreground">Yuklanmoqda...</p>
                  ) : incomeHistory.length ? (
                    <div className="divide-y">
                      {incomeHistory.map((item) => (
                        <div key={item.id} className="flex items-start justify-between gap-3 px-3 py-2 text-sm">
                          <div className="min-w-0">
                            <p className="font-medium text-emerald-700">
                              +{formatQty(item.quantity)}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {item.created_at} · {item.user_name || item.user_login || "-"}
                            </p>
                            {item.comment ? (
                              <p className="truncate text-xs text-muted-foreground">{item.comment}</p>
                            ) : null}
                          </div>
                          <span className="shrink-0 text-xs text-muted-foreground">
                            {formatQty(item.quantity_after)}
                          </span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="px-3 py-4 text-sm text-muted-foreground">Kirim tarixi yo'q</p>
                  )}
                </div>
              </div>
            </div>
          ) : null}

          <DialogFooter className="mx-0 mb-0 gap-3 rounded-b-xl border-t bg-muted/50 pt-4">
            <Button
              className="h-10 w-full gap-2 px-5 sm:w-auto"
              disabled={saving || !selectedComponent}
              onClick={() => void submitIncome()}
            >
              <PackagePlus className="size-4" />
              {saving ? "Saqlanmoqda..." : "Kirim qilish"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageContainer>
  )
}
