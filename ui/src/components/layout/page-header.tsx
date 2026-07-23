import type { ReactNode } from "react"
import { cn } from "@/lib/utils"

type PageHeaderProps = {
  title: string
  titleAddon?: ReactNode
  description?: string
  center?: ReactNode
  actions?: ReactNode
  className?: string
}

export function PageHeader({
  title,
  titleAddon,
  description,
  center,
  actions,
  className,
}: PageHeaderProps) {
  if (!center) {
    return (
      <div className={cn("flex shrink-0 items-center justify-between gap-3", className)}>
        <div className="flex min-w-0 flex-1 items-baseline gap-2 overflow-hidden">
          <h1 className="shrink-0 truncate text-xl font-semibold tracking-tight text-foreground sm:text-2xl">
            {title}
          </h1>
          {description ? (
            <>
              <span className="hidden shrink-0 text-muted-foreground sm:inline" aria-hidden>
                —
              </span>
              <p className="min-w-0 truncate text-xs text-muted-foreground sm:text-sm">{description}</p>
            </>
          ) : null}
        </div>
        {titleAddon || actions ? (
          <div className="flex shrink-0 items-center justify-end gap-2">
            {titleAddon}
            {actions}
          </div>
        ) : null}
      </div>
    )
  }

  return (
    <div
      className={cn(
        "flex shrink-0 flex-col gap-3 lg:flex-row lg:items-center lg:gap-4",
        className,
      )}
    >
      <div className="shrink-0 space-y-1">
        <h1 className="text-xl font-semibold tracking-tight text-foreground sm:text-2xl">
          {title}
        </h1>
        {description ? (
          <p className="max-w-2xl text-xs text-muted-foreground sm:text-sm">{description}</p>
        ) : null}
      </div>
      <div className="flex min-w-0 flex-1 items-center justify-center lg:px-4">{center}</div>
      {titleAddon || actions ? (
        <div className="flex min-w-0 max-w-full items-center justify-end gap-2 overflow-x-auto [scrollbar-width:thin]">
          {titleAddon}
          {actions}
        </div>
      ) : null}
    </div>
  )
}
