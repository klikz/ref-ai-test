import type { ReactNode } from "react"
import { cn } from "@/lib/utils"

type PanelProps = {
  children: ReactNode
  className?: string
  title?: string
  description?: string
  action?: ReactNode
  noPadding?: boolean
  bodyClassName?: string
}

export function Panel({
  children,
  className,
  title,
  description,
  action,
  noPadding,
  bodyClassName,
}: PanelProps) {
  return (
    <section
      className={cn(
        "min-h-0 rounded-xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-sm",
        "ring-1 ring-black/[0.03] dark:ring-white/[0.06]",
        className,
      )}
    >
      {(title || action) && (
        <div className="flex shrink-0 items-start justify-between gap-3 border-b border-border/50 px-4 py-2.5">
          <div>
            {title ? <h2 className="text-sm font-semibold text-foreground sm:text-base">{title}</h2> : null}
            {description ? (
              <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
            ) : null}
          </div>
          {action}
        </div>
      )}
      <div className={cn("min-h-0", !noPadding && "p-3 sm:p-4", bodyClassName)}>{children}</div>
    </section>
  )
}
