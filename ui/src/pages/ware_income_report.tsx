import { useCallback, useEffect, useMemo, useState } from "react"

import { saveAs } from "file-saver"

import { ChevronLeft, ChevronRight, FileSpreadsheet, Search } from "lucide-react"

import { useNavigate } from "react-router-dom"

import {

  createColumnHelper,

  flexRender,

  getCoreRowModel,

  getFilteredRowModel,

  getPaginationRowModel,

  useReactTable,

  type Table as ReactTable,

} from "@tanstack/react-table"

import { Backend_Request } from "@/services/backend"

import { PageContainer } from "@/components/layout/page-container"

import { Panel } from "@/components/layout/panel"

import { Button } from "@/components/ui/button"

import { Input } from "@/components/ui/input"

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



type ComponentSummaryRow = {

  component_id: number

  manufacturer_code: string

  standard_name_uz: string

  odoo_code: string

  total_quantity: number

  transaction_count: number

}



const PAGE_SIZE_OPTIONS = [25, 50, 100, 200] as const

const DEFAULT_PAGE_SIZE = 50



function toDateInputValue(date: Date) {

  return date.toISOString().slice(0, 10)

}



function normalizeSearchValue(value: unknown) {

  return String(value ?? "").trim().toLocaleLowerCase()

}



function buildComponentSummary(history: WareIncomeRow[]) {

  const grouped = new Map<number, ComponentSummaryRow>()



  for (const row of history) {

    if (!row.component_id) {

      continue

    }



    const existing = grouped.get(row.component_id)

    if (existing) {

      existing.total_quantity += Number(row.quantity) || 0

      existing.transaction_count += 1

      if (!existing.manufacturer_code && row.manufacturer_code) {

        existing.manufacturer_code = row.manufacturer_code

      }

      if (!existing.standard_name_uz && row.standard_name_uz) {

        existing.standard_name_uz = row.standard_name_uz

      }

      if (!existing.odoo_code && row.odoo_code) {

        existing.odoo_code = row.odoo_code

      }

      continue

    }



    grouped.set(row.component_id, {

      component_id: row.component_id,

      manufacturer_code: row.manufacturer_code ?? "",

      standard_name_uz: row.standard_name_uz ?? "",

      odoo_code: row.odoo_code ?? "",

      total_quantity: Number(row.quantity) || 0,

      transaction_count: 1,

    })

  }



  return Array.from(grouped.values()).sort((left, right) =>

    left.manufacturer_code.localeCompare(right.manufacturer_code, undefined, { sensitivity: "base" }),

  )

}



const columnHelper = createColumnHelper<ComponentSummaryRow>()



function TablePaginationControls<T>({

  table,

  controlId,

  pageSize,

  onPageSizeChange,

}: {

  table: ReactTable<T>

  controlId: string

  pageSize: number

  onPageSizeChange: (size: number) => void

}) {

  const filteredCount = table.getFilteredRowModel().rows.length

  const pageIndex = table.getState().pagination.pageIndex

  const pageCount = table.getPageCount()



  if (filteredCount === 0) {

    return null

  }



  const from = pageIndex * pageSize + 1

  const to = Math.min((pageIndex + 1) * pageSize, filteredCount)



  return (

    <div className="mt-3 flex flex-wrap items-center justify-between gap-3 border-t border-border/50 pt-3">

      <span className="text-sm text-muted-foreground">

        {from}–{to} / {filteredCount} ta

      </span>

      <div className="flex flex-wrap items-center gap-3">

        <div className="flex items-center gap-2">

          <Label htmlFor={`page-size-${controlId}`} className="text-sm text-muted-foreground">

            Sahifada

          </Label>

          <select

            id={`page-size-${controlId}`}

            value={pageSize}

            onChange={(event) => onPageSizeChange(Number(event.target.value))}

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

            disabled={!table.getCanPreviousPage()}

          >

            <ChevronLeft className="size-4" />

            Oldingi

          </Button>

          <span className="min-w-16 text-center text-sm font-medium">

            {pageIndex + 1} / {Math.max(pageCount, 1)}

          </span>

          <Button

            variant="outline"

            size="sm"

            className="h-9 gap-1"

            onClick={() => table.nextPage()}

            disabled={!table.getCanNextPage()}

          >

            Keyingi

            <ChevronRight className="size-4" />

          </Button>

        </div>

      </div>

    </div>

  )

}



export default function WareIncomeReportPage() {

  const navigate = useNavigate()

  const [history, setHistory] = useState<WareIncomeRow[]>([])

  const [globalFilter, setGlobalFilter] = useState("")

  const [dateFrom, setDateFrom] = useState(toDateInputValue(new Date()))

  const [dateTo, setDateTo] = useState(toDateInputValue(new Date()))

  const [loading, setLoading] = useState(false)

  const [exporting, setExporting] = useState(false)

  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE)

  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE })



  const loadHistory = useCallback(async () => {

    setLoading(true)

    const result = await Backend_Request<WareIncomeRow[]>(

      { date_from: dateFrom, date_to: dateTo },

      "/api/ware/income/history",

    )

    setLoading(false)

    if (result.result === "ok") {

      setHistory(result.data ?? [])

      setPagination((current) => ({ ...current, pageIndex: 0 }))

    } else {

      ShowErrorToast(result.error || "Tarix yuklanmadi")

    }

  }, [dateFrom, dateTo])



  useEffect(() => {

    void loadHistory()

  }, [loadHistory])



  const summaryRows = useMemo(() => buildComponentSummary(history), [history])



  function openComponentDetail(row: ComponentSummaryRow) {

    if (!row.component_id) {

      ShowErrorToast("Komponent aniqlanmadi")

      return

    }



    const params = new URLSearchParams({

      date_from: dateFrom,

      date_to: dateTo,

      component_id: String(row.component_id),

      manufacturer_code: row.manufacturer_code,

      standard_name_uz: row.standard_name_uz,

    })

    navigate(`/ombor/kirim/report/detail?${params.toString()}`)

  }



  const columns = useMemo(

    () => [

      columnHelper.accessor("manufacturer_code", {

        header: "Komponent",

        cell: ({ row, getValue }) => (

          <button

            type="button"

            className="text-left font-medium text-primary underline-offset-2 hover:underline"

            onClick={(event) => {

              event.stopPropagation()

              openComponentDetail(row.original)

            }}

          >

            {getValue() || `ID ${row.original.component_id}`}

          </button>

        ),

      }),

      columnHelper.accessor("odoo_code", { header: "ODOO code" }),

      columnHelper.accessor("standard_name_uz", { header: "Standard name" }),

      columnHelper.accessor("total_quantity", {

        header: "Jami kirim",

        cell: ({ getValue }) => (

          <span className="font-medium text-emerald-700 tabular-nums">

            +{formatQty(Number(getValue()))}

          </span>

        ),

      }),

      columnHelper.accessor("transaction_count", {

        header: "Tranzaksiyalar",

        cell: ({ getValue }) => <span className="tabular-nums">{formatQty(Number(getValue()))}</span>,

      }),

    ],

    [dateFrom, dateTo],

  )



  const table = useReactTable({

    data: summaryRows,

    columns,

    state: { globalFilter, pagination },

    onGlobalFilterChange: setGlobalFilter,

    onPaginationChange: setPagination,

    getCoreRowModel: getCoreRowModel(),

    getFilteredRowModel: getFilteredRowModel(),

    getPaginationRowModel: getPaginationRowModel(),

    globalFilterFn: (row, _columnId, filterValue) => {

      const query = normalizeSearchValue(filterValue)

      if (!query) {

        return true

      }

      const item = row.original

      return [item.manufacturer_code, item.standard_name_uz, item.odoo_code]

        .map(normalizeSearchValue)

        .join(" ")

        .includes(query)

    },

  })



  function handlePageSizeChange(size: number) {

    setPageSize(size)

    setPagination({ pageIndex: 0, pageSize: size })

  }



  async function exportXlsx() {

    if (!history.length) {

      ShowErrorToast("Export uchun ma'lumot yo'q")

      return

    }



    setExporting(true)

    try {

      const XLSX = await import("xlsx")

      const data = history.map((row) => [

        row.created_at,

        row.manufacturer_code,

        row.odoo_code,

        row.standard_name_uz,

        row.quantity,

        row.quantity_after,

        row.user_name || row.user_login || "",

        row.comment,

      ])

      const worksheet = XLSX.utils.aoa_to_sheet([

        ["vaqt", "manufacturer_code", "odoo_code", "standard_name_uz", "kirim", "qoldiq_keyin", "foydalanuvchi", "izoh"],

        ...data,

      ])

      const workbook = XLSX.utils.book_new()

      XLSX.utils.book_append_sheet(workbook, worksheet, "Kirim")

      const excelBuffer = XLSX.write(workbook, { bookType: "xlsx", type: "array" })

      saveAs(

        new Blob([excelBuffer], {

          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",

        }),

        `ware_income_${dateFrom}_${dateTo}.xlsx`,

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

      title="Kirim hisoboti"

      description="Omborga qabul qilingan komponentlar tarixi"

      fullWidth

      center={

        <label className="relative flex h-14 w-full max-w-xl cursor-text items-center gap-3 rounded-2xl border-2 border-cyan-500/35 bg-cyan-50/80 px-4 shadow-md ring-1 ring-cyan-500/15 dark:border-cyan-400/30 dark:bg-cyan-950/35 dark:ring-cyan-400/10">

          <Search className="size-5 shrink-0 text-cyan-700 dark:text-cyan-300" />

          <Input

            value={globalFilter}

            onChange={(event) => setGlobalFilter(event.target.value)}

            placeholder="Komponent bo'yicha qidirish..."

            className="h-full min-h-0 border-0 bg-transparent px-0 text-base font-medium shadow-none placeholder:text-muted-foreground/80 focus-visible:ring-0"

          />

        </label>

      }

      actions={

        <Button variant="outline" className="h-10" onClick={() => navigate("/ombor/kirim")}>

          Kirim

        </Button>

      }

    >

      <Panel

        title="Kirimlar tarixi"

        description="Komponent ustiga bosing — tanlangan davr bo'yicha batafsil tranzaksiyalar"

        action={

          <span className="text-sm text-muted-foreground">

            {table.getFilteredRowModel().rows.length} ta komponent

          </span>

        }

      >

        <div className="mb-4 flex shrink-0 flex-wrap items-end gap-3">

          <div className="space-y-2">

            <Label htmlFor="ware-report-date-from">Dan</Label>

            <Input

              id="ware-report-date-from"

              type="date"

              value={dateFrom}

              onChange={(event) => setDateFrom(event.target.value)}

            />

          </div>

          <div className="space-y-2">

            <Label htmlFor="ware-report-date-to">Gacha</Label>

            <Input

              id="ware-report-date-to"

              type="date"

              value={dateTo}

              onChange={(event) => setDateTo(event.target.value)}

            />

          </div>

          <Button variant="outline" onClick={() => void loadHistory()} disabled={loading}>

            {loading ? "Yuklanmoqda..." : "Yangilash"}

          </Button>

          <Button onClick={() => void exportXlsx()} disabled={exporting || !history.length}>

            <FileSpreadsheet className="size-4" />

            {exporting ? "Eksport..." : "XLSX"}

          </Button>

        </div>



        <div className="overflow-x-auto rounded-xl border border-border/60">

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

                table.getRowModel().rows.map((row) => (

                  <TableRow

                    key={row.id}

                    className="cursor-pointer hover:bg-muted/50"

                    onClick={() => openComponentDetail(row.original)}

                  >

                    {row.getVisibleCells().map((cell) => (

                      <TableCell key={cell.id}>

                        {flexRender(cell.column.columnDef.cell, cell.getContext())}

                      </TableCell>

                    ))}

                  </TableRow>

                ))

              ) : (

                <TableRow>

                  <TableCell colSpan={columns.length} className="h-20 text-center text-muted-foreground">

                    {loading ? "Yuklanmoqda..." : "Kirim topilmadi"}

                  </TableCell>

                </TableRow>

              )}

            </TableBody>

          </Table>

        </div>



        <div className="shrink-0">

          <TablePaginationControls

            table={table}

            controlId="ware-income-report"

            pageSize={pageSize}

            onPageSizeChange={handlePageSizeChange}

          />

        </div>

      </Panel>

    </PageContainer>

  )

}


