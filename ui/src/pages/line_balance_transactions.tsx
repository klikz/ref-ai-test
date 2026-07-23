import { useEffect, useMemo, useState } from "react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { FileDown, Search } from "lucide-react"
import { flexRender } from "@tanstack/react-table"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
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
import {
  DEFAULT_PAGE_SIZE,
  TablePaginationControls,
  buildBalanceWorkbook,
  downloadBlob,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  matchesComponentFilter,
  normalizeBalanceRows,
  normalizeBalanceTransactions,
  toDateInputValue,
  txColumnHelper,
  useLineBalanceLines,
  useReactTable,
  type BalanceRow,
  type BalanceTransaction,
} from "./line_balance_shared"

function defaultDateFrom() {
  const date = new Date()
  date.setDate(date.getDate() - 30)
  return toDateInputValue(date)
}

export default function LineBalanceTransactionsPage() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const lineIdParam = Number(searchParams.get("line_id") || 0)
  const initialLineId = lineIdParam > 0 ? lineIdParam : null

  const { lines, selectedLine, setSelectedLine, linesLoading } = useLineBalanceLines(initialLineId)
  const [balances, setBalances] = useState<BalanceRow[]>([])
  const [transactions, setTransactions] = useState<BalanceTransaction[]>([])
  const [balanceFilter, setBalanceFilter] = useState("")
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE)
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE })
  const [dateFrom, setDateFrom] = useState(defaultDateFrom)
  const [dateTo, setDateTo] = useState(() => toDateInputValue(new Date()))
  const [exporting, setExporting] = useState(false)
  const [transactionsLoading, setTransactionsLoading] = useState(false)

  useEffect(() => {
    if (!selectedLine?.line_id) {
      return
    }
    const params = new URLSearchParams(searchParams)
    if (params.get("line_id") === String(selectedLine.line_id)) {
      return
    }
    params.set("line_id", String(selectedLine.line_id))
    setSearchParams(params, { replace: true })
  }, [searchParams, selectedLine?.line_id, setSearchParams])

  async function loadBalances(lineId: number) {
    const result = await Backend_Request<BalanceRow[]>({ line_id: lineId }, "/api/lines/balance")
    if (result.result === "ok") {
      setBalances(normalizeBalanceRows(result.data))
    }
  }

  async function loadTransactions(lineId: number) {
    setTransactionsLoading(true)
    const result = await Backend_Request<BalanceTransaction[]>(
      {
        line_id: lineId,
        date_from: dateFrom,
        date_to: dateTo,
      },
      "/api/lines/balance/transactions",
    )
    setTransactionsLoading(false)
    if (result.result === "ok") {
      setTransactions(normalizeBalanceTransactions(result.data))
      setPagination((current) => ({ ...current, pageIndex: 0 }))
      return
    }
    setTransactions([])
    ShowErrorToast(result.error || "Xatolik")
  }

  useEffect(() => {
    if (!selectedLine?.line_id) {
      setBalances([])
      setTransactions([])
      return
    }
    void loadBalances(selectedLine.line_id)
    void loadTransactions(selectedLine.line_id)
  }, [selectedLine?.line_id, dateFrom, dateTo])

  const transactionColumns = useMemo(
    () => [
      txColumnHelper.accessor("created_at", { header: "Vaqt" }),
      txColumnHelper.accessor("manufacturer_code", { header: "Komponent" }),
      txColumnHelper.accessor("odoo_code", {
        header: "ODOO code",
        cell: ({ getValue }) => String(getValue() ?? "") || "—",
      }),
      txColumnHelper.accessor("quantity_change", {
        header: "O'zgarish",
        cell: ({ getValue }) => {
          const value = Number(getValue())
          const prefix = value > 0 ? "+" : ""
          return `${prefix}${value.toLocaleString()}`
        },
      }),
      txColumnHelper.accessor("quantity_after", {
        header: "Balans keyin",
        cell: ({ getValue }) => Number(getValue()).toLocaleString(),
      }),
      txColumnHelper.display({
        id: "user",
        header: "Foydalanuvchi",
        cell: ({ row }) => row.original.user_name || row.original.user_login || "-",
      }),
      txColumnHelper.accessor("source", { header: "Manba" }),
      txColumnHelper.accessor("comment", { header: "Izoh" }),
    ],
    [],
  )

  const transactionTable = useReactTable({
    data: transactions,
    columns: transactionColumns,
    state: { globalFilter: balanceFilter, pagination },
    onGlobalFilterChange: setBalanceFilter,
    onPaginationChange: setPagination,
    autoResetPageIndex: true,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getRowId: (row) => String(row.id),
    globalFilterFn: (row, _columnId, filterValue) => {
      const item = row.original
      return matchesComponentFilter(
        [
          item.manufacturer_code,
          item.odoo_code,
          item.standard_name_uz,
          item.component_id,
          item.comment,
          item.user_name,
          item.user_login,
        ],
        String(filterValue ?? ""),
      )
    },
  })

  function handlePageSizeChange(size: number) {
    setPageSize(size)
    setPagination({ pageIndex: 0, pageSize: size })
  }

  const visibleRows = transactionTable.getPaginationRowModel().rows

  async function exportToExcel() {
    if (!selectedLine) {
      ShowErrorToast("Liniyani tanlang")
      return
    }
    if (transactions.length === 0 && balances.length === 0) {
      ShowErrorToast("Export uchun ma'lumot yo'q")
      return
    }

    setExporting(true)
    try {
      const fileData = await buildBalanceWorkbook({
        lineName: selectedLine.name,
        dateFrom,
        dateTo,
        balances,
        transactions,
      })
      const safeLine = selectedLine.name.replace(/[^\w.-]+/g, "_")
      downloadBlob(fileData, `balance_${safeLine}_${dateFrom}_${dateTo}.xlsx`)
      ShowOKToast("XLSX fayl yuklandi")
    } catch {
      ShowErrorToast("Exportda xatolik")
    } finally {
      setExporting(false)
    }
  }

  return (
    <PageContainer
      title="Tranzaksiyalar tarixi"
      description={selectedLine ? `Liniya: ${selectedLine.name}` : "Liniya tranzaksiyalari"}
      fullWidth
      center={
        <label className="relative flex h-14 w-full max-w-xl cursor-text items-center gap-3 rounded-2xl border-2 border-cyan-500/35 bg-cyan-50/80 px-4 shadow-md ring-1 ring-cyan-500/15 dark:border-cyan-400/30 dark:bg-cyan-950/35 dark:ring-cyan-400/10">
          <Search className="size-5 shrink-0 text-cyan-700 dark:text-cyan-300" />
          <Input
            value={balanceFilter}
            onChange={(event) => setBalanceFilter(event.target.value)}
            placeholder="Komponent bo'yicha qidirish..."
            className="h-full min-h-0 border-0 bg-transparent px-0 text-base font-medium shadow-none placeholder:text-muted-foreground/80 focus-visible:ring-0"
          />
        </label>
      }
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="outline"
                className={cn(
                  "h-14 min-w-56 justify-between rounded-2xl border-2 border-border/80 bg-background px-4 text-base font-semibold shadow-md",
                  "hover:border-primary/40 hover:bg-muted/50",
                )}
              >
                {selectedLine?.name || "Liniyani tanlang"}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent className="rounded-xl">
              <DropdownMenuGroup>
                {lines.map((line) => (
                  <DropdownMenuItem key={line.line_id} onClick={() => setSelectedLine(line)}>
                    {line.name}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
          <Button
            type="button"
            variant="outline"
            className="h-14 rounded-2xl px-5"
            onClick={() => navigate("/production/balance")}
          >
            Orqaga
          </Button>
        </div>
      }
    >
      <Panel
        title="Tranzaksiyalar tarixi"
        noPadding
        action={
          <span className="text-sm text-muted-foreground">
            {transactionsLoading || linesLoading
              ? "Yuklanmoqda..."
              : `${transactionTable.getFilteredRowModel().rows.length} ta`}
          </span>
        }
      >
        <div className="flex shrink-0 flex-wrap items-end gap-3 border-b border-border/60 px-4 py-3">
          <div className="space-y-2">
            <Label htmlFor="date-from">Dan</Label>
            <Input id="date-from" type="date" value={dateFrom} onChange={(e) => setDateFrom(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="date-to">Gacha</Label>
            <Input id="date-to" type="date" value={dateTo} onChange={(e) => setDateTo(e.target.value)} />
          </div>
          <Button
            variant="outline"
            onClick={() => selectedLine && void loadTransactions(selectedLine.line_id)}
            disabled={!selectedLine}
          >
            Yangilash
          </Button>
          <Button onClick={() => void exportToExcel()} disabled={!selectedLine || exporting}>
            <FileDown className="size-4" />
            {exporting ? "Eksport..." : "XLSX eksport"}
          </Button>
        </div>

        <div className="overflow-x-auto border-t border-border/60">
          <Table>
            <TableHeader className="sticky top-14 z-10 bg-muted/90 backdrop-blur">
              {transactionTable.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id} className="border-b border-border/60 bg-muted/40 hover:bg-muted/40">
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id} className="px-4 py-3 text-sm font-semibold">
                      {flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {visibleRows.length ? (
                visibleRows.map((row, index) => (
                  <TableRow
                    key={row.id}
                    className={cn("border-border/40", index % 2 === 1 && "bg-muted/20")}
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
                  <TableCell colSpan={transactionColumns.length} className="h-24 text-center text-muted-foreground">
                    {transactionsLoading || linesLoading
                      ? "Yuklanmoqda..."
                      : selectedLine
                        ? "Tranzaksiya topilmadi"
                        : "Liniyani tanlang"}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
        <div className="shrink-0 border-t border-border/60 px-4 pb-3 pt-3">
          <TablePaginationControls
            table={transactionTable}
            controlId="transactions"
            pageSize={pageSize}
            onPageSizeChange={handlePageSizeChange}
          />
        </div>
      </Panel>
    </PageContainer>
  )
}
