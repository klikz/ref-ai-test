import { useLayoutEffect, useRef, useState } from "react"
import { useWindowVirtualizer, type VirtualItem } from "@tanstack/react-virtual"

export const TABLE_HEADER_HEIGHT = 45

export function getVirtualRowStyle(virtualRow: VirtualItem, scrollMargin: number) {
  return {
    transform: `translateY(${virtualRow.start - scrollMargin}px)`,
    height: `${virtualRow.size}px`,
  }
}

export function useWindowTableVirtualizer(rowCount: number, estimateSize = 45) {
  const listRef = useRef<HTMLDivElement>(null)
  const [scrollMargin, setScrollMargin] = useState(0)

  useLayoutEffect(() => {
    const element = listRef.current
    if (!element) {
      return
    }

    const updateScrollMargin = () => {
      const next = element.offsetTop
      setScrollMargin((current) => (current === next ? current : next))
    }

    updateScrollMargin()
    const frame = window.requestAnimationFrame(updateScrollMargin)
    const observer = new ResizeObserver(updateScrollMargin)
    observer.observe(element)
    if (element.parentElement) {
      observer.observe(element.parentElement)
    }
    window.addEventListener("resize", updateScrollMargin)
    return () => {
      window.cancelAnimationFrame(frame)
      observer.disconnect()
      window.removeEventListener("resize", updateScrollMargin)
    }
  }, [rowCount])

  const virtualizer = useWindowVirtualizer({
    count: rowCount,
    estimateSize: () => estimateSize,
    overscan: 10,
    scrollMargin,
  })

  return { listRef, virtualizer, scrollMargin, tableHeaderHeight: TABLE_HEADER_HEIGHT }
}
