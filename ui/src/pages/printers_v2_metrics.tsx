import { useCallback, useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { ArrowLeft, ChevronDown, ChevronRight, Loader2, RefreshCcw, Trash2 } from "lucide-react"
import { toast } from "sonner"

import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

type PrintV2Metrics = {
  success: number
  fail: number
  total_ms: number
  last_ms: number
  in_flight: number
  date_from?: string
  date_to?: string
  lines?: PrintV2LineMetrics[]
}

type PrintV2LineMetrics = {
  line_id: number
  line_name: string
  success: number
  fail: number
  total_ms: number
  last_ms: number
}

type PrintV2ErrorEvent = {
  id: number
  created_at: string
  ok: boolean
  duration_ms: number
  line_id: number
  line_name: string
  printer_v2_id: number
  printer_name: string
  template_id: number
  print_language: string
  effective_language: string
  serial: string
  stage: string
  error_message: string
  error_detail: string
  meta?: Record<string, unknown>
}

function todayISO(): string {
  const d = new Date()
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, "0")
  const day = String(d.getDate()).padStart(2, "0")
  return `${y}-${m}-${day}`
}

function avgMs(m: Pick<PrintV2Metrics, "success" | "fail" | "total_ms"> | null): number | null {
  if (!m) return null
  const n = m.success + m.fail
  if (n <= 0) return null
  return Math.round(m.total_ms / n)
}

function failRate(m: Pick<PrintV2Metrics, "success" | "fail"> | null): string {
  if (!m) return "—"
  const n = m.success + m.fail
  if (n <= 0) return "—"
  return `${((m.fail / n) * 100).toFixed(1)}%`
}

function formatTs(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString()
}

function metricCards(
  m: Pick<PrintV2Metrics, "success" | "fail" | "total_ms" | "last_ms"> | null,
  opts?: { inFlight?: number | null },
) {
  const cards: { label: string; value: string | number; hint: string }[] = [
    { label: "Success", value: m?.success ?? "—", hint: "Muvaffaqiyatli chop" },
    { label: "Fail", value: m?.fail ?? "—", hint: "Xato bilan tugagan" },
  ]
  if (opts && "inFlight" in opts) {
    cards.push({ label: "In flight", value: opts.inFlight ?? "—", hint: "Hozir bajarilayotgan" })
  }
  cards.push(
    { label: "Last ms", value: m?.last_ms ?? "—", hint: "Oxirgi job davomiyligi" },
    { label: "Avg ms", value: avgMs(m) ?? "—", hint: "O‘rtacha davomiylik" },
    { label: "Fail rate", value: failRate(m), hint: "Xato foizi" },
  )
  return cards
}

export default function PrintersV2MetricsPage() {
  const [dateFrom, setDateFrom] = useState(todayISO)
  const [dateTo, setDateTo] = useState(todayISO)
  const [metrics, setMetrics] = useState<PrintV2Metrics | null>(null)
  const [errors, setErrors] = useState<PrintV2ErrorEvent[]>([])
  const [loading, setLoading] = useState(false)
  const [resetting, setResetting] = useState(false)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [expandedId, setExpandedId] = useState<number | null>(null)

  const rangeBody = useMemo(
    () => ({ date_from: dateFrom, date_to: dateTo }),
    [dateFrom, dateTo],
  )

  const loadAll = useCallback(async () => {
    setLoading(true)
    const [metricsRes, errorsRes] = await Promise.all([
      Backend_Request<PrintV2Metrics>(rangeBody, "/api/tech/printers-v2/metrics"),
      Backend_Request<{ items: PrintV2ErrorEvent[] }>(
        { ...rangeBody, limit: 100, offset: 0 },
        "/api/tech/printers-v2/metrics/errors",
      ),
    ])
    setLoading(false)

    if (metricsRes.result === "ok" && metricsRes.data) {
      setMetrics(metricsRes.data)
    } else {
      toast.error(metricsRes.error || "Metrics yuklanmadi")
    }

    if (errorsRes.result === "ok" && errorsRes.data) {
      setErrors(errorsRes.data.items || [])
    } else if (errorsRes.result === "error") {
      toast.error(errorsRes.error || "Xatolar yuklanmadi")
    }
  }, [rangeBody])

  useEffect(() => {
    void loadAll()
  }, [loadAll])

  useEffect(() => {
    if (!autoRefresh) return
    const id = window.setInterval(() => {
      void loadAll()
    }, 5000)
    return () => window.clearInterval(id)
  }, [autoRefresh, loadAll])

  const onReset = async () => {
    const ok = window.confirm(
      `${dateFrom} — ${dateTo} oraligidagi print eventlarni o‘chirishni tasdiqlaysizmi?`,
    )
    if (!ok) return
    setResetting(true)
    const result = await Backend_Request<{ deleted: number }>(
      rangeBody,
      "/api/tech/printers-v2/metrics/reset",
    )
    setResetting(false)
    if (result.result === "ok") {
      toast.success(`O‘chirildi: ${result.data?.deleted ?? 0}`)
      setExpandedId(null)
      void loadAll()
    } else {
      toast.error(result.error || "Reset muvaffaqiyatsiz")
    }
  }

  const lines = metrics?.lines ?? []
  const totalCards = metricCards(metrics, { inFlight: metrics?.in_flight ?? null })

  return (
    <PageContainer
      title="Print metrics (V2)"
      description="DB da saqlanadigan V2 chop statistikasi. Sana bo‘yicha filtrlang, liniya ko‘rsatkichlari va xatolarni to‘liq o‘qing."
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" asChild>
            <Link to="/printers-v2" className="gap-2">
              <ArrowLeft className="size-4" />
              Printerlar V2
            </Link>
          </Button>
          <Button variant="outline" className="gap-2" onClick={() => void loadAll()} disabled={loading}>
            {loading ? <Loader2 className="size-4 animate-spin" /> : <RefreshCcw className="size-4" />}
            Yangilash
          </Button>
          <Button
            variant="destructive"
            className="gap-2"
            onClick={() => void onReset()}
            disabled={resetting || loading}
          >
            {resetting ? <Loader2 className="size-4 animate-spin" /> : <Trash2 className="size-4" />}
            Reset
          </Button>
        </div>
      }
    >
      <Panel title="Filtr" className="mb-4">
        <div className="flex flex-wrap items-end gap-3">
          <label className="grid gap-1 text-sm">
            <span className="text-muted-foreground">Dan</span>
            <Input type="date" value={dateFrom} onChange={(e) => setDateFrom(e.target.value)} className="w-44" />
          </label>
          <label className="grid gap-1 text-sm">
            <span className="text-muted-foreground">Gacha</span>
            <Input type="date" value={dateTo} onChange={(e) => setDateTo(e.target.value)} className="w-44" />
          </label>
          <label className="mb-2 flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
            />
            Avto yangilash (5 s)
          </label>
        </div>
      </Panel>

      <Panel title="Ko‘rsatkichlar — Jami" className="mb-4">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {totalCards.map((c) => (
            <div key={c.label} className="rounded-lg border border-border bg-background px-4 py-3">
              <div className="text-xs text-muted-foreground">{c.label}</div>
              <div className="mt-1 text-2xl font-semibold tabular-nums">{c.value}</div>
              <div className="mt-1 text-[11px] text-muted-foreground">{c.hint}</div>
            </div>
          ))}
        </div>
      </Panel>

      {lines.length === 0 ? (
        <Panel title="Liniyalar" className="mb-4">
          <p className="text-sm text-muted-foreground">Liniya ko‘rsatkichlari yuklanmadi.</p>
        </Panel>
      ) : (
        lines.map((line) => {
          const cards = metricCards(line)
          return (
            <Panel
              key={line.line_name || line.line_id}
              title={`Ko‘rsatkichlar — ${line.line_name}`}
              className="mb-4"
            >
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {cards.map((c) => (
                  <div key={c.label} className="rounded-lg border border-border bg-background px-4 py-3">
                    <div className="text-xs text-muted-foreground">{c.label}</div>
                    <div className="mt-1 text-2xl font-semibold tabular-nums">{c.value}</div>
                    <div className="mt-1 text-[11px] text-muted-foreground">{c.hint}</div>
                  </div>
                ))}
              </div>
            </Panel>
          )
        })
      )}

      <Panel title={`Xatolar (${errors.length})`} className="mb-4">
        {errors.length === 0 ? (
          <p className="text-sm text-muted-foreground">Tanlangan oralikda xato yo‘q.</p>
        ) : (
          <div className="space-y-2">
            {errors.map((ev) => {
              const open = expandedId === ev.id
              return (
                <div key={ev.id} className="rounded-lg border border-border">
                  <button
                    type="button"
                    className="flex w-full items-start gap-2 px-3 py-2 text-left hover:bg-muted/40"
                    onClick={() => setExpandedId(open ? null : ev.id)}
                  >
                    {open ? (
                      <ChevronDown className="mt-0.5 size-4 shrink-0" />
                    ) : (
                      <ChevronRight className="mt-0.5 size-4 shrink-0" />
                    )}
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                        <span>{formatTs(ev.created_at)}</span>
                        <span className="font-medium text-foreground">
                          {ev.line_name || (ev.line_id ? `Liniya #${ev.line_id}` : "—")}
                        </span>
                        <span>{ev.printer_name || "—"}</span>
                        <span>stage: {ev.stage || "—"}</span>
                        <span>{ev.duration_ms} ms</span>
                        {ev.serial ? <span>serial: {ev.serial}</span> : null}
                      </div>
                      <div className="mt-1 truncate text-sm">{ev.error_message || ev.error_detail}</div>
                    </div>
                  </button>
                  {open ? (
                    <div className="border-t border-border bg-muted/20 px-3 py-3 text-sm">
                      <div className="mb-2 grid gap-1 text-xs text-muted-foreground sm:grid-cols-2">
                        <div>line_id: {ev.line_id || "—"}</div>
                        <div>line_name: {ev.line_name || "—"}</div>
                        <div>printer_v2_id: {ev.printer_v2_id || "—"}</div>
                        <div>template_id: {ev.template_id || "—"}</div>
                        <div>print_language: {ev.print_language || "—"}</div>
                        <div>effective_language: {ev.effective_language || "—"}</div>
                      </div>
                      <div className="mb-1 text-xs font-medium text-muted-foreground">To‘liq xato</div>
                      <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-words rounded-md border border-border bg-background p-3 text-xs">
                        {ev.error_detail || ev.error_message || "—"}
                      </pre>
                      {ev.meta ? (
                        <>
                          <div className="mb-1 mt-3 text-xs font-medium text-muted-foreground">Meta</div>
                          <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-words rounded-md border border-border bg-background p-3 text-xs">
                            {JSON.stringify(ev.meta, null, 2)}
                          </pre>
                        </>
                      ) : null}
                    </div>
                  ) : null}
                </div>
              )
            })}
          </div>
        )}
      </Panel>
    </PageContainer>
  )
}
