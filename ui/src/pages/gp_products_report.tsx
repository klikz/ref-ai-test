import { useEffect, useMemo, useState } from "react"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ShowErrorToast } from "@/components/showToast"

type GPBalanceSummaryRow = {
  model_id: number
  modeli: string
  model_name: string
  count: number
}

type GPTransactionRow = {
  id: number
  serial: string
  modeli: string
  model_name: string
  user_name: string
  received_at: string
}

type GPReportResponse = {
  summary: GPBalanceSummaryRow[]
  items: GPTransactionRow[]
}

function todayDate() {
  return new Date().toISOString().slice(0, 10)
}

function applyDatePreset(type: "month" | "year") {
  const now = new Date()
  if (type === "year") {
    const start = new Date(now.getFullYear(), 0, 1)
    const month = String(start.getMonth() + 1).padStart(2, "0")
    const day = String(start.getDate()).padStart(2, "0")
    return {
      dateFrom: `${start.getFullYear()}-${month}-${day}`,
      dateTo: todayDate(),
    }
  }
  const start = new Date(now.getFullYear(), now.getMonth(), 1)
  const month = String(start.getMonth() + 1).padStart(2, "0")
  const day = String(start.getDate()).padStart(2, "0")
  return {
    dateFrom: `${start.getFullYear()}-${month}-${day}`,
    dateTo: todayDate(),
  }
}

export default function GPProductsReportPage() {
  const [dateFrom, setDateFrom] = useState(todayDate())
  const [dateTo, setDateTo] = useState(todayDate())
  const [loading, setLoading] = useState(false)
  const [summary, setSummary] = useState<GPBalanceSummaryRow[]>([])
  const [items, setItems] = useState<GPTransactionRow[]>([])

  async function loadReport() {
    setLoading(true)
    const result = await Backend_Request<GPReportResponse>(
      { date_from: dateFrom, date_to: dateTo, limit: 500 },
      "/api/ware/gp-products/report",
    )
    setLoading(false)

    if (result.result !== "ok" || !result.data) {
      setSummary([])
      setItems([])
      ShowErrorToast(result.error || "Hisobotni yuklab bo'lmadi")
      return
    }

    setSummary(result.data.summary ?? [])
    setItems(result.data.items ?? [])
  }

  useEffect(() => {
    void loadReport()
  }, [])

  const totalBalance = useMemo(
    () => summary.reduce((acc, row) => acc + Number(row.count || 0), 0),
    [summary],
  )

  return (
    <PageContainer
      title="GP Mahsulot hisoboti"
      description="Ombordagi GP mahsulot balansi va davr bo'yicha qabul tranzaksiyalari"
      fullWidth
    >
      <Panel>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-[220px_220px_auto] lg:items-end">
          <div className="space-y-1">
            <Label htmlFor="gp-date-from">Sana dan</Label>
            <Input id="gp-date-from" type="date" value={dateFrom} onChange={(e) => setDateFrom(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="gp-date-to">Sana gacha</Label>
            <Input id="gp-date-to" type="date" value={dateTo} onChange={(e) => setDateTo(e.target.value)} />
          </div>
          <Button type="button" className="h-10" disabled={loading} onClick={() => void loadReport()}>
            {loading ? "Yuklanmoqda..." : "Ko'rish"}
          </Button>
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          <Button
            type="button"
            variant="secondary"
            size="sm"
            className="rounded-full"
            disabled={loading}
            onClick={() => {
              const next = applyDatePreset("month")
              setDateFrom(next.dateFrom)
              setDateTo(next.dateTo)
            }}
          >
            Oy boshidan
          </Button>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            className="rounded-full"
            disabled={loading}
            onClick={() => {
              const next = applyDatePreset("year")
              setDateFrom(next.dateFrom)
              setDateTo(next.dateTo)
            }}
          >
            Yil boshidan
          </Button>
        </div>
      </Panel>

      <Panel title={`Ombor GP balans (${totalBalance} ta)`}>
        {summary.length === 0 ? (
          <p className="text-sm text-muted-foreground">Balans bo'sh.</p>
        ) : (
          <div className="space-y-2">
            {summary.map((row) => (
              <div
                key={row.model_id}
                className="grid gap-1 rounded-lg border border-border/70 bg-muted/20 px-3 py-2 sm:grid-cols-[1fr_auto]"
              >
                <div className="min-w-0">
                  <p className="truncate font-medium">{row.modeli || "—"}</p>
                  <p className="truncate text-xs text-muted-foreground">{row.model_name || "—"}</p>
                </div>
                <p className="text-right text-sm font-semibold">{row.count} ta</p>
              </div>
            ))}
          </div>
        )}
      </Panel>

      <Panel title={`Qabul tranzaksiyalari (${items.length} ta)`}>
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">Tanlangan davrda tranzaksiya yo'q.</p>
        ) : (
          <div className="space-y-2">
            {items.map((row) => (
              <div
                key={row.id}
                className="grid gap-1 rounded-lg border border-border/70 bg-muted/20 px-3 py-2 text-sm sm:grid-cols-[1fr_auto]"
              >
                <div className="min-w-0">
                  <p className="truncate font-mono text-foreground">{row.serial}</p>
                  <p className="truncate text-xs text-muted-foreground">
                    {row.modeli || "—"} {row.model_name ? `· ${row.model_name}` : ""}
                  </p>
                </div>
                <div className="text-right text-xs text-muted-foreground">
                  <p>{row.received_at || "—"}</p>
                  <p>{row.user_name || "—"}</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </Panel>
    </PageContainer>
  )
}
