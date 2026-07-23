import { useRef } from "react"
import { useVirtualizer, type VirtualItem } from "@tanstack/react-virtual"

export const PANEL_TABLE_ROW_HEIGHT = 64

export function usePanelTableVirtualizer(
  rowCount: number,
  estimateSize = PANEL_TABLE_ROW_HEIGHT,
  overscan = 2,
) {
  const scrollRef = useRef<HTMLDivElement>(null)

  const virtualizer = useVirtualizer({
    count: rowCount,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => estimateSize,
    overscan,
    getItemKey: (index) => index,
  })

  return { scrollRef, virtualizer }
}

export function getPanelVirtualRowStyle(virtualRow: VirtualItem) {
  return {
    transform: `translateY(${virtualRow.start}px)`,
    height: `${virtualRow.size}px`,
  }
}
