import { cn } from "@/lib/utils"

type PlanRowProgressProps = {
  planned: number
  actual: number
  allowOverplan?: boolean
  className?: string
}

export function PlanRowProgress({ planned, actual, allowOverplan, className }: PlanRowProgressProps) {
  const pct = planned > 0 ? Math.min(100, Math.round((actual / planned) * 100)) : 0
  const over = planned > 0 && actual > planned

  return (
    <div className={cn("w-full min-w-[9rem] max-w-[12rem]", className)}>
      <div className="mb-1.5 flex items-baseline justify-between gap-2 tabular-nums">
        <span
          className={cn(
            "text-sm font-bold",
            over && !allowOverplan && "text-destructive",
            over && allowOverplan && "text-amber-700 dark:text-amber-300",
            !over && pct >= 100 && "text-emerald-700 dark:text-emerald-300",
          )}
        >
          {actual}/{planned}
        </span>
        <span className="text-xs font-medium text-muted-foreground">{pct}%</span>
      </div>
      <div className="h-2.5 overflow-hidden rounded-full bg-muted/80">
        <div
          className={cn(
            "h-full rounded-full transition-all",
            over && allowOverplan ? "bg-amber-500" : pct >= 100 ? "bg-emerald-500" : "bg-cyan-600",
          )}
          style={{ width: `${Math.min(100, pct)}%` }}
        />
      </div>
    </div>
  )
}

export function PlanTotalsBadge({ actual, planned }: { actual: number; planned: number }) {
  if (planned <= 0) {
    return null
  }
  return (
    <div className="rounded-full border border-cyan-500/30 bg-cyan-50/80 px-3 py-1.5 text-sm font-semibold tabular-nums text-cyan-900 dark:bg-cyan-950/40 dark:text-cyan-100">
      Jami: {actual}/{planned}
    </div>
  )
}