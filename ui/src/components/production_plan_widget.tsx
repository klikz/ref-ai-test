import { useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { cn } from "@/lib/utils"
import { fetchPlanDay, type CurrentShiftInfo, type PlanItemRow } from "@/pages/production_plan_shared"

type Props = {
  lineId: number
  lineLabel?: string
  className?: string
  refreshKey?: number
  hideHeader?: boolean
  /** Chop etish sahifalari uchun ixcham ko'rinish (planshet) */
  compact?: boolean
}

export function ProductionPlanWidget({
  lineId,
  lineLabel,
  className,
  refreshKey = 0,
  hideHeader = false,
  compact = false,
}: Props) {
  const [items, setItems] = useState<PlanItemRow[]>([])
  const [status, setStatus] = useState("")
  const [currentShift, setCurrentShift] = useState<CurrentShiftInfo | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    void (async () => {
      setLoading(true)
      const result = await fetchPlanDay(lineId, undefined, { currentShift: true })
      setLoading(false)
      if (result.result === "ok" && result.data) {
        setItems(result.data.items ?? [])
        setStatus(result.data.status ?? "")
        setCurrentShift(result.data.current_shift ?? null)
      } else {
        setItems([])
        setStatus("")
        setCurrentShift(null)
      }
    })()
  }, [lineId, refreshKey])

  const shiftLabel =
    currentShift?.label ||
    (currentShift?.shift_no ? `${currentShift.shift_no}-sm smena` : "")

  if (loading) {
    return (
      <div className={cn("rounded-2xl border border-border/60 bg-muted/30 px-4 py-3 text-sm text-muted-foreground", className)}>
        Reja yuklanmoqda...
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div className={cn("rounded-2xl border border-amber-500/30 bg-amber-50/80 px-4 py-3 text-sm dark:bg-amber-950/30", className)}>
        <span className="font-medium text-amber-800 dark:text-amber-200">
          {lineLabel ? `${lineLabel}: ` : ""}
          {shiftLabel ? `${shiftLabel}: ` : ""}
        </span>
        Joriy smena rejasi yo&apos;q yoki tasdiqlanmagan.
        <Link to="/production/plan" className="ml-2 text-primary underline-offset-2 hover:underline">
          Rejani ochish
        </Link>
      </div>
    )
  }

  if (compact) {
    const manyItems = items.length > 4

    return (
      <div
        className={cn(
          "w-full rounded-xl border border-cyan-500/25 bg-cyan-50/60 px-3 py-2.5 dark:bg-cyan-950/20",
          className,
        )}
      >
        {shiftLabel ? (
          <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-cyan-800 dark:text-cyan-200">
            {shiftLabel}
            {currentShift?.start_time && currentShift?.end_time
              ? ` · ${currentShift.start_time}–${currentShift.end_time}`
              : ""}
          </p>
        ) : null}
        <div
          className={cn(
            "grid w-full gap-2",
            manyItems
              ? "grid-flow-col auto-cols-[minmax(10.5rem,1fr)] overflow-x-auto pb-0.5 [scrollbar-width:thin]"
              : "min-w-0",
          )}
          style={
            manyItems ? undefined : { gridTemplateColumns: `repeat(${items.length}, minmax(0, 1fr))` }
          }
        >
          {items.map((row) => {
            const pct = row.planned_qty > 0 ? Math.min(100, row.completion_pct) : 0
            const over = row.actual_qty > row.planned_qty
            return (
              <div
                key={`${row.item_key}-${row.shift_no}-${row.label}`}
                className="min-w-0 rounded-lg border border-border/50 bg-background/70 px-3 py-2"
              >
                <div className="mb-1 flex items-baseline justify-between gap-2">
                  <span className="min-w-0 truncate text-xs font-medium sm:text-sm" title={row.label}>
                    {row.label}
                  </span>
                  <span className="shrink-0 tabular-nums text-[11px] text-muted-foreground sm:text-xs">
                    {row.actual_qty}/{row.planned_qty}
                    {!row.allow_overplan && over ? <span className="ml-0.5 text-destructive">!</span> : null}
                  </span>
                </div>
                <div className="h-2 overflow-hidden rounded-full bg-muted">
                  <div
                    className={cn(
                      "h-full rounded-full transition-all",
                      over && row.allow_overplan ? "bg-amber-500" : pct >= 100 ? "bg-emerald-500" : "bg-cyan-600",
                    )}
                    style={{ width: `${Math.min(100, pct)}%` }}
                  />
                </div>
              </div>
            )
          })}
        </div>
      </div>
    )
  }

  return (
    <div className={cn("rounded-2xl border border-cyan-500/25 bg-cyan-50/60 p-4 dark:bg-cyan-950/20", className)}>
      {!hideHeader ? (
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h3 className="text-sm font-semibold">
            {shiftLabel || "Joriy smena"}
            {lineLabel ? ` — ${lineLabel}` : ""}
          </h3>
          <span
            className={cn(
              "rounded-full px-2.5 py-0.5 text-xs font-medium",
              status === "locked"
                ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-200"
                : "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-200",
            )}
          >
            {status === "locked" ? "Tasdiqlangan" : "Qoralama"}
          </span>
        </div>
      ) : shiftLabel ? (
        <p className="mb-2 text-xs font-semibold text-cyan-800 dark:text-cyan-200">{shiftLabel}</p>
      ) : null}
      <div className="space-y-2">
        {items.map((row) => {
          const pct = row.planned_qty > 0 ? Math.min(100, row.completion_pct) : 0
          const over = row.actual_qty > row.planned_qty
          return (
            <div key={`${row.item_key}-${row.shift_no}-${row.label}`}>
              <div className="mb-1 flex justify-between text-sm">
                <span className="font-medium">{row.label}</span>
                <span className="tabular-nums text-muted-foreground">
                  {row.actual_qty} / {row.planned_qty}
                  {!row.allow_overplan && over ? (
                    <span className="ml-1 text-destructive">!</span>
                  ) : null}
                </span>
              </div>
              <div className="h-2 overflow-hidden rounded-full bg-muted">
                <div
                  className={cn(
                    "h-full rounded-full transition-all",
                    over && row.allow_overplan ? "bg-amber-500" : pct >= 100 ? "bg-emerald-500" : "bg-cyan-600",
                  )}
                  style={{ width: `${Math.min(100, pct)}%` }}
                />
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
