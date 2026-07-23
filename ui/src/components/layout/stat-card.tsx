import type { LucideIcon } from "lucide-react"
import { cn } from "@/lib/utils"

type StatCardProps = {
  label: string
  value: string | number
  icon?: LucideIcon
  trend?: string
  variant?: "default" | "primary" | "accent"
  className?: string
  large?: boolean
}

const variants = {
  default: "from-card to-muted/30 border-border/60",
  primary: "from-primary/15 via-primary/5 to-transparent border-primary/20",
  accent: "from-chart-2/20 via-chart-2/5 to-transparent border-chart-2/25",
}

export function StatCard({
  label,
  value,
  icon: Icon,
  trend,
  variant = "default",
  className,
  large,
}: StatCardProps) {
  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-2xl border bg-gradient-to-br p-5 shadow-sm",
        variants[variant],
        className,
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="space-y-2">
          <p className="text-sm font-medium text-muted-foreground">{label}</p>
          <p
            className={cn(
              "font-semibold tracking-tight text-foreground tabular-nums",
              large ? "text-5xl sm:text-6xl lg:text-7xl" : "text-3xl sm:text-4xl",
            )}
          >
            {value}
          </p>
          {trend ? <p className="text-xs text-muted-foreground">{trend}</p> : null}
        </div>
        {Icon ? (
          <div className="rounded-xl bg-background/60 p-2.5 shadow-sm ring-1 ring-border/50">
            <Icon className="size-5 text-primary" />
          </div>
        ) : null}
      </div>
      <div className="pointer-events-none absolute -right-8 -top-8 size-32 rounded-full bg-primary/5 blur-2xl" />
    </div>
  )
}
