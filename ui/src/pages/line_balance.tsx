import { useEffect, useRef, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Search } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
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
  matchesComponentFilter,
  normalizeBalanceRows,
  useLineBalanceLines,
  type BalanceRow,
} from "./line_balance_shared"

export default function LineBalancePage() {
  const navigate = useNavigate()
  const { lines, selectedLine, setSelectedLine, linesLoading } = useLineBalanceLines()
  const [balances, setBalances] = useState<BalanceRow[]>([])
  const [balanceFilter, setBalanceFilter] = useState("")
  const [balanceLoading, setBalanceLoading] = useState(false)
  const loadRequestRef = useRef(0)

  useEffect(() => {
    const lineId = selectedLine?.line_id
    if (!lineId) {
      return
    }

    const requestId = ++loadRequestRef.current
    setBalanceLoading(true)

    void (async () => {
      const result = await Backend_Request<BalanceRow[]>({ line_id: lineId }, "/api/lines/balance")
      if (requestId !== loadRequestRef.current) {
        return
      }

      setBalanceLoading(false)
      if (result.result !== "ok") {
        setBalances([])
        ShowErrorToast(result.error || "Xatolik")
        return
      }

      setBalances(normalizeBalanceRows(result.data))
    })()
  }, [selectedLine?.line_id])

  const query = balanceFilter.trim().toLowerCase()
  const visibleRows = query
    ? balances.filter((row) =>
        matchesComponentFilter([row.main_code, row.odoo_code, row.full_name_uz, row.quantity], query),
      )
    : balances

  function openTransactions() {
    if (!selectedLine) {
      ShowErrorToast("Liniyani tanlang")
      return
    }
    navigate(`/production/balance/transactions?line_id=${selectedLine.line_id}`)
  }

  return (
    <PageContainer
      title="Liniya balansi"
      description="Har bir liniya bo'yicha komponent qoldig'i"
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
          <Button type="button" className="h-14 rounded-2xl px-5" onClick={openTransactions}>
            Tranzaksiyalar tarixi
          </Button>
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
        </div>
      }
    >
      <Panel
        title="Joriy balans"
        noPadding
        action={
          <span className="text-sm text-muted-foreground">
            {balanceLoading || linesLoading ? "Yuklanmoqda..." : `${visibleRows.length} ta`}
          </span>
        }
      >
        <div className="overflow-x-auto border-t border-border/60">
          <Table>
            <TableHeader>
              <TableRow className="border-b border-border/60 bg-muted/40 hover:bg-muted/40">
                <TableHead className="px-4 py-3 text-sm font-semibold">Korxona kodi</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">ODOO code</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Komponent nomi</TableHead>
                <TableHead className="px-4 py-3 text-right text-sm font-semibold">Soni</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {visibleRows.map((row, index) => (
                <TableRow
                  key={`${row.id}-${row.component_id}-${index}`}
                  className={cn("border-border/40", index % 2 === 1 && "bg-muted/20")}
                >
                  <TableCell className="px-4 py-3 text-sm">{row.main_code}</TableCell>
                  <TableCell className="px-4 py-3 text-sm">{row.odoo_code || "—"}</TableCell>
                  <TableCell className="px-4 py-3 text-sm">{row.full_name_uz || "—"}</TableCell>
                  <TableCell className="px-4 py-3 text-right text-sm font-medium tabular-nums">
                    {Number(row.quantity).toLocaleString()}
                  </TableCell>
                </TableRow>
              ))}
              {!balanceLoading && !linesLoading && visibleRows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className="h-24 text-center text-muted-foreground">
                    {selectedLine ? "Balans bo'sh" : "Liniyani tanlang"}
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
