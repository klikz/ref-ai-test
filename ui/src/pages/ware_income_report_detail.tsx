import { useCallback, useEffect, useMemo, useState } from "react"
import { saveAs } from "file-saver"
import { ChevronLeft, ChevronRight, FileSpreadsheet } from "lucide-react"
import { useNavigate, useSearchParams } from "react-router-dom"
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  useReactTable,
} from "@tanstack/react-table"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
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
import { formatQty } from "@/lib/ware-quantity"

type WareIncomeRow = {
  id: number
  component_id: number
  manufacturer_code: string
  standard_name_uz: string
  odoo_code: string
  quantity: number
  quantity_after: number
  user_name: string
  user_login: string
  comment: string
  created_at: string
}

const PAGE_SIZE_OPTIONS = [25, 50, 100, 200] as const
const DEFAULT_PAGE_SIZE = 50

const EXPORT_HEADERS = [
  "Vaqt",
  "Komponent",
  "ODOO code",
  "Standard name",
  "Kirim",
  "Qoldiq keyin",
  "Foydalanuvchi",
  "Izoh",
]

const COLORS = {
  titleBg: "FF0F172A",
  titleText: "FFFFFFFF",
  metaBg: "FFE2E8F0",
  headerBg: "FF0891B2",
  headerText: "FFFFFFFF",
  zebra: "FFF8FAFC",
  border: "FFCBD5E1",
  positive: "FF166534",
  positiveBg: "FFDCFCE7",
  summaryBg: "FFEFF6FF",
}

const columnHelper = createColumnHelper<WareIncomeRow>()

function parsePositiveInt(value: string | null) {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0
}

function sanitizeFilePart(value: string) {
  return value.replace(/[^\w.-]+/g, "_").replace(/_+/g, "_").replace(/^_|_$/g, "") || "komponent"
}

function applyThinBorder(cell: { border?: object }) {
  cell.border = {
    top: { style: "thin", color: { argb: COLORS.border } },
    left: { style: "thin", color: { argb: COLORS.border } },
    bottom: { style: "thin", color: { argb: COLORS.border } },
    right: { style: "thin", color: { argb: COLORS.border } },
  }
}

function styleTitleRow(
  worksheet: import("exceljs").Worksheet,
  title: string,
  subtitle: string,
  columnCount: number,
) {
  worksheet.mergeCells(1, 1, 1, columnCount)
  const titleCell = worksheet.getCell(1, 1)
  titleCell.value = title
  titleCell.font = { bold: true, size: 16, color: { argb: COLORS.titleText } }
  titleCell.alignment = { vertical: "middle", horizontal: "center" }
  titleCell.fill = {
    type: "pattern",
    pattern: "solid",
    fgColor: { argb: COLORS.titleBg },
  }
  worksheet.getRow(1).height = 30

  worksheet.mergeCells(2, 1, 2, columnCount)
  const subtitleCell = worksheet.getCell(2, 1)
  subtitleCell.value = subtitle
  subtitleCell.font = { size: 11, color: { argb: "FF334155" } }
  subtitleCell.alignment = { vertical: "middle", horizontal: "center", wrapText: true }
  subtitleCell.fill = {
    type: "pattern",
    pattern: "solid",
    fgColor: { argb: COLORS.metaBg },
  }
  worksheet.getRow(2).height = 24
}

function styleHeaderRow(worksheet: import("exceljs").Worksheet, rowNumber: number, headers: string[]) {
  const row = worksheet.getRow(rowNumber)
  headers.forEach((header, index) => {
    const cell = row.getCell(index + 1)
    cell.value = header
    cell.font = { bold: true, color: { argb: COLORS.headerText } }
    cell.alignment = { vertical: "middle", horizontal: "center", wrapText: true }
    cell.fill = {
      type: "pattern",
      pattern: "solid",
      fgColor: { argb: COLORS.headerBg },
    }
    applyThinBorder(cell)
  })
  row.height = 24
}

async function buildIncomeDetailWorkbook(params: {
  componentLabel: string
  summary: string
  rows: WareIncomeRow[]
}) {
  const ExcelJS = await import("exceljs")
  const workbook = new ExcelJS.Workbook()
  workbook.creator = "Premier AC"
  workbook.created = new Date()

  const sheet = workbook.addWorksheet("Kirim tranzaksiyalari")
  styleTitleRow(sheet, "Kirim tranzaksiyalari", params.summary, EXPORT_HEADERS.length)
  styleHeaderRow(sheet, 4, EXPORT_HEADERS)

  params.rows.forEach((item, index) => {
    const rowNumber = 5 + index
    const row = sheet.getRow(rowNumber)
    row.getCell(1).value = item.created_at
    row.getCell(2).value = item.manufacturer_code
    row.getCell(3).value = item.odoo_code
    row.getCell(4).value = item.standard_name_uz
    row.getCell(5).value = Number(item.quantity)
    row.getCell(6).value = Number(item.quantity_after)
    row.getCell(7).value = item.user_name || item.user_login || ""
    row.getCell(8).value = item.comment

    row.eachCell((cell, colNumber) => {
      cell.alignment = {
        vertical: "middle",
        horizontal: colNumber === 8 ? "left" : "center",
        wrapText: colNumber === 4 || colNumber === 8,
      }
      applyThinBorder(cell)
      if (index % 2 === 1) {
        cell.fill = {
          type: "pattern",
          pattern: "solid",
          fgColor: { argb: COLORS.zebra },
        }
      }
    })

    const incomeCell = row.getCell(5)
    incomeCell.numFmt = "#,##0.####"
    incomeCell.font = { bold: true, color: { argb: COLORS.positive } }
    incomeCell.fill = {
      type: "pattern",
      pattern: "solid",
      fgColor: { argb: COLORS.positiveBg },
    }
    row.getCell(6).numFmt = "#,##0.####"
  })

  const totalQuantity = params.rows.reduce((sum, item) => sum + (Number(item.quantity) || 0), 0)
  const summaryRowNumber = 5 + params.rows.length + 1
  const summaryRow = sheet.getRow(summaryRowNumber)
  sheet.mergeCells(summaryRowNumber, 1, summaryRowNumber, 4)
  summaryRow.getCell(1).value = "Jami"
  summaryRow.getCell(5).value = totalQuantity
  summaryRow.getCell(6).value = params.rows.length
  summaryRow.getCell(7).value = "tranzaksiya"

  summaryRow.eachCell((cell, colNumber) => {
    cell.font = { bold: true }
    cell.alignment = {
      vertical: "middle",
      horizontal: colNumber === 1 ? "right" : "center",
    }
    cell.fill = {
      type: "pattern",
      pattern: "solid",
      fgColor: { argb: COLORS.summaryBg },
    }
    applyThinBorder(cell)
  })
  summaryRow.getCell(5).numFmt = "#,##0.####"
  summaryRow.getCell(5).font = { bold: true, color: { argb: COLORS.positive } }

  sheet.views = [{ state: "frozen", ySplit: 4 }]
  ;[20, 18, 18, 28, 12, 14, 18, 30].forEach((width, index) => {
    sheet.getColumn(index + 1).width = width
  })

  const buffer = await workbook.xlsx.writeBuffer()
  return buffer
}

export default function WareIncomeReportDetailPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const dateFrom = searchParams.get("date_from") ?? ""
  const dateTo = searchParams.get("date_to") ?? ""
  const componentId = parsePositiveInt(searchParams.get("component_id"))
  const manufacturerCode = searchParams.get("manufacturer_code") ?? ""
  const standardNameUz = searchParams.get("standard_name_uz") ?? ""

  const [rows, setRows] = useState<WareIncomeRow[]>([])
  const [loading, setLoading] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE)
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE })

  const filtersValid = Boolean(dateFrom && dateTo && componentId)

  const loadHistory = useCallback(async () => {
    if (!filtersValid) {
      return
    }

    setLoading(true)
    const result = await Backend_Request<WareIncomeRow[]>(
      { date_from: dateFrom, date_to: dateTo, component_id: componentId },
      "/api/ware/income/history",
    )
    setLoading(false)

    if (result.result === "ok") {
      setRows(result.data ?? [])
      setPagination((current) => ({ ...current, pageIndex: 0 }))
      return
    }

    ShowErrorToast(result.error || "Tarix yuklanmadi")
    setRows([])
  }, [componentId, dateFrom, dateTo, filtersValid])

  useEffect(() => {
    void loadHistory()
  }, [loadHistory])

  const columns = useMemo(
    () => [
      columnHelper.accessor("created_at", { header: "Vaqt" }),
      columnHelper.accessor("manufacturer_code", { header: "Komponent" }),
      columnHelper.accessor("odoo_code", { header: "ODOO code" }),
      columnHelper.accessor("standard_name_uz", { header: "Standard name" }),
      columnHelper.accessor("quantity", {
        header: "Kirim",
        cell: ({ getValue }) => (
          <span className="font-medium text-emerald-700">+{formatQty(Number(getValue()))}</span>
        ),
      }),
      columnHelper.accessor("quantity_after", {
        header: "Qoldiq keyin",
        cell: ({ getValue }) => formatQty(Number(getValue())),
      }),
      columnHelper.display({
        id: "user",
        header: "Foydalanuvchi",
        cell: ({ row }) => row.original.user_name || row.original.user_login || "-",
      }),
      columnHelper.accessor("comment", { header: "Izoh" }),
    ],
    [],
  )

  const table = useReactTable({
    data: rows,
    columns,
    state: { pagination },
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  })

  function handlePageSizeChange(size: number) {
    setPageSize(size)
    setPagination({ pageIndex: 0, pageSize: size })
  }

  const totalCount = rows.length
  const pageIndex = pagination.pageIndex
  const pageCount = Math.max(table.getPageCount(), 1)
  const from = totalCount === 0 ? 0 : pageIndex * pageSize + 1
  const to = Math.min((pageIndex + 1) * pageSize, totalCount)

  const componentLabel =
    manufacturerCode && standardNameUz
      ? `${manufacturerCode} — ${standardNameUz}`
      : manufacturerCode || standardNameUz || `ID ${componentId}`
  const summary = `${dateFrom} → ${dateTo} • ${componentLabel}`

  async function exportXlsx() {
    if (!rows.length) {
      ShowErrorToast("Export uchun ma'lumot yo'q")
      return
    }

    setExporting(true)
    try {
      const buffer = await buildIncomeDetailWorkbook({
        componentLabel,
        summary,
        rows,
      })
      const fileName = `kirim_${sanitizeFilePart(manufacturerCode || String(componentId))}_${dateFrom}_${dateTo}.xlsx`
      saveAs(
        new Blob([buffer], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
        fileName,
      )
      ShowOKToast("Excel yuklandi")
    } catch {
      ShowErrorToast("Excel yaratishda xatolik")
    } finally {
      setExporting(false)
    }
  }

  return (
    <PageContainer
      title="Kirim tranzaksiyalari"
      titleAddon={
        <Button
          variant="outline"
          size="sm"
          className="h-9 gap-2 rounded-xl"
          disabled={exporting || !rows.length}
          onClick={() => void exportXlsx()}
        >
          <FileSpreadsheet className="size-4" />
          {exporting ? "Eksport..." : "XLSX"}
        </Button>
      }
      description={filtersValid ? summary : "Filtr parametrlari noto'g'ri"}
      fullWidth
      actions={
        <Button variant="outline" className="h-10" onClick={() => navigate("/ombor/kirim/report")}>
          Orqaga
        </Button>
      }
    >
      <Panel
        title="Batafsil ro'yxat"
        action={
          <span className="text-sm text-muted-foreground">
            {loading ? "Yuklanmoqda..." : `${totalCount} ta`}
          </span>
        }
        noPadding
      >
        {!filtersValid ? (
          <div className="p-6 text-center text-muted-foreground">
            Hisobot parametrlari topilmadi
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader className="sticky top-0 z-10 bg-muted/90 backdrop-blur">
                  {table.getHeaderGroups().map((headerGroup) => (
                    <TableRow
                      key={headerGroup.id}
                      className="border-b border-border/60 bg-muted/40 hover:bg-muted/40"
                    >
                      {headerGroup.headers.map((header) => (
                        <TableHead
                          key={header.id}
                          className="sticky top-0 z-10 bg-muted/90 font-semibold backdrop-blur"
                        >
                          {flexRender(header.column.columnDef.header, header.getContext())}
                        </TableHead>
                      ))}
                    </TableRow>
                  ))}
                </TableHeader>
                <TableBody>
                  {table.getRowModel().rows.length ? (
                    table.getRowModel().rows.map((row, index) => (
                      <TableRow
                        key={row.id}
                        className={cn(
                          "border-border/40",
                          index % 2 === 0 ? "bg-transparent" : "bg-muted/20",
                        )}
                      >
                        {row.getVisibleCells().map((cell) => (
                          <TableCell key={cell.id} className="px-4 py-3 text-sm">
                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))
                  ) : (
                    <TableRow>
                      <TableCell
                        colSpan={columns.length}
                        className="h-24 text-center text-muted-foreground"
                      >
                        {loading ? "Yuklanmoqda..." : "Tranzaksiya topilmadi"}
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>

            {totalCount > 0 ? (
              <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-t border-border/50 px-4 py-3">
                <span className="text-sm text-muted-foreground">
                  {from}–{to} / {totalCount} ta
                </span>
                <div className="flex flex-wrap items-center gap-3">
                  <div className="flex items-center gap-2">
                    <Label
                      htmlFor="ware-income-detail-page-size"
                      className="text-sm text-muted-foreground"
                    >
                      Sahifada
                    </Label>
                    <select
                      id="ware-income-detail-page-size"
                      value={pageSize}
                      onChange={(event) => handlePageSizeChange(Number(event.target.value))}
                      className={cn(
                        "h-9 rounded-lg border border-input bg-background px-2.5 text-sm font-medium shadow-sm",
                        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                      )}
                    >
                      {PAGE_SIZE_OPTIONS.map((size) => (
                        <option key={size} value={size}>
                          {size}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-9 gap-1"
                      onClick={() => table.previousPage()}
                      disabled={!table.getCanPreviousPage() || loading}
                    >
                      <ChevronLeft className="size-4" />
                      Oldingi
                    </Button>
                    <span className="min-w-16 text-center text-sm font-medium">
                      {pageIndex + 1} / {pageCount}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-9 gap-1"
                      onClick={() => table.nextPage()}
                      disabled={!table.getCanNextPage() || loading}
                    >
                      Keyingi
                      <ChevronRight className="size-4" />
                    </Button>
                  </div>
                </div>
              </div>
            ) : null}
          </>
        )}
      </Panel>
    </PageContainer>
  )
}
