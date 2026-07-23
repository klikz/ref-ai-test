import { useEffect, useMemo, useRef, useState } from "react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { Search } from "lucide-react"
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
import { ShowErrorToast } from "@/components/showToast"
import { cn } from "@/lib/utils"
import {
  matchesProductFilter,
  normalizeProductTransferReport,
  toDateInputValue,
  useProductBalanceLines,
  type ProductTransferReportRow,
} from "./product_balance_shared"

export default function ProductBalanceReportPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const initialLineId = Number(searchParams.get("line_id") || 0) || null
  const { lines, selectedLine, setSelectedLine, linesLoading } = useProductBalanceLines(initialLineId)

  const today = useMemo(() => new Date(), [])
  const weekAgo = useMemo(() => {
    const date = new Date()
    date.setDate(date.getDate() - 7)
    return date
  }, [])

  const [dateFrom, setDateFrom] = useState(toDateInputValue(weekAgo))
  const [dateTo, setDateTo] = useState(toDateInputValue(today))
  const [rows, setRows] = useState<ProductTransferReportRow[]>([])
  const [filter, setFilter] = useState("")
  const [loading, setLoading] = useState(false)
  const loadRequestRef = useRef(0)

  useEffect(() => {
    const lineId = selectedLine?.line_id
    if (!lineId) {
      return
    }

    const requestId = ++loadRequestRef.current
    setLoading(true)

    void (async () => {
      const result = await Backend_Request<unknown>(
        { line_id: lineId, date_from: dateFrom, date_to: dateTo },
        "/api/lines/product/report",
      )
      if (requestId !== loadRequestRef.current) {
        return
      }

      setLoading(false)
      if (result.result !== "ok") {
        setRows([])
        ShowErrorToast(result.error || "Xatolik")
        return
      }

      setRows(normalizeProductTransferReport(result.data))
    })()
  }, [selectedLine?.line_id, dateFrom, dateTo])

  const query = filter.trim().toLowerCase()
  const visibleRows = query
    ? rows.filter((row) =>
        matchesProductFilter(
          [row.serial, row.line_name, row.model_name, row.modeli, row.odoo_code, row.user_name, row.transferred_at],
          query,
        ),
      )
    : rows

  return (
    <PageContainer
      title="Mahsulot o'tkazishlar"
      description="Keyingi liniyaga o'tkazilgan seriallar (transferred)"
      fullWidth
      center={
        <label className="relative flex h-14 w-full max-w-xl cursor-text items-center gap-3 rounded-2xl border-2 border-cyan-500/35 bg-cyan-50/80 px-4 shadow-md ring-1 ring-cyan-500/15 dark:border-cyan-400/30 dark:bg-cyan-950/35 dark:ring-cyan-400/10">
          <Search className="size-5 shrink-0 text-cyan-700 dark:text-cyan-300" />
          <Input
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
            placeholder="Serial yoki model bo'yicha qidirish..."
            className="h-full min-h-0 border-0 bg-transparent px-0 text-base font-medium shadow-none placeholder:text-muted-foreground/80 focus-visible:ring-0"
          />
        </label>
      }
      actions={
        <div className="flex flex-wrap items-end gap-3">
          <div className="flex flex-col gap-1">
            <Label htmlFor="date-from" className="text-xs text-muted-foreground">
              Dan
            </Label>
            <Input
              id="date-from"
              type="date"
              value={dateFrom}
              onChange={(event) => setDateFrom(event.target.value)}
              className="h-11 w-40"
            />
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="date-to" className="text-xs text-muted-foreground">
              Gacha
            </Label>
            <Input
              id="date-to"
              type="date"
              value={dateTo}
              onChange={(event) => setDateTo(event.target.value)}
              className="h-11 w-40"
            />
          </div>
          <Button
            type="button"
            variant="outline"
            className="h-11"
            onClick={() => navigate("/production/product-balance")}
          >
            Balansga qaytish
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="outline"
                className={cn(
                  "h-11 min-w-48 justify-between rounded-xl border-2 border-border/80 bg-background px-4 font-semibold",
                )}
              >
                {selectedLine?.name || "Liniya"}
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
        </div>
      }
    >
      <Panel
        title="O'tkazilgan seriallar"
        noPadding
        action={
          <span className="text-sm text-muted-foreground">
            {loading || linesLoading ? "Yuklanmoqda..." : `${visibleRows.length} ta`}
          </span>
        }
      >
        <div className="overflow-x-auto border-t border-border/60">
          <Table>
            <TableHeader>
              <TableRow className="border-b border-border/60 bg-muted/40 hover:bg-muted/40">
                <TableHead className="px-4 py-3 text-sm font-semibold">Vaqt</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Serial</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Liniya</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Model</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Modeli</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">ODOO code</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Foydalanuvchi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {visibleRows.map((row, index) => (
                <TableRow
                  key={`${row.id}-${row.serial}-${index}`}
                  className={cn("border-border/40", index % 2 === 1 && "bg-muted/20")}
                >
                  <TableCell className="px-4 py-3 text-sm tabular-nums">{row.transferred_at}</TableCell>
                  <TableCell className="px-4 py-3 font-mono text-sm">{row.serial}</TableCell>
                  <TableCell className="px-4 py-3 text-sm">{row.line_name}</TableCell>
                  <TableCell className="px-4 py-3 text-sm">{row.model_name}</TableCell>
                  <TableCell className="px-4 py-3 text-sm">{row.modeli}</TableCell>
                  <TableCell className="px-4 py-3 text-sm">{row.odoo_code || "—"}</TableCell>
                  <TableCell className="px-4 py-3 text-sm">{row.user_name || "—"}</TableCell>
                </TableRow>
              ))}
              {!loading && !linesLoading && visibleRows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                    {selectedLine ? "Ma'lumot topilmadi" : "Liniyani tanlang"}
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>
      </Panel>
    </PageContainer>
  )
}
