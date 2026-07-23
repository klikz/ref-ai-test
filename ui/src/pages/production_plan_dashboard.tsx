import { useCallback, useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { Backend_Request } from "@/services/backend"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ShowErrorToast } from "@/components/showToast"
import { cn } from "@/lib/utils"
import { todayDateInput, type PlanDashboardResponse } from "./production_plan_shared"

export default function ProductionPlanDashboardPage() {
  const [planDate, setPlanDate] = useState(todayDateInput())
  const [data, setData] = useState<PlanDashboardResponse | null>(null)
  const [loading, setLoading] = useState(false)

  const loadDashboard = useCallback(
    async (showLoading = false) => {
      if (showLoading) {
        setLoading(true)
      }
      const result = await Backend_Request<PlanDashboardResponse>(
        { plan_date: planDate },
        "/api/production/plan/dashboard",
      )
      if (showLoading) {
        setLoading(false)
      }
      if (result.result === "ok" && result.data) {
        setData(result.data)
      } else if (showLoading) {
        setData(null)
        ShowErrorToast(result.error || "Dashboard yuklanmadi")
      }
    },
    [planDate],
  )

  useEffect(() => {
    void loadDashboard(true)
    const interval = setInterval(() => {
      void loadDashboard(false)
    }, 2000)
    return () => clearInterval(interval)
  }, [loadDashboard])

  const totalPlanned = data?.lines.reduce((s, l) => s + l.planned_total, 0) ?? 0
  const totalActual = data?.lines.reduce((s, l) => s + l.actual_total, 0) ?? 0
  const totalPct = totalPlanned > 0 ? Math.round((totalActual * 100) / totalPlanned) : 0

  return (
    <PageContainer
      title="Reja dashboard"
      description="Kunlik ishlab chiqarish rejasi va bajarilish"
      fullWidth
      actions={
        <>
          <div className="flex items-center gap-2">
            <Label htmlFor="dash-date" className="shrink-0 text-sm text-muted-foreground">
              Sana
            </Label>
            <Input
              id="dash-date"
              type="date"
              value={planDate}
              onChange={(e) => setPlanDate(e.target.value)}
              className="h-9 w-[10.5rem]"
            />
          </div>
          <Button variant="outline" asChild>
            <Link to="/production/plan">Reja boshqaruvi</Link>
          </Button>
        </>
      }
    >
      <div className="grid gap-4 md:grid-cols-3">
        <Panel
          title="Jami reja"
          className="bg-gradient-to-br from-slate-900 to-slate-800 text-white [&_h2]:text-white [&_.border-b]:border-white/20"
        >
          <p className="text-4xl font-bold tabular-nums">{loading ? "…" : totalPlanned.toLocaleString()}</p>
        </Panel>
        <Panel
          title="Jami fakt"
          className="bg-gradient-to-br from-cyan-700 to-cyan-900 text-white [&_h2]:text-white [&_.border-b]:border-white/20"
        >
          <p className="text-4xl font-bold tabular-nums">{loading ? "…" : totalActual.toLocaleString()}</p>
        </Panel>
        <Panel
          title="Bajarilish"
          className="bg-gradient-to-br from-emerald-700 to-emerald-900 text-white [&_h2]:text-white [&_.border-b]:border-white/20"
        >
          <p className="text-4xl font-bold tabular-nums">{loading ? "…" : `${totalPct}%`}</p>
        </Panel>
      </div>

      <div className="mt-4 grid gap-4 lg:grid-cols-2">
        {data?.lines.map((line) => (
          <Panel key={line.line_id} title={line.line_name}>
            <div className="mb-2 flex justify-between text-sm text-muted-foreground">
              <span>
                {line.actual_total} / {line.planned_total}
              </span>
              <span>{line.completion_pct}%</span>
            </div>
            <div className="mb-3 h-3 overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-cyan-600 transition-all"
                style={{ width: `${Math.min(100, line.completion_pct)}%` }}
              />
            </div>
            <div className="mb-2 grid grid-cols-2 gap-2 text-xs text-muted-foreground">
              <div className="rounded-lg border border-border/50 bg-muted/30 px-2 py-1.5">
                <p className="font-medium text-foreground">1-sm</p>
                <p className="tabular-nums">
                  {line.shift1_actual ?? 0} / {line.shift1_planned ?? 0}
                </p>
              </div>
              <div className="rounded-lg border border-border/50 bg-muted/30 px-2 py-1.5">
                <p className="font-medium text-foreground">2-sm</p>
                <p className="tabular-nums">
                  {line.shift2_actual ?? 0} / {line.shift2_planned ?? 0}
                </p>
              </div>
            </div>
            <p className="text-sm text-muted-foreground">
              {line.models_complete} / {line.models_in_plan} pozitsiya bajarildi
              {data?.current_shift_no
                ? ` · joriy: ${data.current_shift_no}-sm`
                : ""}
            </p>
          </Panel>
        ))}
      </div>

      <Panel title="Batafsil" className="mt-4" noPadding>
        <div className="max-h-[28rem] overflow-auto">
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-muted/90">
              <tr className="border-b">
                <th className="px-4 py-2 text-left">Liniya</th>
                <th className="px-4 py-2 text-left">Pozitsiya</th>
                <th className="px-4 py-2 text-left">Smena</th>
                <th className="px-4 py-2 text-right">Reja</th>
                <th className="px-4 py-2 text-right">Fakt</th>
                <th className="px-4 py-2 text-right">%</th>
              </tr>
            </thead>
            <tbody>
              {data?.items.map((row) => {
                const line = data.lines.find((l) => l.line_id === row.line_id)
                return (
                  <tr key={row.id} className="border-b border-border/40">
                    <td className="px-4 py-2">{line?.line_name ?? row.line_id}</td>
                    <td className="px-4 py-2 font-medium">{row.label}</td>
                    <td className="px-4 py-2 tabular-nums">{row.shift_no || 1}-sm</td>
                    <td className="px-4 py-2 text-right tabular-nums">{row.planned_qty}</td>
                    <td className="px-4 py-2 text-right tabular-nums">{row.actual_qty}</td>
                    <td
                      className={cn(
                        "px-4 py-2 text-right tabular-nums",
                        !row.allow_overplan && row.actual_qty > row.planned_qty && "text-destructive font-semibold",
                      )}
                    >
                      {row.completion_pct}%
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </Panel>
    </PageContainer>
  )
}
