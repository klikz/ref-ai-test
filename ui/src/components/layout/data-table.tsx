import { flexRender, type Table as TanstackTable } from "@tanstack/react-table"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { cn } from "@/lib/utils"

type DataTableProps<T> = {
  table: TanstackTable<T>
  onRowClick?: (row: T) => void
  emptyMessage?: string
  className?: string
  dense?: boolean
  display?: boolean
  relaxed?: boolean
  stickyHeader?: boolean
}

export function DataTable<T>({
  table,
  onRowClick,
  emptyMessage = "Ma'lumot topilmadi",
  className,
  dense,
  display,
  relaxed,
  stickyHeader = false,
}: DataTableProps<T>) {
  const rows = table.getRowModel().rows

  return (
    <div
      className={cn(
        "rounded-xl border border-border/60 bg-background/50",
        className,
      )}
    >
      <Table>
        <TableHeader
          className={cn(
            stickyHeader && "sticky top-0 z-10 bg-muted/90 backdrop-blur",
          )}
        >
          {table.getHeaderGroups().map((hg) => (
            <TableRow
              key={hg.id}
              className="border-b border-border/60 bg-muted/40 hover:bg-muted/40"
            >
              {hg.headers.map((header) => (
                <TableHead
                  key={header.id}
                  onClick={header.column.getToggleSortingHandler()}
                  className={cn(
                    "bg-muted/90 font-semibold text-foreground/80",
                    stickyHeader && "backdrop-blur",
                    header.column.getCanSort() && "cursor-pointer select-none",
                    dense ? "h-10 px-3 text-xs" : relaxed ? "h-12 px-4 text-sm" : "h-11 px-4 text-sm",
                    display && "text-base",
                  )}
                >
                  <span className="inline-flex items-center gap-1">
                    {flexRender(header.column.columnDef.header, header.getContext())}
                    {{
                      asc: " ↑",
                      desc: " ↓",
                    }[header.column.getIsSorted() as string] ?? null}
                  </span>
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {rows.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={table.getAllColumns().length}
                className="h-24 text-center text-muted-foreground"
              >
                {emptyMessage}
              </TableCell>
            </TableRow>
          ) : (
            rows.map((row, index) => (
              <TableRow
                key={row.id}
                onClick={onRowClick ? () => onRowClick(row.original) : undefined}
                className={cn(
                  "border-border/40 transition-colors",
                  index % 2 === 0 ? "bg-transparent" : "bg-muted/20",
                  onRowClick && "cursor-pointer hover:bg-primary/5",
                )}
              >
                {row.getVisibleCells().map((cell) => (
                  <TableCell
                    key={cell.id}
                    className={cn(
                      dense ? "px-3 py-2 text-xs" : relaxed ? "px-4 py-4 text-sm" : "px-4 py-3 text-sm",
                      display && "py-4 text-lg font-medium",
                    )}
                  >
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  )
}
