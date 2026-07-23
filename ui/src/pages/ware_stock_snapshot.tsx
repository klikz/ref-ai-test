import { useEffect, useMemo, useState } from "react"
import { Search } from "lucide-react"
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
import { ShowErrorToast } from "@/components/showToast"
import { cn } from "@/lib/utils"

type SnapshotRow = {
  snapshot_date: string
  component_id: number
  manufacturer_code: string
  standard_name_uz: string
  quantity: number
}

const PAGE_SIZE_OPTIONS = [25, 50, 100, 200] as const
const DEFAULT_PAGE_SIZE = 50
const columnHelper = createColumnHelper<SnapshotRow>()

function toDateInputValue(date: Date) {
  return date.toISOString().slice(0, 10)
}

function normalizeSearchValue(value: unknown) {
  return String(value ?? "").trim().toLocaleLowerCase()
}

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
      </div>
    </div>
  )
}

export default function WareStockSnapshotPage() {
  const [snapshotDate, setSnapshotDate] = useState(toDateInputValue(new Date()))
  const [rows, setRows] = useState<SnapshotRow[]>([])
  const [loading, setLoading] = useState(false)
  const [globalFilter, setGlobalFilter] = useState("")
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE)
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE })

  async function loadSnapshot() {
    setLoading(true)
    const result = await Backend_Request<SnapshotRow[]>(
      { snapshot_date: snapshotDate },
      "/api/ware/stock/snapshot",
    )
    setLoading(false)
    if (result.result === "ok") {
      setRows(result.data ?? [])
      setPagination((current) => ({ ...current, pageIndex: 0 }))
    } else {
      ShowErrorToast(result.error || "Slepok yuklanmadi")
    }
  }

  useEffect(() => {
    void loadSnapshot()
  }, [])

  const columns = useMemo(
    () => [
      columnHelper.accessor("manufacturer_code", { header: "Komponent" }),
      columnHelper.accessor("standard_name_uz", { header: "Standard name" }),
      columnHelper.accessor("quantity", {
        header: "Qoldiq",
        cell: ({ getValue }) => Number(getValue()).toLocaleString(),
      }),
    ],
    [],
  )

  const table = useReactTable({
    data: rows,
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
      return [item.manufacturer_code, item.standard_name_uz]
        .map(normalizeSearchValue)
        .join(" ")
        .includes(query)
    },
  })

  function handlePageSizeChange(size: number) {
    setPageSize(size)
    setPagination({ pageIndex: 0, pageSize: size })
  }

  return (
    <PageContainer
      title="Ombor qoldiq slеpogi"
      description="Tanlangan sana bo'yicha komponent qoldiqlari"
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
    >
      <Panel
        title="Qoldiqlar"
        action={<span className="text-sm text-muted-foreground">{table.getFilteredRowModel().rows.length} ta</span>}
      >
        <div className="mb-4 flex shrink-0 flex-wrap items-end gap-3">
          <div className="space-y-2">
            <Label htmlFor="snapshot-date">Sana</Label>
            <Input
              id="snapshot-date"
              type="date"
              value={snapshotDate}
              onChange={(event) => setSnapshotDate(event.target.value)}
            />
          </div>
          <Button variant="outline" onClick={() => void loadSnapshot()} disabled={loading}>
            {loading ? "Yuklanmoqda..." : "Ko'rish"}
          </Button>
        </div>

        <div className="overflow-x-auto rounded-xl border border-border/60">
          <Table>
            <TableHeader className="sticky top-14 z-10 bg-muted/90 backdrop-blur">
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id} className="border-b border-border/60 bg-muted/40 hover:bg-muted/40">
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id} className="sticky top-14 z-10 bg-muted/90 font-semibold backdrop-blur">
                      {flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows.length ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={columns.length} className="h-20 text-center text-muted-foreground">
                    {loading ? "Yuklanmoqda..." : "Ma'lumot topilmadi"}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>

        <div className="shrink-0">
          <TablePaginationControls
            table={table}
            controlId="ware-stock-snapshot"
            pageSize={pageSize}
            onPageSizeChange={handlePageSizeChange}
          />
        </div>
      </Panel>
    </PageContainer>
  )
}
