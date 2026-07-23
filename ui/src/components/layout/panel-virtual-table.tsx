import { flexRender, type Row, type Table as ReactTable } from "@tanstack/react-table"
import {
  VirtualTableCell,
  VirtualTableHeader,
  VirtualTableHeaderCell,
  VirtualTableRow,
} from "@/components/layout/virtual-table"
import {
  getPanelVirtualRowStyle,
  PANEL_TABLE_ROW_HEIGHT,
  usePanelTableVirtualizer,
} from "@/hooks/use-panel-table-virtualizer"
import { LAST_RECORDS_TABLE_VIEWPORT_CLASS } from "@/lib/last-records"
import { cn } from "@/lib/utils"

type PanelVirtualTableProps<T> = {
  table: ReactTable<T>
  gridTemplateColumns: string
  emptyMessage: string
  rowHeight?: number
  overscan?: number
  viewportClassName?: string
  onRowClick?: (row: Row<T>) => void
  getRowClassName?: (row: Row<T>, index: number) => string | undefined
}

export function PanelVirtualTable<T>({
  table,
  gridTemplateColumns,
  emptyMessage,
  rowHeight = PANEL_TABLE_ROW_HEIGHT,
  overscan = 8,
  viewportClassName,
  onRowClick,
  getRowClassName,
}: PanelVirtualTableProps<T>) {
  const rows = table.getRowModel().rows
  const { scrollRef, virtualizer } = usePanelTableVirtualizer(rows.length, rowHeight, overscan)
  const headers = table.getHeaderGroups()[0]?.headers ?? []

  return (
    <div
      ref={scrollRef}
      className={cn(
        LAST_RECORDS_TABLE_VIEWPORT_CLASS,
        "shrink-0 overflow-x-auto overflow-y-auto overscroll-y-contain rounded-b-xl",
        viewportClassName,
      )}
    >
      <VirtualTableHeader
        gridTemplateColumns={gridTemplateColumns}
        className="sticky top-0 z-10 border-border/60 bg-muted/95 backdrop-blur-sm"
      >
        {headers.map((header) => (
          <VirtualTableHeaderCell
            key={header.id}
            className="px-3 text-sm"
            onClick={header.column.getToggleSortingHandler()}
          >
            {flexRender(header.column.columnDef.header, header.getContext())}
            {{ asc: " ↑", desc: " ↓" }[header.column.getIsSorted() as string] ?? null}
          </VirtualTableHeaderCell>
        ))}
      </VirtualTableHeader>

      {rows.length === 0 ? (
        <div className="flex h-24 items-center justify-center text-sm text-muted-foreground">{emptyMessage}</div>
      ) : (
        <div className="relative w-full" style={{ height: virtualizer.getTotalSize() }}>
          {virtualizer.getVirtualItems().map((virtualRow) => {
            const row = rows[virtualRow.index]
            if (!row) {
              return null
            }

            return (
              <VirtualTableRow
                key={row.id}
                gridTemplateColumns={gridTemplateColumns}
                className={cn(
                  "items-center hover:bg-muted/50",
                  getRowClassName?.(row, virtualRow.index),
                )}
                style={getPanelVirtualRowStyle(virtualRow)}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
              >
                {row.getVisibleCells().map((cell) => (
                  <VirtualTableCell key={cell.id} className="px-3 py-2.5 text-xs sm:text-sm">
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </VirtualTableCell>
                ))}
              </VirtualTableRow>
            )
          })}
        </div>
      )}
    </div>
  )
}
