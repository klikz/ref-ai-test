import type { CSSProperties, ReactNode } from "react"

import { TABLE_HEADER_HEIGHT } from "@/hooks/use-window-table-virtualizer"
import { cn } from "@/lib/utils"

type VirtualTableHeaderProps = {
  children: ReactNode
  gridTemplateColumns: string
  className?: string
}

export function VirtualTableHeader({
  children,
  gridTemplateColumns,
  className,
}: VirtualTableHeaderProps) {
  return (
    <div
      className={cn(
        "sticky z-20 grid w-full items-center border-b border-border/60 bg-muted text-muted-foreground",
        className,
      )}
      style={{ gridTemplateColumns, height: TABLE_HEADER_HEIGHT }}
    >
      {children}
    </div>
  )
}

type VirtualTableHeaderCellProps = {
  children: ReactNode
  className?: string
  onClick?: (event: unknown) => void
}

export function VirtualTableHeaderCell({
  children,
  className,
  onClick,
}: VirtualTableHeaderCellProps) {
  return (
    <div
      role={onClick ? "button" : undefined}
      tabIndex={onClick ? 0 : undefined}
      onClick={onClick}
      onKeyDown={
        onClick
          ? (event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault()
                onClick(event)
              }
            }
          : undefined
      }
      className={cn(
        "truncate px-2 text-xs font-semibold sm:text-sm",
        onClick && "cursor-pointer select-none",
        className,
      )}
    >
      {children}
    </div>
  )
}

type VirtualTableRowProps = {
  children: ReactNode
  gridTemplateColumns: string
  className?: string
  style?: CSSProperties
  onClick?: () => void
}

export function VirtualTableRow({
  children,
  gridTemplateColumns,
  className,
  style,
  onClick,
}: VirtualTableRowProps) {
  return (
    <div
      className={cn(
        "absolute left-0 grid w-full items-center border-b border-border/40 bg-background",
        onClick && "cursor-pointer hover:bg-primary/5",
        className,
      )}
      style={{ gridTemplateColumns, ...style }}
      onClick={onClick}
    >
      {children}
    </div>
  )
}

export function VirtualTableCell({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("truncate px-2 py-2 text-sm", className)}>{children}</div>
}
