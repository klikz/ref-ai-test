import { useState, type ReactNode } from "react"
import { ChevronDown, Layers } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const GROUPED_STORAGE_KEY = "label-editor-properties-grouped"

export function readPropertiesGroupedPreference(): boolean {
  try {
    const raw = sessionStorage.getItem(GROUPED_STORAGE_KEY)
    if (raw === "0") {
      return false
    }
    if (raw === "1") {
      return true
    }
  } catch {
    // ignore
  }
  return true
}

export function writePropertiesGroupedPreference(grouped: boolean) {
  try {
    sessionStorage.setItem(GROUPED_STORAGE_KEY, grouped ? "1" : "0")
  } catch {
    // ignore
  }
}

export function PropertiesGroupToggle({
  grouped,
  onChange,
}: {
  grouped: boolean
  onChange: (grouped: boolean) => void
}) {
  return (
    <Button
      type="button"
      variant={grouped ? "default" : "outline"}
      size="sm"
      className="h-7 gap-1.5 px-2 text-xs"
      title="Parametrlarni bo'limlarga guruhlash"
      onClick={() => onChange(!grouped)}
    >
      <Layers className="size-3.5" />
      Guruhlash
    </Button>
  )
}

export function PropertiesSection({
  title,
  grouped,
  defaultOpen = true,
  children,
}: {
  title: string
  grouped: boolean
  defaultOpen?: boolean
  children: ReactNode
}) {
  const [open, setOpen] = useState(defaultOpen)

  if (!grouped) {
    return <div className="space-y-2">{children}</div>
  }

  return (
    <div className="overflow-hidden rounded-md border bg-muted/10">
      <button
        type="button"
        className="flex w-full items-center justify-between gap-2 px-2.5 py-2 text-left text-xs font-semibold text-muted-foreground hover:bg-muted/30"
        onClick={() => setOpen((prev) => !prev)}
      >
        <span>{title}</span>
        <ChevronDown className={cn("size-4 shrink-0 transition-transform", open && "rotate-180")} />
      </button>
      {open ? <div className="space-y-2 border-t px-2.5 pb-2.5 pt-2">{children}</div> : null}
    </div>
  )
}
