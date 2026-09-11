import { useEffect, useRef, useState } from "react"
import { useNavigate } from "react-router-dom"
import { FileDown } from "lucide-react"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
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
  buildProductBalanceWorkbook,
  downloadBlob,
  normalizeProductBalance,
  useProductBalanceLines,
  type ProductBalanceItem,
  type ProductBalanceSummaryRow,
} from "./product_balance_shared"

export default function ProductBalancePage() {
  const navigate = useNavigate()
  const { lines, selectedLine, setSelectedLine, linesLoading } = useProductBalanceLines()
  const [summary, setSummary] = useState<ProductBalanceSummaryRow[]>([])
  const [items, setItems] = useState<ProductBalanceItem[]>([])
  const [balanceLoading, setBalanceLoading] = useState(false)
  const [exporting, setExporting] = useState(false)
  const loadRequestRef = useRef(0)

  useEffect(() => {
    const lineId = selectedLine?.line_id
    if (!lineId) {
      return
    }

    const requestId = ++loadRequestRef.current
    setBalanceLoading(true)

    void (async () => {
      const result = await Backend_Request<unknown>({ line_id: lineId }, "/api/lines/product/balance")
      if (requestId !== loadRequestRef.current) {
        return
      }

      setBalanceLoading(false)
      if (result.result !== "ok") {
        setSummary([])
        setItems([])
        ShowErrorToast(result.error || "Xatolik")
        return
      }

      const parsed = normalizeProductBalance(result.data)
      setSummary(parsed.summary)
      setItems(parsed.items)
    })()
  }, [selectedLine?.line_id])

  const totalCount = summary.reduce((sum, row) => sum + row.count, 0)

  function openReport() {
    if (!selectedLine) {
      ShowErrorToast("Liniyani tanlang")
      return
    }
    navigate(`/production/product-balance/report?line_id=${selectedLine.line_id}`)
  }

  async function exportToExcel() {
    if (!selectedLine) {
      ShowErrorToast("Liniyani tanlang")
      return
    }
    if (summary.length === 0 && items.length === 0) {
      ShowErrorToast("Export uchun ma'lumot yo'q")
      return
    }

    setExporting(true)
    try {
      const fileData = await buildProductBalanceWorkbook({
        lineName: selectedLine.name,
        summary,
        items,
      })
      const safeLine = selectedLine.name.replace(/[^\w.-]+/g, "_")
      const dateStamp = new Date().toISOString().slice(0, 10)
      downloadBlob(fileData, `mahsulot_balansi_${safeLine}_${dateStamp}.xlsx`)
      ShowOKToast("XLSX fayl yuklandi")
    } catch {
      ShowErrorToast("Exportda xatolik")
    } finally {
      setExporting(false)
    }
  }

  return (
    <PageContainer
      title="Mahsulot balansi"
      description="Boshlang'ich yig'uv, T1–T3 va Yakuniy yig'uv uchastkalaridagi faol serial (WIP)"
      fullWidth
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Button
            type="button"
            variant="outline"
            className="h-14 gap-2 rounded-2xl px-5"
            onClick={() => void exportToExcel()}
            disabled={exporting || balanceLoading || linesLoading || totalCount === 0}
          >
            <FileDown className="size-5" />
            {exporting ? "Yuklanmoqda..." : "Excel export"}
          </Button>
          <Button type="button" className="h-14 rounded-2xl px-5" onClick={openReport}>
            O&apos;tkazishlar hisoboti
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
        title="Jami"
        action={
          <span className="text-3xl font-bold tabular-nums text-primary">
            {balanceLoading || linesLoading ? "…" : totalCount.toLocaleString()}
          </span>
        }
      >
        <p className="text-sm text-muted-foreground">
          {selectedLine
            ? `${selectedLine.name} — batafsil seriallar Excel export orqali`
            : "Liniyani tanlang"}
        </p>
      </Panel>

      <Panel title="Model bo'yicha" noPadding className="mt-4">
        <div className="overflow-x-auto border-t border-border/60">
          <Table>
            <TableHeader>
              <TableRow className="border-b border-border/60 bg-muted/40 hover:bg-muted/40">
                <TableHead className="px-4 py-3 text-sm font-semibold">Modeli</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">Model</TableHead>
                <TableHead className="px-4 py-3 text-sm font-semibold">ODOO code</TableHead>
                <TableHead className="px-4 py-3 text-right text-sm font-semibold">Soni</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {summary.map((row, index) => (
                <TableRow
                  key={row.model_id}
                  className={cn("border-border/40", index % 2 === 1 && "bg-muted/20")}
                >
                  <TableCell className="px-4 py-3 text-sm">{row.modeli || "—"}</TableCell>
                  <TableCell className="px-4 py-3 text-sm">{row.model_name || "—"}</TableCell>
                  <TableCell className="px-4 py-3 text-sm">{row.odoo_code || "—"}</TableCell>
                  <TableCell className="px-4 py-3 text-right text-sm font-semibold tabular-nums">
                    {row.count}
                  </TableCell>
                </TableRow>
              ))}
              {!balanceLoading && !linesLoading && summary.length === 0 ? (
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
